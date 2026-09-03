package gateway

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
)

// AdoptionService 覆盖率/采用度(P3)。口径集中定义(规避 AIHelms 同一概念散落三处互相矛盾的坑)：
//   - 激活用户：期内有任意 LLM(聚合表)或 MCP(日志表)调用且 user_id<>0 的去重用户；
//     Skill/纯平台资源调用不计入激活。
//   - 分母(启用用户)：sys_users.status='0' 且未软删，含从未使用 AI 的用户(决策层全员视角)。
//   - 部门归属：直挂口径(同成本分析 costDeptAnchor——部门Key归部门/个人Key归个人主部门)；
//     成员分母按 sys_users.dept_id 单值直挂计数，天然规避 AIHelms 多部门 JOIN 重复计数坑。
//   - 人均 token：分母=激活用户数(活跃人均)，前端 tooltip 注明口径。
//   - 覆盖率环比：当期-上期百分点差；分母均为当前用户快照(历史分母不可还原，注释明示)。
type AdoptionService struct{}

// GetAdoptionOverview KPI(覆盖率/新增活跃/日均调用/人均 token)+DAU 按日趋势。
func (s *AdoptionService) GetAdoptionOverview(ctx context.Context, f *gatewayReq.AdoptionSearch) (gatewayResp.AdoptionOverview, error) {
	now := time.Now()
	start, end, days := normalizeCostRange(f.StartDate, f.EndDate, now)
	var ov gatewayResp.AdoptionOverview

	userDept, err := s.enabledUserDepts(ctx, expandDeptSubtree(ctx, f.DepartmentId))
	if err != nil {
		return ov, err
	}
	curActive, err := s.activeUserSet(ctx, f, start, end)
	if err != nil {
		return ov, err
	}
	prevStart, prevEnd := prevCostRange(start, end)
	prevActive, err := s.activeUserSet(ctx, f, prevStart, prevEnd)
	if err != nil {
		return ov, err
	}
	newActive := 0
	for id := range curActive {
		if _, ok := prevActive[id]; !ok {
			newActive++
		}
	}

	// 调用数/token(LLM+MCP 合并口径，与成本分析 overview 一致)
	ca := CostAnalysisService{}
	_, _, curReq, curTok, err := ca.sumCost(ctx, f, start, end)
	if err != nil {
		return ov, err
	}
	_, _, mCalls, err := sumMcpCost(ctx, f, start, end)
	if err != nil {
		return ov, err
	}
	curReq += mCalls

	total := len(userDept)
	active := len(curActive)
	kpi := gatewayResp.AdoptionKpi{
		TotalUsers: total, ActiveUsers: active,
		Coverage:       adoptionPct(active, total),
		CoverageChange: adoptionPct(active, total) - adoptionPct(len(prevActive), total),
		NewActiveUsers: newActive, PrevActiveUsers: len(prevActive),
		TotalRequests: curReq, Days: days,
	}
	if days > 0 {
		kpi.DailyRequests = float64(curReq) / float64(days)
	}
	if active > 0 {
		kpi.PerCapitaTokens = curTok / int64(active)
	}
	ov.Kpi = kpi

	trend, err := s.dauTrend(ctx, f, start, end)
	if err != nil {
		return ov, err
	}
	ov.Trend = trend
	return ov, nil
}

// GetAdoptionDepartments 部门覆盖率明细：全部部门行(含零调用部门,覆盖率视角)，
// 消耗为 LLM+MCP 合并、按部门锚点聚合；激活按成员直挂部门计数。
func (s *AdoptionService) GetAdoptionDepartments(ctx context.Context, f *gatewayReq.AdoptionSearch) ([]gatewayResp.AdoptionDeptRow, error) {
	now := time.Now()
	start, end, _ := normalizeCostRange(f.StartDate, f.EndDate, now)
	deptIds := expandDeptSubtree(ctx, f.DepartmentId)

	userDept, err := s.enabledUserDepts(ctx, deptIds)
	if err != nil {
		return nil, err
	}
	active, err := s.activeUserSet(ctx, f, start, end)
	if err != nil {
		return nil, err
	}
	memberCount, activeCount := map[int64]int{}, map[int64]int{}
	for uid, dept := range userDept {
		memberCount[dept]++
		if _, ok := active[uid]; ok {
			activeCount[dept]++
		}
	}

	// 消耗按部门锚点(LLM 聚合表 + MCP 日志表 Go 层合并)
	type usageAgg struct {
		DeptId       int64
		Requests     int
		TotalTokens  int64
		InternalCost float64
	}
	usage := map[int64]*gatewayResp.AdoptionDeptRow{}
	acc := func(rows []usageAgg, withTokens bool) {
		for _, r := range rows {
			v := usage[r.DeptId]
			if v == nil {
				v = &gatewayResp.AdoptionDeptRow{}
				usage[r.DeptId] = v
			}
			v.Requests += r.Requests
			v.InternalCost += r.InternalCost
			if withTokens {
				v.TotalTokens += r.TotalTokens
			}
		}
	}
	var llmRows []usageAgg
	llmDb := global.OPS_DB.WithContext(ctx).Table("gateway_cost_summary_daily s").Joins(costDeptJoins)
	llmDb = applyCostFilter(llmDb, f, start, end, deptIds)
	if err := llmDb.Select(fmt.Sprintf(`CAST(%s AS BIGINT) AS dept_id,
		COALESCE(SUM(s.request_count),0) AS requests,
		COALESCE(SUM(s.total_tokens),0) AS total_tokens,
		COALESCE(SUM(s.internal_cost),0) AS internal_cost`, costDeptAnchor)).
		Group(costDeptAnchor).Scan(&llmRows).Error; err != nil {
		return nil, err
	}
	acc(llmRows, true)

	var mcpRows []usageAgg
	if err := mcpCostBaseQuery(ctx, f, start, end, deptIds).
		Select(fmt.Sprintf(`CAST(%s AS BIGINT) AS dept_id,
			COUNT(*) AS requests,
			COALESCE(SUM(m.internal_cost),0) AS internal_cost`, costMcpDeptAnchor)).
		Group(costMcpDeptAnchor).Scan(&mcpRows).Error; err != nil {
		return nil, err
	}
	acc(mcpRows, false)

	// 部门列表(树筛联动；全量展示含零调用部门)
	deptDb := global.OPS_DB.WithContext(ctx).Model(&system.SysDepartment{}).Select("dept_id, dept_name")
	if len(deptIds) > 0 {
		deptDb = deptDb.Where("dept_id IN ?", deptIds)
	}
	var depts []system.SysDepartment
	if err := deptDb.Order("order_num ASC, dept_id ASC").Find(&depts).Error; err != nil {
		return nil, err
	}

	rows := make([]gatewayResp.AdoptionDeptRow, 0, len(depts)+1)
	seen := map[int64]bool{}
	for i := range depts {
		id := depts[i].DeptId
		seen[id] = true
		row := gatewayResp.AdoptionDeptRow{DeptId: id, DeptName: depts[i].DeptName}
		if v := usage[id]; v != nil {
			row.Requests, row.TotalTokens, row.InternalCost = v.Requests, v.TotalTokens, v.InternalCost
		}
		row.MemberCount = memberCount[id]
		row.ActiveCount = activeCount[id]
		row.Coverage = adoptionPct(row.ActiveCount, row.MemberCount)
		rows = append(rows, row)
	}
	// 未挂部门的消耗/成员兜底行(锚点=0)，部门表已有 0 时不重复
	if memberCount[0] > 0 || usage[0] != nil {
		if !seen[0] {
			row := gatewayResp.AdoptionDeptRow{DeptId: 0, DeptName: "未分配",
				MemberCount: memberCount[0], ActiveCount: activeCount[0]}
			if v := usage[0]; v != nil {
				row.Requests, row.TotalTokens, row.InternalCost = v.Requests, v.TotalTokens, v.InternalCost
			}
			row.Coverage = adoptionPct(row.ActiveCount, row.MemberCount)
			rows = append(rows, row)
		}
	}
	// 排序：激活数降序 → 成员数降序(零调用部门沉底可预期)
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].ActiveCount != rows[j].ActiveCount {
			return rows[i].ActiveCount > rows[j].ActiveCount
		}
		return rows[i].MemberCount > rows[j].MemberCount
	})
	return rows, nil
}

// GetAdoptionDeptUsers 部门下钻成员明细：全部启用成员(含未激活，兼「未使用人员」清单)，
// 激活在前；消耗为该成员 LLM+MCP 合并(部门 Key 消耗 user_id=0 不属于任何成员，不出现在行中)。
func (s *AdoptionService) GetAdoptionDeptUsers(ctx context.Context, deptId int64, f *gatewayReq.AdoptionSearch) ([]gatewayResp.AdoptionUserRow, error) {
	now := time.Now()
	start, end, _ := normalizeCostRange(f.StartDate, f.EndDate, now)
	if deptId == 0 {
		return []gatewayResp.AdoptionUserRow{}, nil
	}

	var users []system.SysUser
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).
		Select("id, nick_name").Where("dept_id = ? AND status = ?", deptId, "0").
		Order("id ASC").Find(&users).Error; err != nil {
		return nil, err
	}

	type userAgg struct {
		UserId       string
		Requests     int
		TotalTokens  int64
		InternalCost float64
		LastActive   *time.Time
	}
	agg := map[int64]*userAgg{}

	var llmRows []userAgg
	llmDb := global.OPS_DB.WithContext(ctx).Table("gateway_cost_summary_daily s").Joins(costDeptJoins)
	llmDb = applyCostFilter(llmDb, f, start, end, nil).Where(costDeptAnchor+" = ?", deptId)
	if err := llmDb.Select(`CAST(s.user_id AS TEXT) AS user_id,
		COALESCE(SUM(s.request_count),0) AS requests,
		COALESCE(SUM(s.total_tokens),0) AS total_tokens,
		COALESCE(SUM(s.internal_cost),0) AS internal_cost,
		MAX(s.summary_date) AS last_active`).
		Group("s.user_id").Scan(&llmRows).Error; err != nil {
		return nil, err
	}
	for _, r := range llmRows {
		var id int64
		if _, err := fmt.Sscanf(r.UserId, "%d", &id); err != nil || id == 0 {
			continue
		}
		agg[id] = &userAgg{Requests: r.Requests, TotalTokens: r.TotalTokens, InternalCost: r.InternalCost, LastActive: r.LastActive}
	}

	var mcpRows []userAgg
	if err := mcpCostBaseQuery(ctx, f, start, end, nil).Where(costMcpDeptAnchor+" = ?", deptId).
		Select(`CAST(m.user_id AS TEXT) AS user_id,
			COUNT(*) AS requests,
			MAX(m.started_at) AS last_active`).
		Group("m.user_id").Scan(&mcpRows).Error; err != nil {
		return nil, err
	}
	for _, r := range mcpRows {
		var id int64
		if _, err := fmt.Sscanf(r.UserId, "%d", &id); err != nil || id == 0 {
			continue
		}
		v := agg[id]
		if v == nil {
			v = &userAgg{}
			agg[id] = v
		}
		v.Requests += r.Requests
		if r.LastActive != nil && (v.LastActive == nil || r.LastActive.After(*v.LastActive)) {
			v.LastActive = r.LastActive
		}
	}

	rows := make([]gatewayResp.AdoptionUserRow, 0, len(users))
	for i := range users {
		uid := users[i].UserId
		row := gatewayResp.AdoptionUserRow{UserId: uid, UserName: users[i].NickName}
		if v := agg[uid]; v != nil && v.Requests > 0 {
			row.Active = true
			row.Requests = v.Requests
			row.TotalTokens = v.TotalTokens
			row.InternalCost = v.InternalCost
			if v.LastActive != nil {
				row.LastActiveAt = v.LastActive.Local().Format("2006-01-02 15:04")
			}
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Active != rows[j].Active {
			return rows[i].Active
		}
		return rows[i].Requests > rows[j].Requests
	})
	return rows, nil
}

// GetAdoptionModels 模型分布(LLM 维;MCP 调用无模型概念)，调用数降序全量。
func (s *AdoptionService) GetAdoptionModels(ctx context.Context, f *gatewayReq.AdoptionSearch) ([]gatewayResp.AdoptionModelRow, error) {
	now := time.Now()
	start, end, _ := normalizeCostRange(f.StartDate, f.EndDate, now)

	type modelAgg struct {
		Model        string
		Requests     int
		TotalTokens  int64
		InternalCost float64
		ActiveUsers  int
	}
	var rows []modelAgg
	db := global.OPS_DB.WithContext(ctx).Table("gateway_cost_summary_daily s").Joins(costDeptJoins)
	db = applyCostFilter(db, f, start, end, expandDeptSubtree(ctx, f.DepartmentId))
	if err := db.Select(`COALESCE(NULLIF(s.model,''),'未知') AS model,
		COALESCE(SUM(s.request_count),0) AS requests,
		COALESCE(SUM(s.total_tokens),0) AS total_tokens,
		COALESCE(SUM(s.internal_cost),0) AS internal_cost,
		COUNT(DISTINCT s.user_id) AS active_users`).
		Group("COALESCE(NULLIF(s.model,''),'未知')").
		Order("requests DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}

	totalReq, totalCost := 0, 0.0
	for _, r := range rows {
		totalReq += r.Requests
		totalCost += r.InternalCost
	}
	items := make([]gatewayResp.AdoptionModelRow, 0, len(rows))
	for _, r := range rows {
		item := gatewayResp.AdoptionModelRow{
			Model: r.Model, Requests: r.Requests, TotalTokens: r.TotalTokens,
			InternalCost: r.InternalCost, ActiveUsers: r.ActiveUsers,
		}
		if totalReq > 0 {
			item.RequestShare = float64(r.Requests) / float64(totalReq) * 100
		}
		if totalCost > 0 {
			item.CostShare = r.InternalCost / totalCost * 100
		}
		items = append(items, item)
	}
	return items, nil
}

// ───────────────── 内部辅助 ─────────────────

// enabledUserDepts 启用用户 id→主部门(分母与部门归因共用；部门树筛选联动)。
func (s *AdoptionService) enabledUserDepts(ctx context.Context, deptIds []int64) (map[int64]int64, error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).
		Select("id, dept_id").Where("status = ?", "0")
	if len(deptIds) > 0 {
		db = db.Where("dept_id IN ?", deptIds)
	}
	type row struct {
		Id     int64
		DeptId int64
	}
	var rows []row
	if err := db.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[int64]int64, len(rows))
	for _, r := range rows {
		out[r.Id] = r.DeptId
	}
	return out, nil
}

// activeUserSet 期内激活用户集合(LLM 聚合表 ∪ MCP 日志表，user_id<>0)。
func (s *AdoptionService) activeUserSet(ctx context.Context, f *gatewayReq.AdoptionSearch, start, end string) (map[int64]struct{}, error) {
	deptIds := expandDeptSubtree(ctx, f.DepartmentId)
	out := map[int64]struct{}{}

	var llmIds []int64
	llmDb := global.OPS_DB.WithContext(ctx).Table("gateway_cost_summary_daily s").Joins(costDeptJoins)
	llmDb = applyCostFilter(llmDb, f, start, end, deptIds)
	if err := llmDb.Select("DISTINCT s.user_id").Where("s.user_id <> 0").Scan(&llmIds).Error; err != nil {
		return nil, err
	}
	for _, id := range llmIds {
		out[id] = struct{}{}
	}

	var mcpIds []int64
	if err := mcpCostBaseQuery(ctx, f, start, end, deptIds).
		Select("DISTINCT m.user_id").Where("m.user_id <> 0").Scan(&mcpIds).Error; err != nil {
		return nil, err
	}
	for _, id := range mcpIds {
		out[id] = struct{}{}
	}
	return out, nil
}

// dauTrend DAU 按日趋势：(业务日,用户) 粒度取 LLM∪MCP 去重后按日计数；
// 调用数按日为两源之和(与 KPI 口径一致)。
func (s *AdoptionService) dauTrend(ctx context.Context, f *gatewayReq.AdoptionSearch, start, end string) ([]gatewayResp.AdoptionTrendItem, error) {
	deptIds := expandDeptSubtree(ctx, f.DepartmentId)

	type dauPair struct {
		Date   string
		UserId int64
	}
	pairSet := map[string]map[int64]struct{}{}
	addPairs := func(rows []dauPair) {
		for _, r := range rows {
			if r.UserId == 0 {
				continue
			}
			if pairSet[r.Date] == nil {
				pairSet[r.Date] = map[int64]struct{}{}
			}
			pairSet[r.Date][r.UserId] = struct{}{}
		}
	}

	var llmPairs []dauPair
	llmDb := global.OPS_DB.WithContext(ctx).Table("gateway_cost_summary_daily s").Joins(costDeptJoins)
	llmDb = applyCostFilter(llmDb, f, start, end, deptIds)
	if err := llmDb.Select(`to_char(s.summary_date,'YYYY-MM-DD') AS date, s.user_id AS user_id`).
		Where("s.user_id <> 0").Group("s.summary_date, s.user_id").Scan(&llmPairs).Error; err != nil {
		return nil, err
	}
	addPairs(llmPairs)

	var mcpPairs []dauPair
	if err := mcpCostBaseQuery(ctx, f, start, end, deptIds).
		Select(`to_char((m.started_at AT TIME ZONE 'Asia/Shanghai')::date,'YYYY-MM-DD') AS date, m.user_id AS user_id`).
		Where("m.user_id <> 0").
		Group(`(m.started_at AT TIME ZONE 'Asia/Shanghai')::date, m.user_id`).Scan(&mcpPairs).Error; err != nil {
		return nil, err
	}
	addPairs(mcpPairs)

	// 调用数按日(LLM 聚合 + MCP 实时，Go 层合并)
	reqByDate := map[string]int{}
	var llmDays []gatewayResp.CostTrendItem
	dayDb := global.OPS_DB.WithContext(ctx).Table("gateway_cost_summary_daily s").Joins(costDeptJoins)
	dayDb = applyCostFilter(dayDb, f, start, end, deptIds)
	if err := dayDb.Select(`to_char(s.summary_date,'YYYY-MM-DD') AS date,
		COALESCE(SUM(s.request_count),0) AS requests`).
		Group("s.summary_date").Scan(&llmDays).Error; err != nil {
		return nil, err
	}
	for _, d := range llmDays {
		reqByDate[d.Date] += d.Requests
	}
	mcpTrendMap, err := mcpTrend(ctx, f, start, end)
	if err != nil {
		return nil, err
	}
	for date, item := range mcpTrendMap {
		reqByDate[date] += item.Requests
	}

	dates := make([]string, 0, len(pairSet))
	for date := range pairSet {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	trend := make([]gatewayResp.AdoptionTrendItem, 0, len(dates))
	for _, date := range dates {
		trend = append(trend, gatewayResp.AdoptionTrendItem{
			Date: date, ActiveUsers: len(pairSet[date]), Requests: reqByDate[date],
		})
	}
	return trend, nil
}

// adoptionPct 覆盖率%(分母 0 时给 0)。
func adoptionPct(active, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(active) / float64(total) * 100
}
