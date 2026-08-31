package gateway

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	"github.com/hllkk/devops-admin/server/model/system"
)

// CostAnalysisService 成本分析(P3)：读聚合表 cost_summary_daily 做多维聚合，
// 部门维度读时归因(不写聚合表，人员调岗不污染历史成本)。
// 对齐 AIHelms CostView 的 KPI/趋势/多维明细/部门下钻结构，规避其坑：
// 明细服务端分页(非全量+静默截断)、LEFT JOIN 不丢未归因行、部门行与下钻同直挂口径(行=子和)、
// 模型维度直接按聚合表 model 分组(不做模糊反查)。
type CostAnalysisService struct{}

// 成本分析维度(dashboard.go 已有 dimensionUser/dimensionModel/dimensionAiKey，此处补齐)
const (
	costDimensionDepartment = "department"
	costDimensionProvider   = "provider"
	costDimensionDate       = "date"
)

// costDeptJoins 部门读时归因 JOIN(表别名 s = gateway_cost_summary_daily)。
// 软删 Key 仍按其历史归属归因(成本是历史事实)，故不筛 deleted_at。
const costDeptJoins = `LEFT JOIN sys_users u ON u.id = s.user_id AND s.user_id <> 0
	LEFT JOIN gateway_ai_key k ON k.ai_key_id = s.ai_key_id AND s.ai_key_id <> 0`

// costDeptAnchor 部门锚点：部门Key(owner_type=dept)的消耗归部门，个人Key归个人主部门
// (sys_users.dept_id)；两者皆无为 0(未分配)。部门列表行与下钻成员同用此直挂口径。
const costDeptAnchor = `CASE WHEN k.owner_type = 'dept' THEN k.owner_id ELSE COALESCE(u.dept_id, 0) END`

// GetCostOverview KPI(含等长上一期环比)+按日趋势，随筛选联动。
func (s *CostAnalysisService) GetCostOverview(ctx context.Context, f *gatewayReq.CostAnalysisSearch) (gatewayResp.CostOverview, error) {
	now := time.Now()
	start, end, days := normalizeCostRange(f.StartDate, f.EndDate, now)
	var ov gatewayResp.CostOverview

	curInt, curExt, curReq, curTok, err := s.sumCost(ctx, f, start, end)
	if err != nil {
		return ov, err
	}
	prevStart, prevEnd := prevCostRange(start, end)
	prevInt, prevExt, _, _, err := s.sumCost(ctx, f, prevStart, prevEnd)
	if err != nil {
		return ov, err
	}

	kpi := gatewayResp.CostKpi{
		InternalCost: curInt, ExternalCost: curExt, CostDiff: curExt - curInt,
		InternalChange: costChange(curInt, prevInt), ExternalChange: costChange(curExt, prevExt),
		TotalRequests: curReq, TotalTokens: curTok, Days: days,
	}
	if days > 0 {
		kpi.DailyAvgInternal = curInt / float64(days)
	}
	ov.Kpi = kpi

	db := s.costBaseQuery(ctx, f, start, end)
	var trend []gatewayResp.CostTrendItem
	if err := db.Select(`to_char(s.summary_date,'YYYY-MM-DD') AS date,
		COALESCE(SUM(s.internal_cost),0) AS internal_cost,
		COALESCE(SUM(s.external_cost),0) AS external_cost,
		COALESCE(SUM(s.request_count),0) AS requests,
		COALESCE(SUM(s.total_tokens),0) AS tokens`).
		Group("s.summary_date").Order("s.summary_date ASC").Scan(&trend).Error; err != nil {
		return ov, err
	}
	ov.Trend = trend
	return ov, nil
}

// GetCostDetail 多维聚合明细(服务端分页，排序白名单降序)。
// 维度六选一：department/user/model/aiKey/provider/date。
func (s *CostAnalysisService) GetCostDetail(ctx context.Context, f *gatewayReq.CostAnalysisSearch) ([]gatewayResp.CostDetailRow, int64, error) {
	now := time.Now()
	start, end, _ := normalizeCostRange(f.StartDate, f.EndDate, now)
	dimension := costDimensionOf(f.Dimension)
	groupExpr, valueExpr := costGroupExpr(dimension)

	db := s.costBaseQuery(ctx, f, start, end)

	// 组数(分页 total)：COUNT(DISTINCT 分组表达式)
	var total int64
	if err := db.Session(&gorm.Session{}).Select(fmt.Sprintf("COUNT(DISTINCT %s)", groupExpr)).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	type costDetailAgg struct {
		Value            string
		Requests         int
		PromptTokens     int64
		CompletionTokens int64
		TotalTokens      int64
		InternalCost     float64
		ExternalCost     float64
		ActiveUsers      int
	}
	var rows []costDetailAgg
	limit, offset := f.LimitOffset()
	if err := db.Select(fmt.Sprintf(`%s AS value,
		COALESCE(SUM(s.request_count),0) AS requests,
		COALESCE(SUM(s.prompt_tokens),0) AS prompt_tokens,
		COALESCE(SUM(s.completion_tokens),0) AS completion_tokens,
		COALESCE(SUM(s.total_tokens),0) AS total_tokens,
		COALESCE(SUM(s.internal_cost),0) AS internal_cost,
		COALESCE(SUM(s.external_cost),0) AS external_cost,
		COUNT(DISTINCT s.user_id) AS active_users`, valueExpr)).
		Group(groupExpr).Order(costSortColumn(f.Sort) + " DESC").Limit(limit).Offset(offset).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]gatewayResp.CostDetailRow, 0, len(rows))
	for _, r := range rows {
		row := gatewayResp.CostDetailRow{
			Value: r.Value, Requests: r.Requests,
			PromptTokens: r.PromptTokens, CompletionTokens: r.CompletionTokens, TotalTokens: r.TotalTokens,
			InternalCost: r.InternalCost, ExternalCost: r.ExternalCost, CostDiff: r.ExternalCost - r.InternalCost,
			ActiveUsers: r.ActiveUsers,
		}
		if r.ActiveUsers > 0 {
			row.PerCapita = r.InternalCost / float64(r.ActiveUsers)
		}
		items = append(items, row)
	}
	s.fillDetailLabels(ctx, dimension, items)
	return items, total, nil
}

// GetCostScopeUsers 部门下钻：直挂口径(部门锚点=该部门)按用户分组，保证「部门行=子和」。
// UserId=0 为该部门「部门Key消耗/未归因」合并行；不叠部门子树过滤(锚点已限定单部门)。
func (s *CostAnalysisService) GetCostScopeUsers(ctx context.Context, deptId int64, f *gatewayReq.CostAnalysisSearch) ([]gatewayResp.CostScopeUserRow, error) {
	now := time.Now()
	start, end, _ := normalizeCostRange(f.StartDate, f.EndDate, now)
	if deptId == 0 {
		return []gatewayResp.CostScopeUserRow{}, nil
	}
	db := s.costBaseQuery(ctx, f, start, end).Where(costDeptAnchor+" = ?", deptId)
	type scopeAgg struct {
		UserId       string
		Requests     int
		TotalTokens  int64
		InternalCost float64
		ExternalCost float64
	}
	var rows []scopeAgg
	if err := db.Select(`CAST(s.user_id AS TEXT) AS user_id,
		COALESCE(SUM(s.request_count),0) AS requests,
		COALESCE(SUM(s.total_tokens),0) AS total_tokens,
		COALESCE(SUM(s.internal_cost),0) AS internal_cost,
		COALESCE(SUM(s.external_cost),0) AS external_cost`).
		Group("s.user_id").Order("internal_cost DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}

	userIds := make([]int64, 0, len(rows))
	for _, r := range rows {
		if r.UserId != "0" && r.UserId != "" {
			var id int64
			if _, err := fmt.Sscanf(r.UserId, "%d", &id); err == nil && id != 0 {
				userIds = append(userIds, id)
			}
		}
	}
	names := map[int64]string{}
	if len(userIds) > 0 {
		var users []system.SysUser
		if err := global.OPS_DB.WithContext(ctx).Select("id, nick_name").Where("id IN ?", userIds).Find(&users).Error; err == nil {
			for i := range users {
				names[users[i].UserId] = users[i].NickName
			}
		}
	}
	items := make([]gatewayResp.CostScopeUserRow, 0, len(rows))
	for _, r := range rows {
		item := gatewayResp.CostScopeUserRow{
			Requests: r.Requests, TotalTokens: r.TotalTokens,
			InternalCost: r.InternalCost, ExternalCost: r.ExternalCost,
		}
		var id int64
		if _, err := fmt.Sscanf(r.UserId, "%d", &id); err == nil {
			item.UserId = id
		}
		if item.UserId == 0 {
			item.UserName = "部门Key/未归因"
		} else if n := names[item.UserId]; n != "" {
			item.UserName = n
		} else {
			item.UserName = "用户:" + r.UserId
		}
		items = append(items, item)
	}
	return items, nil
}

// costBaseQuery 聚合表 + 部门归因 JOIN + 公共筛选(时间/部门子树/用户/Key/模型/供应商)。
func (s *CostAnalysisService) costBaseQuery(ctx context.Context, f *gatewayReq.CostAnalysisSearch, start, end string) *gorm.DB {
	db := global.OPS_DB.WithContext(ctx).Table("gateway_cost_summary_daily s").Joins(costDeptJoins)
	return applyCostFilter(db, f, start, end, expandDeptSubtree(ctx, f.DepartmentId))
}

// sumCost 期间成本/请求/token 合计(KPI 与环比共用)。
func (s *CostAnalysisService) sumCost(ctx context.Context, f *gatewayReq.CostAnalysisSearch, start, end string) (internal, external float64, requests int, tokens int64, err error) {
	type agg struct {
		Internal float64
		External float64
		Requests int
		Tokens   int64
	}
	var a agg
	if err = s.costBaseQuery(ctx, f, start, end).
		Select(`COALESCE(SUM(s.internal_cost),0) AS internal,
			COALESCE(SUM(s.external_cost),0) AS external,
			COALESCE(SUM(s.request_count),0) AS requests,
			COALESCE(SUM(s.total_tokens),0) AS tokens`).Scan(&a).Error; err != nil {
		return
	}
	return a.Internal, a.External, a.Requests, a.Tokens, nil
}

// applyCostFilter 公共筛选下推。部门筛选展开子树后下推到部门锚点表达式
// (需 costDeptJoins 在场)；时间为业务日闭区间。
func applyCostFilter(db *gorm.DB, f *gatewayReq.CostAnalysisSearch, start, end string, deptIds []int64) *gorm.DB {
	db = db.Where("s.summary_date >= ? AND s.summary_date <= ?", start, end)
	if len(deptIds) > 0 {
		db = db.Where(costDeptAnchor+" IN ?", deptIds)
	}
	if f.UserId != 0 {
		db = db.Where("s.user_id = ?", f.UserId)
	}
	if f.AiKeyId != 0 {
		db = db.Where("s.ai_key_id = ?", f.AiKeyId)
	}
	if f.Model != "" {
		db = db.Where("s.model = ?", f.Model)
	}
	if f.Provider != "" {
		db = db.Where("s.provider = ?", f.Provider)
	}
	return db
}

// costDimensionOf 维度白名单(默认 department)。
func costDimensionOf(dim string) string {
	switch dim {
	case dimensionUser, dimensionModel, dimensionAiKey, costDimensionProvider, costDimensionDate:
		return dim
	}
	return costDimensionDepartment
}

// costGroupExpr 维度 → (分组表达式, 取值表达式)。取值表达式负责 ID→TEXT 与空值兜底。
func costGroupExpr(dimension string) (groupExpr, valueExpr string) {
	switch dimension {
	case dimensionUser:
		return "s.user_id", "CAST(s.user_id AS TEXT)"
	case dimensionModel:
		return "s.model", "COALESCE(NULLIF(s.model,''),'未知')"
	case dimensionAiKey:
		return "s.ai_key_id", "CAST(s.ai_key_id AS TEXT)"
	case costDimensionProvider:
		return "s.provider", "COALESCE(NULLIF(s.provider,''),'未知')"
	case costDimensionDate:
		return "s.summary_date", "to_char(s.summary_date,'YYYY-MM-DD')"
	default: // department
		return costDeptAnchor, "CAST(" + costDeptAnchor + " AS TEXT)"
	}
}

// costSortColumn 明细排序聚合列白名单(外层引用 SELECT 别名，防注入)，默认内部成本。
func costSortColumn(sort string) string {
	switch sort {
	case "external":
		return "external_cost"
	case "requests":
		return "requests"
	case "tokens":
		return "total_tokens"
	}
	return "internal_cost"
}

// normalizeCostRange 归一化查询期间：非法/缺省回退本月；start>end 交换；返回闭区间业务日与天数。
func normalizeCostRange(startStr, endStr string, now time.Time) (start, end string, days int) {
	const layout = "2006-01-02"
	end = now.Format(layout)
	start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format(layout)
	if v, err := time.ParseInLocation(layout, startStr, now.Location()); err == nil {
		start = v.Format(layout)
	}
	if v, err := time.ParseInLocation(layout, endStr, now.Location()); err == nil {
		end = v.Format(layout)
	}
	if start > end {
		start, end = end, start
	}
	st, _ := time.ParseInLocation(layout, start, now.Location())
	et, _ := time.ParseInLocation(layout, end, now.Location())
	days = int(et.Sub(st).Hours()/24) + 1
	return start, end, days
}

// prevCostRange 等长上一期(紧贴本期之前)，环比口径与 AIHelms 一致。
func prevCostRange(start, end string) (string, string) {
	const layout = "2006-01-02"
	st, _ := time.ParseInLocation(layout, start, time.Local)
	et, _ := time.ParseInLocation(layout, end, time.Local)
	days := int(et.Sub(st).Hours()/24) + 1
	prevEnd := st.AddDate(0, 0, -1)
	prevStart := prevEnd.AddDate(0, 0, -(days - 1))
	return prevStart.Format(layout), prevEnd.Format(layout)
}

// costChange 环比%：上期为 0 时给 0(前端可结合本期值显示"新增")。
func costChange(cur, prev float64) float64 {
	if prev == 0 {
		return 0
	}
	return (cur - prev) / prev * 100
}

// expandDeptSubtree 部门子树展开(含自身；内存 BFS，部门表量小)。
// 不复用 system.DataScopeService.ExpandDeptIDs：service/system 已引 service/gateway
// (用户生命周期级联)，反向引用会成环。
func expandDeptSubtree(ctx context.Context, deptId int64) []int64 {
	if deptId == 0 {
		return nil
	}
	var depts []system.SysDepartment
	global.OPS_DB.WithContext(ctx).Select("dept_id, parent_id").Find(&depts)
	children := make(map[int64][]int64, len(depts))
	for _, d := range depts {
		children[d.ParentId] = append(children[d.ParentId], d.DeptId)
	}
	out, queue := []int64{deptId}, []int64{deptId}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, c := range children[cur] {
			out = append(out, c)
			queue = append(queue, c)
		}
	}
	return out
}

// fillDetailLabels 维度 ID → 可读名批量回填(每页一次 IN 查询，防 N+1)。
func (s *CostAnalysisService) fillDetailLabels(ctx context.Context, dimension string, rows []gatewayResp.CostDetailRow) {
	if len(rows) == 0 {
		return
	}
	db := global.OPS_DB.WithContext(ctx)
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		var id int64
		if _, err := fmt.Sscanf(r.Value, "%d", &id); err == nil && id != 0 {
			ids = append(ids, id)
		}
	}
	switch dimension {
	case costDimensionDepartment:
		names := map[int64]string{}
		if len(ids) > 0 {
			var depts []system.SysDepartment
			if err := db.Select("dept_id, dept_name").Where("dept_id IN ?", ids).Find(&depts).Error; err == nil {
				for i := range depts {
					names[depts[i].DeptId] = depts[i].DeptName
				}
			}
		}
		for i := range rows {
			var id int64
			_, _ = fmt.Sscanf(rows[i].Value, "%d", &id)
			if id == 0 {
				rows[i].Label = "未分配"
			} else if n := names[id]; n != "" {
				rows[i].Label = n
			} else {
				rows[i].Label = "部门:" + rows[i].Value
			}
		}
	case dimensionUser:
		names := map[int64]string{}
		if len(ids) > 0 {
			var users []system.SysUser
			if err := db.Select("id, nick_name").Where("id IN ?", ids).Find(&users).Error; err == nil {
				for i := range users {
					names[users[i].UserId] = users[i].NickName
				}
			}
		}
		for i := range rows {
			var id int64
			_, _ = fmt.Sscanf(rows[i].Value, "%d", &id)
			if id == 0 {
				rows[i].Label = "未归因"
			} else if n := names[id]; n != "" {
				rows[i].Label = n
			} else {
				rows[i].Label = "用户:" + rows[i].Value
			}
		}
	case dimensionAiKey:
		names := map[int64]string{}
		if len(ids) > 0 {
			var keys []gateway.AiKey
			if err := db.Select("ai_key_id, name").Where("ai_key_id IN ?", ids).Find(&keys).Error; err == nil {
				for i := range keys {
					names[keys[i].AiKeyId] = keys[i].Name
				}
			}
		}
		for i := range rows {
			var id int64
			_, _ = fmt.Sscanf(rows[i].Value, "%d", &id)
			if id == 0 {
				rows[i].Label = "未归因"
			} else if n := names[id]; n != "" {
				rows[i].Label = n
			} else {
				rows[i].Label = "Key:" + rows[i].Value
			}
		}
	default:
		// model/provider/date 维度取值表达式已是可读名
		for i := range rows {
			rows[i].Label = rows[i].Value
		}
	}
}
