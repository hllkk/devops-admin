package gateway

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils/litellm"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// 聚合常量（对齐 AIHelms）
const (
	rollingRebuildDays = 60 // 滚动重建窗口
	bizTimezone        = "Asia/Shanghai"
)

// UsageAggregateService 用量聚合、预算重算与超限停用闭环。
// 日汇总滚动重建(自愈)+ budget_used 重算覆盖(幂等)+ 硬限超限自动停用(补 AIHelms 的坑)。
type UsageAggregateService struct{}

// AggregateUsage 聚合主任务：①滚动重建 cost_summary_daily ②重算 budget_used ③硬限超限停用闭环。
func (s *UsageAggregateService) AggregateUsage(ctx context.Context) (map[string]int, error) {
	result := map[string]int{"rebuilt": 0, "keysRecomputed": 0, "keysDisabled": 0}
	mdb := global.OPS_DB.WithContext(ctx)

	// ① 滚动重建日汇总
	if err := s.rebuildDailySummary(ctx, mdb); err != nil {
		return result, fmt.Errorf("日汇总重建失败: %w", err)
	}

	// ② 重算 budget_used
	result["keysRecomputed"] = s.recomputeBudgetUsed(ctx, mdb)

	// ③ 硬限超限停用闭环
	result["keysDisabled"] = s.enforceBudgetHardLimit(ctx, mdb)

	logger.WithCtx(ctx).Mod("gateway").Info(fmt.Sprintf("聚合完成: rebuilt summary, recomputed=%d keys, disabled=%d keys",
		result["keysRecomputed"], result["keysDisabled"]))
	return result, nil
}

// rebuildDailySummary 滚动重建日汇总：DELETE 近60天 → INSERT 分组聚合。
// 日桶按 Asia/Shanghai 切（规避 UTC date_trunc 错位8h）；聚合表不带状态机，DELETE+INSERT 自愈。
func (s *UsageAggregateService) rebuildDailySummary(ctx context.Context, db *gorm.DB) error {
	rebuildStart := time.Now().UTC().AddDate(0, 0, -rollingRebuildDays)
	return db.Transaction(func(tx *gorm.DB) error {
		// 先物理删近60天（Unscoped：聚合表是派生缓存，软删会占行累加；防中间态事务内 DELETE+INSERT）
		if err := tx.Unscoped().Where("summary_date >= ?", rebuildStart.Format("2006-01-02")).
			Delete(&gateway.CostSummaryDaily{}).Error; err != nil {
			return fmt.Errorf("删除旧聚合失败: %w", err)
		}
		// 分组聚合插入：按业务日(Shanghai切)+用户+密钥+模型+供应商分组
		// external/internal 都从原始日志 SUM（P1 internal 同 external）；token 各自 SUM
		// 注意：ai_key_id=0(未归因)与 user_id=0 也分组保留（成本不丢）
		sql := fmt.Sprintf(`
INSERT INTO gateway_cost_summary_daily
  (summary_date, user_id, ai_key_id, model, provider,
   request_count, prompt_tokens, completion_tokens, total_tokens,
   cache_read_tokens, cache_creation_tokens, external_cost, internal_cost, create_time, update_time)
SELECT
  date_trunc('day', started_at AT TIME ZONE '%s')::date AS summary_date,
  user_id, ai_key_id, model, COALESCE(provider, ''),
  COUNT(*) AS request_count,
  COALESCE(SUM(prompt_tokens),0), COALESCE(SUM(completion_tokens),0), COALESCE(SUM(total_tokens),0),
  COALESCE(SUM(cache_read_tokens),0), COALESCE(SUM(cache_creation_tokens),0),
  COALESCE(SUM(external_cost),0), COALESCE(SUM(internal_cost),0),
  NOW(), NOW()
FROM gateway_llm_log
WHERE started_at >= ?
GROUP BY summary_date, user_id, ai_key_id, model, provider`, bizTimezone)
		return tx.Exec(sql, rebuildStart).Error
	})
}

// recomputeBudgetUsed 重算每个有预算 Key 的 budget_used：按 budget_duration 滚动窗口
// 从原始日志 SUM external_cost 覆盖（幂等可回放，无调用记录归零，不做事件驱动累加）。
func (s *UsageAggregateService) recomputeBudgetUsed(ctx context.Context, db *gorm.DB) int {
	var keys []gateway.AiKey
	// 有预算限制或硬限的 Key 才重算（含已停用的，保持 budget_used 反映历史成本）
	if err := db.Where("budget_limit IS NOT NULL OR budget_hard_limit = true").
		Find(&keys).Error; err != nil {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Error("重算 budget_used: 查 Key 失败")
		return 0
	}
	now := time.Now().UTC()
	cnt := 0
	for i := range keys {
		window := budgetWindowDuration(keys[i].BudgetDuration)
		var sum float64
		db.Model(&gateway.LlmLog{}).
			Where("ai_key_id = ? AND ai_key_id <> 0 AND started_at >= ?", keys[i].AiKeyId, now.Add(-window)).
			Select("COALESCE(SUM(external_cost),0)").Scan(&sum)
		if err := db.Model(&gateway.AiKey{}).Where("ai_key_id = ?", keys[i].AiKeyId).
			Update("budget_used", sum).Error; err != nil {
			logger.WithCtx(ctx).Mod("gateway").Err(err).Field("aiKeyId", keys[i].AiKeyId).Error("更新 budget_used 失败")
			continue
		}
		cnt++
	}
	return cnt
}

// enforceBudgetHardLimit 硬限超限停用闭环（补 AIHelms 没做的坑）：
// budget_hard_limit=true && budget_used>=budget_limit && isActive → 复用 syncKeyToLitellm
// 停用(max_budget=0) + SysOperLog 审计 + logger 告警。停用后 budget_used 不清零(历史成本保留)。
func (s *UsageAggregateService) enforceBudgetHardLimit(ctx context.Context, db *gorm.DB) int {
	var keys []gateway.AiKey
	if err := db.Where("budget_hard_limit = true AND is_active = true AND budget_limit IS NOT NULL").
		Find(&keys).Error; err != nil {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Error("超限检查: 查 Key 失败")
		return 0
	}
	cli := litellm.Default()
	disabled := 0
	for i := range keys {
		k := &keys[i]
		if k.BudgetLimit == nil || k.BudgetUsed < *k.BudgetLimit {
			continue // 未超限
		}
		// 停用：复用 slice4 syncKeyToLitellm（isActive=false → max_budget=0）
		k.IsActive = false
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&gateway.AiKey{}).Where("ai_key_id = ?", k.AiKeyId).
				Update("is_active", false).Error; err != nil {
				return err
			}
			if cli != nil && k.LitellmKeyId != "" {
				return syncKeyToLitellm(ctx, cli, tx, k, false)
			}
			return nil
		})
		if err != nil {
			logger.WithCtx(ctx).Mod("gateway").Err(err).Field("aiKeyId", k.AiKeyId).
				Error("预算超限停用失败")
			continue
		}
		disabled++
		// 审计（直接 Create SysOperLog，避免 import service/system 成环）
		auditLog(ctx, db, k)
		logger.WithCtx(ctx).Mod("gateway").Warn(fmt.Sprintf(
			"预算超限已停用: aiKeyId=%d budgetUsed=%.4f budgetLimit=%.4f", k.AiKeyId, k.BudgetUsed, *k.BudgetLimit))
	}
	return disabled
}

// auditLog 记预算超限停用审计日志（直接写 SysOperLog，超限是低频事件同步写可接受）。
func auditLog(ctx context.Context, db *gorm.DB, k *gateway.AiKey) {
	log := system.SysOperLog{
		Title:        "gateway/ai_key",
		BusinessType: "2", // 更新
		Method:       "BudgetEnforcement",
		OperName:     "system",
		OperUrl:      "/gateway/ai-key/budget-enforcement",
		OperParam:    fmt.Sprintf(`{"aiKeyId":%d,"reason":"预算超限停用","budgetUsed":%.4f,"budgetLimit":%.4f}`, k.AiKeyId, k.BudgetUsed, *k.BudgetLimit),
		JsonResult:   `{"action":"disabled","maxBudgetSet":0}`,
		OperTime:     time.Now(),
		Status:       "0",
	}
	if err := db.Create(&log).Error; err != nil {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Error("超限停用审计日志写入失败")
	}
}

// budgetWindowDuration 预算周期 → 滚动窗口时长。
func budgetWindowDuration(duration string) time.Duration {
	switch duration {
	case gateway.BudgetDuration1d:
		return 24 * time.Hour
	case gateway.BudgetDuration7d:
		return 7 * 24 * time.Hour
	default:
		return 30 * 24 * time.Hour
	}
}
