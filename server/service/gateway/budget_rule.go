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
	"github.com/hllkk/devops-admin/server/utils/litellm"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// BudgetRuleService 多维预算管控(P3)：部门/用户级预算规则 CRUD + 软限预警通知 + 硬限超限停用。
// Key 级预算已有(usage_aggregate.go enforceBudgetHardLimit)，本服务管理部门/用户级规则，
// 与 Key 级并行独立；预算已用量读时实时聚合(复用成本分析归因 JOIN 口径)。
type BudgetRuleService struct{}

// GetBudgetRuleList 预算规则分页明细(含读时聚合 budgetUsed + 预警状态)。
func (s *BudgetRuleService) GetBudgetRuleList(ctx context.Context, q *gatewayReq.BudgetRuleSearch) ([]gatewayResp.BudgetRuleView, int64, error) {
	db := global.OPS_DB.WithContext(ctx).Model(&gateway.BudgetRule{})
	if q.ScopeType != "" {
		db = db.Where("scope_type = ?", q.ScopeType)
	}
	if q.IsActive != nil {
		db = db.Where("is_active = ?", *q.IsActive)
	}
	var total int64
	var rows []gateway.BudgetRule
	limit, offset := q.LimitOffset()
	if limit > 0 {
		if err := db.Count(&total).Order("scope_type, scope_id").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, 0, err
		}
	} else {
		if err := db.Count(&total).Order("scope_type, scope_id").Find(&rows).Error; err != nil {
			return nil, 0, err
		}
	}
	views := make([]gatewayResp.BudgetRuleView, 0, len(rows))
	for i := range rows {
		v := s.toView(ctx, &rows[i])
		views = append(views, v)
	}
	return views, total, nil
}

// CreateBudgetRule 新增预算规则(scope_type+scope_id 唯一校验+参数合法校验)。
func (s *BudgetRuleService) CreateBudgetRule(ctx context.Context, req *gatewayReq.BudgetRuleOperateParams, createBy int64) (gatewayResp.BudgetRuleView, error) {
	if err := s.validateOperate(ctx, req, 0); err != nil {
		return gatewayResp.BudgetRuleView{}, err
	}
	row := gateway.BudgetRule{
		ScopeType:       req.ScopeType,
		ScopeId:         req.ScopeId,
		BudgetLimit:     req.BudgetLimit,
		BudgetHardLimit: req.BudgetHardLimit,
		BudgetDuration:  normalizeBudgetDuration(req.BudgetDuration),
		SoftWarnPercent: normalizeSoftWarnPercent(req.SoftWarnPercent),
		IsActive:        req.IsActive == nil || *req.IsActive,
	}
	row.CreateBy = createBy
	row.UpdateBy = createBy
	if err := global.OPS_DB.WithContext(ctx).Create(&row).Error; err != nil {
		return gatewayResp.BudgetRuleView{}, fmt.Errorf("创建预算规则失败: %w", err)
	}
	return s.toView(ctx, &row), nil
}

// UpdateBudgetRule 修改预算规则。
func (s *BudgetRuleService) UpdateBudgetRule(ctx context.Context, req *gatewayReq.BudgetRuleOperateParams, updateBy int64) (gatewayResp.BudgetRuleView, error) {
	if req.RuleId == 0 {
		return gatewayResp.BudgetRuleView{}, fmt.Errorf("规则ID不能为空")
	}
	if err := s.validateOperate(ctx, req, req.RuleId); err != nil {
		return gatewayResp.BudgetRuleView{}, err
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.BudgetRule{}).Where("rule_id = ?", req.RuleId).
		Updates(map[string]any{
			"budget_limit":      req.BudgetLimit,
			"budget_hard_limit": req.BudgetHardLimit,
			"budget_duration":   normalizeBudgetDuration(req.BudgetDuration),
			"soft_warn_percent": normalizeSoftWarnPercent(req.SoftWarnPercent),
			"is_active":         isActive,
			"update_by":         updateBy,
		}).Error; err != nil {
		return gatewayResp.BudgetRuleView{}, fmt.Errorf("修改预算规则失败: %w", err)
	}
	var row gateway.BudgetRule
	global.OPS_DB.WithContext(ctx).Where("rule_id = ?", req.RuleId).First(&row)
	return s.toView(ctx, &row), nil
}

// DeleteBudgetRules 批量删除预算规则。
func (s *BudgetRuleService) DeleteBudgetRules(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return global.OPS_DB.WithContext(ctx).Where("rule_id IN ?", ids).Delete(&gateway.BudgetRule{}).Error
}

// BudgetAlertResult 单条预算检查结果(由 API 层消费，负责通知发送，规避 service↔gateway import 环)。
type BudgetAlertResult struct {
	Rule       *gateway.BudgetRule
	Used       float64
	Percent    float64
	PeriodKey  string
	AlertType  string // soft_warn / hard_limit
	TargetIds  []int64
	ScopeLabel string
	DisabledKeyIds []int64 // 硬限时停用的 Key ID 列表
}

// CheckBudgetAlerts 预算预警检查(由 AggregateUsage 定时任务调用，每 5 分钟)：
// 遍历活跃规则 → 读时聚合 scope 内总成本 → 返回待通知结果列表(通知发送由调用方 API 层负责，
// 规避 service/system↔service/gateway import 环——审批通知同模式)。
func (s *BudgetRuleService) CheckBudgetAlerts(ctx context.Context) ([]BudgetAlertResult, error) {
	db := global.OPS_DB.WithContext(ctx)
	var rules []gateway.BudgetRule
	if err := db.Where("is_active = true AND budget_limit > 0").Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("查预算规则失败: %w", err)
	}
	var results []BudgetAlertResult
	for i := range rules {
		r := &rules[i]
		used := s.calcScopeUsed(ctx, db, r)
		if r.BudgetLimit <= 0 {
			continue
		}
		percent := used / r.BudgetLimit * 100
		periodKey := s.periodKey(r.BudgetDuration)
		// 软限预警
		if percent >= float64(r.SoftWarnPercent) && !s.alertExists(db, r.RuleId, periodKey, gateway.BudgetAlertSoftWarn) {
			if err := db.Create(&gateway.BudgetAlert{RuleId: r.RuleId, PeriodKey: periodKey, AlertType: gateway.BudgetAlertSoftWarn}).Error; err != nil {
				// 落库失败会导致下一轮重复告警通知，须记 Error 供排查
				logger.WithCtx(ctx).Mod("gateway").Err(err).Field("ruleId", r.RuleId).Error("预算软限告警记录写入失败")
			}
			results = append(results, BudgetAlertResult{
				Rule: r, Used: used, Percent: percent, PeriodKey: periodKey,
				AlertType:  gateway.BudgetAlertSoftWarn,
				TargetIds:  s.resolveNotifyTargets(ctx, db, r, gateway.BudgetAlertSoftWarn),
				ScopeLabel: s.scopeLabel(ctx, db, r),
			})
		}
		// 硬限超限
		if r.BudgetHardLimit && used >= r.BudgetLimit && !s.alertExists(db, r.RuleId, periodKey, gateway.BudgetAlertHardLimit) {
			if err := db.Create(&gateway.BudgetAlert{RuleId: r.RuleId, PeriodKey: periodKey, AlertType: gateway.BudgetAlertHardLimit}).Error; err != nil {
				// 落库失败会导致下一轮重复触发硬限，须记 Error 供排查
				logger.WithCtx(ctx).Mod("gateway").Err(err).Field("ruleId", r.RuleId).Error("预算硬限告警记录写入失败")
			}
			keyIds := s.scopeActiveKeyIds(ctx, db, r)
			if len(keyIds) > 0 {
				if err := db.Model(&gateway.AiKey{}).Where("ai_key_id IN ?", keyIds).Update("is_active", false).Error; err != nil {
					// 管控动作失效(超限 Key 继续可用、成本失控)，且本路径在定时任务内 err 不上抛，
					// 不记 Error 则 sys_error 零记录
					logger.WithCtx(ctx).Mod("gateway").Err(err).Field("ruleId", r.RuleId).Error("预算硬限停用 Key 失败")
				}
				if cli := litellm.Default(); cli != nil {
					for _, kid := range keyIds {
						var k gateway.AiKey
						if db.Where("ai_key_id = ?", kid).First(&k).Error == nil {
							syncKeyToLitellm(ctx, cli, db, &k, false)
						}
					}
				}
			}
			// 审计日志
			log := system.SysOperLog{
				Title:        "gateway/budget_rule",
				BusinessType: "2",
				Method:       "BudgetRuleHardLimit",
				OperName:     "system",
				OperUrl:      "/gateway/budget/check",
				OperParam:    fmt.Sprintf(`{"ruleId":%d,"scopeType":"%s","scopeId":%d,"budgetUsed":%.4f,"budgetLimit":%.2f,"disabledKeys":%d}`, r.RuleId, r.ScopeType, r.ScopeId, used, r.BudgetLimit, len(keyIds)),
				JsonResult:   `{"action":"hard_limit_enforced"}`,
				OperTime:     time.Now(),
				Status:       "0",
			}
			if err := db.Create(&log).Error; err != nil {
				logger.WithCtx(ctx).Mod("gateway").Err(err).Field("ruleId", r.RuleId).Error("预算硬限审计日志写入失败")
			}
			results = append(results, BudgetAlertResult{
				Rule: r, Used: used, Percent: percent, PeriodKey: periodKey,
				AlertType:      gateway.BudgetAlertHardLimit,
				TargetIds:      s.resolveNotifyTargets(ctx, db, r, gateway.BudgetAlertHardLimit),
				ScopeLabel:     s.scopeLabel(ctx, db, r),
				DisabledKeyIds: keyIds,
			})
		}
	}
	return results, nil
}

// calcScopeUsed 读时聚合 scope 内总成本(LLM+MCP 双表，与 recomputeBudgetUsed 同口径)。
// 部门维度：部门Key(owner_type=dept,owner_id=scopeId)的消耗 + 该部门成员个人Key消耗。
// 用户维度：用户名下所有 Key 消耗。
func (s *BudgetRuleService) calcScopeUsed(ctx context.Context, db *gorm.DB, r *gateway.BudgetRule) float64 {
	window := budgetWindowDuration(r.BudgetDuration)
	since := time.Now().UTC().Add(-window)
	var sum float64
	switch r.ScopeType {
	case gateway.BudgetScopeDept:
		// 部门 Key 消耗 + 成员个人 Key 消耗(直挂口径,不含子部门)
		db.Raw(`SELECT COALESCE(
			(SELECT SUM(l.external_cost) FROM gateway_llm_log l
			 JOIN gateway_ai_key k ON k.ai_key_id = l.ai_key_id
			 WHERE k.owner_type = 'dept' AND k.owner_id = ? AND l.started_at >= ?),
			0) + COALESCE(
			(SELECT SUM(l.external_cost) FROM gateway_llm_log l
			 JOIN gateway_ai_key k ON k.ai_key_id = l.ai_key_id
			 JOIN sys_users u ON u.id = k.owner_id AND k.owner_type = 'user'
			 WHERE u.dept_id = ? AND l.started_at >= ?),
			0)`, r.ScopeId, since, r.ScopeId, since).Scan(&sum)
		var mcpSum float64
		db.Raw(`SELECT COALESCE(
			(SELECT SUM(m.external_cost) FROM gateway_mcp_log m
			 JOIN gateway_ai_key k ON k.ai_key_id = m.ai_key_id
			 WHERE k.owner_type = 'dept' AND k.owner_id = ? AND m.started_at >= ?),
			0) + COALESCE(
			(SELECT SUM(m.external_cost) FROM gateway_mcp_log m
			 JOIN gateway_ai_key k ON k.ai_key_id = m.ai_key_id
			 JOIN sys_users u ON u.id = k.owner_id AND k.owner_type = 'user'
			 WHERE u.dept_id = ? AND m.started_at >= ?),
			0)`, r.ScopeId, since, r.ScopeId, since).Scan(&mcpSum)
		sum += mcpSum
	case gateway.BudgetScopeUser:
		db.Raw(`SELECT COALESCE(SUM(l.external_cost),0) FROM gateway_llm_log l
			JOIN gateway_ai_key k ON k.ai_key_id = l.ai_key_id
			WHERE k.owner_type = 'user' AND k.owner_id = ? AND l.started_at >= ?`, r.ScopeId, since).Scan(&sum)
		var mcpSum float64
		db.Raw(`SELECT COALESCE(SUM(m.external_cost),0) FROM gateway_mcp_log m
			JOIN gateway_ai_key k ON k.ai_key_id = m.ai_key_id
			WHERE k.owner_type = 'user' AND k.owner_id = ? AND m.started_at >= ?`, r.ScopeId, since).Scan(&mcpSum)
		sum += mcpSum
	}
	return sum
}

// scopeActiveKeyIds scope 内当前活跃 Key 的 ID 列表。
func (s *BudgetRuleService) scopeActiveKeyIds(ctx context.Context, db *gorm.DB, r *gateway.BudgetRule) []int64 {
	var ids []int64
	switch r.ScopeType {
	case gateway.BudgetScopeDept:
		db.Model(&gateway.AiKey{}).
			Where(`is_active = true AND ((owner_type = 'dept' AND owner_id = ?) OR (owner_type = 'user' AND owner_id IN (SELECT id FROM sys_users WHERE dept_id = ?)))`,
				r.ScopeId, r.ScopeId).
			Pluck("ai_key_id", &ids)
	case gateway.BudgetScopeUser:
		db.Model(&gateway.AiKey{}).
			Where("is_active = true AND owner_type = 'user' AND owner_id = ?", r.ScopeId).
			Pluck("ai_key_id", &ids)
	}
	return ids
}

// resolveNotifyTargets 预算通知目标：部门规则→部门管理员(sys_departments.create_by)，
// 用户规则→该用户本人，负责人缺失时超管兜底。
// 硬限超限(alertType=hard_limit)影响面是全体受管成员：部门规则扩为部门直挂全体成员+管理员+超管
// (成员口径与 calcScopeUsed 直挂一致，不含子部门)；用户规则硬限影响面即本人，维持不变。
func (s *BudgetRuleService) resolveNotifyTargets(ctx context.Context, db *gorm.DB, r *gateway.BudgetRule, alertType string) []int64 {
	var targets []int64
	seen := make(map[int64]struct{})
	add := func(ids ...int64) {
		for _, id := range ids {
			if id == 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			targets = append(targets, id)
		}
	}
	switch r.ScopeType {
	case gateway.BudgetScopeDept:
		if alertType == gateway.BudgetAlertHardLimit {
			// 硬限已停用部门直挂全体成员的 Key，全员知晓；超管必达(管控动作留痕通知)
			var memberIds []int64
			db.Model(&system.SysUser{}).Where("dept_id = ?", r.ScopeId).Pluck("id", &memberIds)
			add(memberIds...)
		}
		var dept system.SysDepartment
		if err := db.Select("create_by").Where("dept_id = ?", r.ScopeId).First(&dept).Error; err == nil {
			add(dept.CreateBy)
		}
		if alertType == gateway.BudgetAlertHardLimit || len(targets) == 0 {
			var superIds []int64
			db.Model(&system.SysUser{}).Where("is_superadmin = true").Pluck("id", &superIds)
			add(superIds...)
		}
	case gateway.BudgetScopeUser:
		add(r.ScopeId)
	}
	return targets
}

// BudgetNoticeDraft 预算告警通知草稿(纯数据)。发送由调用方(API 层/timer 闭包)映射
// system.SendRequest 完成三渠道投递，规避 gateway service→system service 反向 import 环。
type BudgetNoticeDraft struct {
	RuleId        int64
	AlertType     string // soft_warn / hard_limit
	Title         string
	Content       string
	TargetUserIds []int64
}

// BudgetAlertNotices 告警结果→通知草稿(纯函数，文案组装修自原 API 层实现，两处调用方共用)。
func BudgetAlertNotices(results []BudgetAlertResult) []BudgetNoticeDraft {
	drafts := make([]BudgetNoticeDraft, 0, len(results))
	for i := range results {
		r := &results[i]
		var title, content string
		if r.AlertType == gateway.BudgetAlertHardLimit {
			title = fmt.Sprintf("AI 预算超限：%s", r.ScopeLabel)
			content = fmt.Sprintf("「%s」预算已超限（已用 ¥%.2f / 限额 ¥%.2f，%.1f%%），已停用该范围内所有活跃 Key。", r.ScopeLabel, r.Used, r.Rule.BudgetLimit, r.Percent)
		} else {
			title = fmt.Sprintf("AI 预算预警：%s", r.ScopeLabel)
			content = fmt.Sprintf("「%s」预算已用 %.1f%%（¥%.2f / ¥%.2f），达到预警阈值 %d%%。", r.ScopeLabel, r.Percent, r.Used, r.Rule.BudgetLimit, r.Rule.SoftWarnPercent)
		}
		drafts = append(drafts, BudgetNoticeDraft{
			RuleId:        r.Rule.RuleId,
			AlertType:     r.AlertType,
			Title:         title,
			Content:       content,
			TargetUserIds: r.TargetIds,
		})
	}
	return drafts
}

// scopeLabel scope 可读名(部门名/用户名)。
func (s *BudgetRuleService) scopeLabel(ctx context.Context, db *gorm.DB, r *gateway.BudgetRule) string {
	switch r.ScopeType {
	case gateway.BudgetScopeDept:
		var d system.SysDepartment
		if err := db.Select("dept_name").Where("dept_id = ?", r.ScopeId).First(&d).Error; err == nil {
			return d.DeptName
		}
	case gateway.BudgetScopeUser:
		var u system.SysUser
		if err := db.Select("nick_name").Where("id = ?", r.ScopeId).First(&u).Error; err == nil {
			return u.NickName
		}
	}
	return fmt.Sprintf("%s:%d", r.ScopeType, r.ScopeId)
}

// toView 预算规则→视图(读时聚合 budgetUsed + 预警状态)。
func (s *BudgetRuleService) toView(ctx context.Context, r *gateway.BudgetRule) gatewayResp.BudgetRuleView {
	db := global.OPS_DB.WithContext(ctx)
	used := s.calcScopeUsed(ctx, db, r)
	percent := 0.0
	if r.BudgetLimit > 0 {
		percent = used / r.BudgetLimit * 100
	}
	periodKey := s.periodKey(r.BudgetDuration)
	v := gatewayResp.BudgetRuleView{
		RuleId:            r.RuleId,
		ScopeType:         r.ScopeType,
		ScopeId:           r.ScopeId,
		ScopeName:         s.scopeLabel(ctx, db, r),
		BudgetLimit:       r.BudgetLimit,
		BudgetUsed:        used,
		BudgetUsedPercent: percent,
		BudgetHardLimit:   r.BudgetHardLimit,
		BudgetDuration:    r.BudgetDuration,
		SoftWarnPercent:   r.SoftWarnPercent,
		IsActive:          r.IsActive,
		IsSoftWarn:        s.alertExists(db, r.RuleId, periodKey, gateway.BudgetAlertSoftWarn),
		IsHardLimited:     s.alertExists(db, r.RuleId, periodKey, gateway.BudgetAlertHardLimit),
	}
	return v
}

// alertExists 本周期是否已告警(去重查询)。
func (s *BudgetRuleService) alertExists(db *gorm.DB, ruleId int64, periodKey, alertType string) bool {
	var cnt int64
	db.Model(&gateway.BudgetAlert{}).
		Where("rule_id = ? AND period_key = ? AND alert_type = ?", ruleId, periodKey, alertType).
		Count(&cnt)
	return cnt > 0
}

// periodKey 预算周期→去重键(月粒度 YYYY-MM，日/周同归月)。
func (s *BudgetRuleService) periodKey(duration string) string {
	now := time.Now()
	switch duration {
	case gateway.BudgetDuration1d:
		return now.Format("2006-01-02")
	case gateway.BudgetDuration7d:
		_, week := now.ISOWeek()
		return fmt.Sprintf("%d-W%02d", now.Year(), week)
	default:
		return now.Format("2006-01")
	}
}

// validateOperate 参数合法校验+scope_type+scope_id 唯一校验。
func (s *BudgetRuleService) validateOperate(ctx context.Context, req *gatewayReq.BudgetRuleOperateParams, excludeId int64) error {
	if req.ScopeType != gateway.BudgetScopeDept && req.ScopeType != gateway.BudgetScopeUser {
		return fmt.Errorf("维度仅支持 dept/user")
	}
	if req.ScopeId == 0 {
		return fmt.Errorf("对象ID不能为空")
	}
	// 校验 scope 对象存在
	db := global.OPS_DB.WithContext(ctx)
	switch req.ScopeType {
	case gateway.BudgetScopeDept:
		var cnt int64
		db.Model(&system.SysDepartment{}).Where("dept_id = ?", req.ScopeId).Count(&cnt)
		if cnt == 0 {
			return fmt.Errorf("部门 %d 不存在", req.ScopeId)
		}
	case gateway.BudgetScopeUser:
		var cnt int64
		db.Model(&system.SysUser{}).Where("id = ?", req.ScopeId).Count(&cnt)
		if cnt == 0 {
			return fmt.Errorf("用户 %d 不存在", req.ScopeId)
		}
	}
	// 唯一校验
	var exist gateway.BudgetRule
	q := db.Where("scope_type = ? AND scope_id = ?", req.ScopeType, req.ScopeId)
	if excludeId > 0 {
		q = q.Where("rule_id <> ?", excludeId)
	}
	if err := q.First(&exist).Error; err == nil {
		return fmt.Errorf("该维度对象已有预算规则(ruleId=%d)", exist.RuleId)
	}
	if req.BudgetHardLimit && req.BudgetLimit <= 0 {
		return fmt.Errorf("硬限必须设置预算上限(>0)")
	}
	return nil
}

// normalizeSoftWarnPercent 规范化软限阈值(非法值回退 80，上限 100)。
func normalizeSoftWarnPercent(p int) int {
	if p <= 0 {
		return 80
	}
	if p > 100 {
		return 100
	}
	return p
}
