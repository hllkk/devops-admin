package gateway

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	"github.com/hllkk/devops-admin/server/model/system"
)

// DashboardService 看板查询（读聚合表 cost_summary_daily + 部分实时聚合）。
// 普通用户看自己(scope=self,userId 过滤)、超管看全部(scope=all)。
type DashboardService struct{}

// 看板维度
const (
	dimensionUser  = "user"
	dimensionModel = "model"
	dimensionAiKey = "aiKey"
)

// GetOverview 总览：总成本/请求数/token/预算汇总（读聚合表）。
func (s *DashboardService) GetOverview(ctx context.Context, start, end time.Time, scope string, userId int64) (gatewayResp.DashboardOverview, error) {
	var ov gatewayResp.DashboardOverview
	db := applyScope(global.OPS_DB.WithContext(ctx).Model(&gateway.CostSummaryDaily{}), scope, userId)
	db = db.Where("summary_date >= ? AND summary_date <= ?", start.Format("2006-01-02"), end.Format("2006-01-02"))
	type agg struct {
		Requests  int
		ExtCost   float64
		IntCost   float64
		Total     int64
		Input     int64
		Output    int64
		CacheRead int64
	}
	var a agg
	if err := db.Select("COALESCE(SUM(request_count),0) AS requests, COALESCE(SUM(external_cost),0) AS ext_cost, COALESCE(SUM(internal_cost),0) AS int_cost, COALESCE(SUM(total_tokens),0) AS total, COALESCE(SUM(prompt_tokens),0) AS input, COALESCE(SUM(completion_tokens),0) AS output, COALESCE(SUM(cache_read_tokens),0) AS cache_read").Scan(&a).Error; err != nil {
		return ov, err
	}
	ov.TotalRequests, ov.TotalCost, ov.InternalCost = a.Requests, a.ExtCost, a.IntCost
	ov.TotalTokens, ov.InputTokens, ov.OutputTokens, ov.CacheReadTokens = a.Total, a.Input, a.Output, a.CacheRead

	// 预算汇总（有预算限制的 Key）
	bdb := global.OPS_DB.WithContext(ctx).Model(&gateway.AiKey{})
	if scope == "self" {
		bdb = bdb.Where("owner_type = ? AND owner_id = ?", gateway.OwnerTypeUser, userId)
	}
	var b struct{ Used, Limit float64 }
	bdb.Where("budget_limit IS NOT NULL").
		Select("COALESCE(SUM(budget_used),0) AS used, COALESCE(SUM(budget_limit),0) AS limit").Scan(&b)
	ov.BudgetUsedTotal, ov.BudgetLimitTotal = b.Used, b.Limit
	return ov, nil
}

// GetTrend 按日成本/调用量趋势（读聚合表 GROUP BY summary_date）。
func (s *DashboardService) GetTrend(ctx context.Context, start, end time.Time, scope string, userId int64) ([]gatewayResp.TrendItem, error) {
	db := applyScope(global.OPS_DB.WithContext(ctx).Model(&gateway.CostSummaryDaily{}), scope, userId)
	db = db.Where("summary_date >= ? AND summary_date <= ?", start.Format("2006-01-02"), end.Format("2006-01-02"))
	var items []gatewayResp.TrendItem
	if err := db.Select("to_char(summary_date,'YYYY-MM-DD') AS date, COALESCE(SUM(external_cost),0) AS cost, COALESCE(SUM(request_count),0) AS requests, COALESCE(SUM(total_tokens),0) AS tokens").
		Group("summary_date").Order("summary_date ASC").Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// GetTop Top10 排行（按维度 user/model/aiKey，排序键 sort: cost/requests/tokens）。
func (s *DashboardService) GetTop(ctx context.Context, start, end time.Time, dimension, sort, scope string, userId int64) ([]gatewayResp.TopItem, error) {
	db := applyScope(global.OPS_DB.WithContext(ctx).Model(&gateway.CostSummaryDaily{}), scope, userId)
	db = db.Where("summary_date >= ? AND summary_date <= ?", start.Format("2006-01-02"), end.Format("2006-01-02"))
	var groupCol, labelExpr string
	switch dimension {
	case dimensionModel:
		groupCol, labelExpr = "model", "model"
	case dimensionAiKey:
		groupCol, labelExpr = "ai_key_id", "COALESCE(CAST(ai_key_id AS TEXT),'未归因')"
	default: // user
		groupCol, labelExpr = "user_id", "COALESCE(CAST(user_id AS TEXT),'未归因')"
	}
	// 排序键白名单映射到聚合列(外层引用别名，PostgreSQL GROUP BY 聚合别名可排序)
	orderCol := map[string]string{
		"cost": "cost", "requests": "requests", "tokens": "tokens",
	}[sort]
	if orderCol == "" {
		orderCol = "cost"
	}
	// 先按维度分组聚合，Top10；label 先用 ID，再 Go 层补名
	type row struct {
		Label    string
		Cost     float64
		Requests int
		Tokens   int64
	}
	var rows []row
	if err := db.Select(fmt.Sprintf("%s AS label, COALESCE(SUM(external_cost),0) AS cost, COALESCE(SUM(request_count),0) AS requests, COALESCE(SUM(total_tokens),0) AS tokens", labelExpr)).
		Where(groupCol + " IS NOT NULL").Group(groupCol).Order(orderCol + " DESC").Limit(10).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]gatewayResp.TopItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, gatewayResp.TopItem{Name: s.resolveDimensionLabel(ctx, dimension, r.Label), Cost: r.Cost, Requests: r.Requests, Tokens: r.Tokens})
	}
	return items, nil
}

// GetBudget 预算执行率（按 Key，有预算限制的）。
func (s *DashboardService) GetBudget(ctx context.Context, scope string, userId int64) ([]gatewayResp.BudgetItem, error) {
	db := global.OPS_DB.WithContext(ctx).Model(&gateway.AiKey{}).Where("budget_limit IS NOT NULL")
	if scope == "self" {
		db = db.Where("owner_type = ? AND owner_id = ?", gateway.OwnerTypeUser, userId)
	}
	var keys []gateway.AiKey
	if err := db.Order("budget_used DESC").Find(&keys).Error; err != nil {
		return nil, err
	}
	items := make([]gatewayResp.BudgetItem, 0, len(keys))
	for i := range keys {
		k := &keys[i]
		rate := 0.0
		if k.BudgetLimit != nil && *k.BudgetLimit > 0 {
			rate = k.BudgetUsed / *k.BudgetLimit * 100
		}
		limit := 0.0
		if k.BudgetLimit != nil {
			limit = *k.BudgetLimit
		}
		items = append(items, gatewayResp.BudgetItem{
			AiKeyId:     k.AiKeyId,
			Name:        k.Name,
			OwnerName:   s.resolveOwnerName(ctx, k),
			BudgetLimit: limit,
			BudgetUsed:  k.BudgetUsed,
			UsageRate:   rate,
			HardLimit:   k.BudgetHardLimit,
			IsActive:    k.IsActive,
		})
	}
	return items, nil
}

// applyScope scope=self 加 user_id 过滤；scope=all 不过滤（超管）。
func applyScope(db *gorm.DB, scope string, userId int64) *gorm.DB {
	if scope == "self" && userId != 0 {
		return db.Where("user_id = ?", userId)
	}
	return db
}

// resolveDimensionLabel 维度 ID → 可读名（用户名/模型名/Key名）。
func (s *DashboardService) resolveDimensionLabel(ctx context.Context, dimension, id string) string {
	if id == "" || id == "未归因" {
		return "未归因"
	}
	switch dimension {
	case dimensionUser:
		var u system.SysUser
		if global.OPS_DB.WithContext(ctx).Select("nick_name").Where("id = ?", id).First(&u).Error == nil {
			return u.NickName
		}
		return "用户:" + id
	case dimensionAiKey:
		var k gateway.AiKey
		if global.OPS_DB.WithContext(ctx).Select("name").Where("ai_key_id = ?", id).First(&k).Error == nil {
			return k.Name
		}
		return "Key:" + id
	}
	return id // model 维度直接是模型名
}

// resolveOwnerName Key 归属 → 用户名/部门名。
func (s *DashboardService) resolveOwnerName(ctx context.Context, k *gateway.AiKey) string {
	switch k.OwnerType {
	case gateway.OwnerTypeUser:
		var u system.SysUser
		if global.OPS_DB.WithContext(ctx).Select("nick_name").Where("id = ?", k.OwnerId).First(&u).Error == nil {
			return u.NickName
		}
	case gateway.OwnerTypeDept:
		var d system.SysDepartment
		if global.OPS_DB.WithContext(ctx).Select("dept_name").Where("dept_id = ?", k.OwnerId).First(&d).Error == nil {
			return d.DeptName
		}
	}
	return fmt.Sprintf("%s:%d", k.OwnerType, k.OwnerId)
}
