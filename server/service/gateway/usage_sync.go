package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// 用量回流常量（对齐 AIHelms，可由 config 覆盖）
const (
	defaultSpendBatchSize      = 1000
	defaultMaxBatchesPerRun    = 50
	defaultReconcileWindowDays = 30
	syncStateKey               = "llm_logs"
	masterKeyToken             = "litellm_proxy_master_key"
	defaultUserId              = "default_user_id"
)

// UsageSyncService 用量回流与成本重算（对齐前端 /gateway/usage/* 与定时任务）。
// 平台 DB(gateway_llm_log) 是用量事实源，LiteLLM_SpendLogs 是回流数据源；
// 本地重算成本不信任 LiteLLM 的 spend 列；归因入库时解析（带缓存）。
type UsageSyncService struct{}

// SyncLLMLogs 从 LiteLLM_SpendLogs 增量回流用量日志：复合游标 keyset 分页 → 归因 →
// 成本重算 → ON CONFLICT(request_id) DO NOTHING 幂等落库 → 推进游标（不前进告警中止）。
func (s *UsageSyncService) SyncLLMLogs(ctx context.Context) (map[string]int, error) {
	result := map[string]int{"scanned": 0, "inserted": 0, "batches": 0}
	sdb := spendDB(ctx)
	mdb := global.OPS_DB.WithContext(ctx)

	state := s.loadSyncState(mdb)
	batchSize := global.OPS_CONFIG.Litellm.LogSyncBatch
	if batchSize <= 0 {
		batchSize = defaultSpendBatchSize
	}

	aiKeyCache := map[string]*gateway.AiKey{}
	userCache := map[string]*int64{}
	depCache := map[string]*gateway.ModelDeployment{}

	for batch := 0; batch < defaultMaxBatchesPerRun; batch++ {
		rows, err := s.fetchSpendBatch(sdb, state, batchSize)
		if err != nil {
			logger.WithCtx(ctx).Mod("gateway").Err(err).Error("用量回流查 LiteLLM_SpendLogs 失败")
			return result, err
		}
		if len(rows) == 0 {
			break
		}
		result["batches"]++

		logs := make([]gateway.LlmLog, 0, len(rows))
		for i := range rows {
			result["scanned"]++
			log := s.toLlmLog(ctx, mdb, &rows[i], aiKeyCache, userCache, depCache)
			if log == nil {
				continue // master key / default_user_id 跳过
			}
			logs = append(logs, *log)
		}
		if len(logs) > 0 {
			// ON CONFLICT(request_id) DO NOTHING 幂等；RowsAffected 为实际插入数(冲突跳过不计)
			tx := mdb.Begin()
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&logs).Error; err != nil {
				tx.Rollback()
				return result, fmt.Errorf("用量日志落库失败: %w", err)
			}
			result["inserted"] += int(tx.RowsAffected)
			tx.Commit()
			touchAiKeyLastUsed(ctx, mdb, logs)
		}

		// 游标推进：复合游标 (COALESCE(endTime,startTime), request_id)
		nextCursor := maxSpendCursor(rows)
		if nextCursor == nil || cursorLE(nextCursor, state.LastSyncAt, state.LastRequestId) {
			logger.WithCtx(ctx).Mod("gateway").Error("用量回流游标未推进，中止本轮（防无效循环）")
			break
		}
		state.LastSyncAt, state.LastRequestId = nextCursor.t, nextCursor.rid
		if err := s.saveSyncState(mdb, state); err != nil {
			return result, fmt.Errorf("游标保存失败: %w", err)
		}

		if len(rows) < batchSize {
			break // 不足一批，已到底
		}
	}
	logger.WithCtx(ctx).Mod("gateway").Info(fmt.Sprintf("用量回流完成: scanned=%d inserted≈%d batches=%d",
		result["scanned"], result["inserted"], result["batches"]))
	return result, nil
}

// ReconcileLLMLogs 对账兜底：每小时回灌近 N 天漏单。
// dev litellm 独立库/prod 共享库，NOT EXISTS 跨库查不到 gateway_llm_log → 分两步：
// sdb 查近 N 天 SpendLogs 行 → mdb 查已回流 request_id 集合 → Go 算差集漏单 → 落库。
func (s *UsageSyncService) ReconcileLLMLogs(ctx context.Context) (map[string]int, error) {
	result := map[string]int{"reconciled": 0}
	sdb := spendDB(ctx)
	mdb := global.OPS_DB.WithContext(ctx)
	window := global.OPS_CONFIG.Litellm.LogReconcileWindow
	if window <= 0 {
		window = defaultReconcileWindowDays
	}
	since := time.Now().UTC().AddDate(0, 0, -window)

	// 1. sdb 查近 N 天 SpendLogs 行（跳过 master key）
	var rows []gateway.LiteLLMSpendLog
	if err := sdb.Table(gateway.LiteLLMSpendLog{}.TableName()).
		Where(`"startTime" >= ?`, since).
		Where(`("api_key" IS NULL OR "api_key" = '' OR "api_key" <> ?)`, masterKeyToken).
		Order(`COALESCE("endTime","startTime") DESC, request_id DESC`).
		Limit(defaultSpendBatchSize).
		Find(&rows).Error; err != nil {
		return result, fmt.Errorf("对账查 SpendLogs 失败: %w", err)
	}
	if len(rows) == 0 {
		return result, nil
	}

	// 2. mdb 查已回流的 request_id 集合
	reqIds := make([]string, 0, len(rows))
	for i := range rows {
		if rows[i].RequestId != "" {
			reqIds = append(reqIds, rows[i].RequestId)
		}
	}
	var existing []string
	if len(reqIds) > 0 {
		mdb.Model(&gateway.LlmLog{}).Where("request_id IN ?", reqIds).Pluck("request_id", &existing)
	}
	existingSet := map[string]bool{}
	for _, r := range existing {
		existingSet[r] = true
	}

	// 3. 漏单 = SpendLogs 中 request_id 不在已回流集合
	aiKeyCache := map[string]*gateway.AiKey{}
	userCache := map[string]*int64{}
	depCache := map[string]*gateway.ModelDeployment{}
	logs := make([]gateway.LlmLog, 0, len(rows))
	for i := range rows {
		if existingSet[rows[i].RequestId] {
			continue // 已回流
		}
		log := s.toLlmLog(ctx, mdb, &rows[i], aiKeyCache, userCache, depCache)
		if log == nil {
			continue
		}
		logs = append(logs, *log)
	}
	if len(logs) > 0 {
		tx := mdb.Begin()
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&logs).Error; err != nil {
			tx.Rollback()
			return result, fmt.Errorf("对账落库失败: %w", err)
		}
		tx.Commit()
		result["reconciled"] = len(logs)
		touchAiKeyLastUsed(ctx, mdb, logs)
	}
	logger.WithCtx(ctx).Mod("gateway").Info(fmt.Sprintf("用量对账完成: reconciled=%d", result["reconciled"]))
	return result, nil
}

// GetUsageLogList 分页查用量日志(管理员视角，按 用户/密钥/部署/模型/时间过滤)。
func (s *UsageSyncService) GetUsageLogList(ctx context.Context, q gatewayReq.UsageLogSearch) (list []gateway.LlmLog, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&gateway.LlmLog{})
	if q.UserId != 0 {
		db = db.Where("user_id = ?", q.UserId)
	}
	if q.AiKeyId != 0 {
		db = db.Where("ai_key_id = ?", q.AiKeyId)
	}
	if q.DeploymentId != 0 {
		db = db.Where("deployment_id = ?", q.DeploymentId)
	}
	if q.Model != "" {
		db = db.Where("model LIKE ?", "%"+q.Model+"%")
	}
	if q.Provider != "" {
		db = db.Where("provider = ?", q.Provider)
	}
	if q.StartTime != "" {
		if t, e := time.Parse(time.RFC3339, q.StartTime); e == nil {
			db = db.Where("started_at >= ?", t)
		}
	}
	if q.EndTime != "" {
		if t, e := time.Parse(time.RFC3339, q.EndTime); e == nil {
			db = db.Where("started_at <= ?", t)
		}
	}
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("started_at DESC").Limit(limit).Offset(offset).Find(&list).Error
	} else {
		err = db.Count(&total).Order("started_at DESC").Find(&list).Error
	}
	return
}

type spendCursor struct {
	t   time.Time
	rid string
}

// maxSpendCursor 取本批最大的复合游标（COALESCE(endTime,startTime), request_id）。
func maxSpendCursor(rows []gateway.LiteLLMSpendLog) *spendCursor {
	var max *spendCursor
	for i := range rows {
		t := rows[i].EndTime
		if t.IsZero() {
			t = rows[i].StartTime
		}
		if t.IsZero() || rows[i].RequestId == "" {
			continue
		}
		c := &spendCursor{t: t.UTC(), rid: rows[i].RequestId}
		if max == nil || cursorGT(c, max) {
			max = c
		}
	}
	return max
}

func cursorGT(a, b *spendCursor) bool {
	if a.t.After(b.t) {
		return true
	}
	if a.t.Equal(b.t) {
		return a.rid > b.rid
	}
	return false
}

func cursorLE(a *spendCursor, t time.Time, rid string) bool {
	at := a.t
	if at.Before(t) {
		return true
	}
	if at.Equal(t) {
		return a.rid <= rid
	}
	return false
}

// fetchSpendBatch 复合游标 keyset 分页查 LiteLLM_SpendLogs。
func (s *UsageSyncService) fetchSpendBatch(sdb *gorm.DB, state *gateway.SyncState, limit int) ([]gateway.LiteLLMSpendLog, error) {
	var rows []gateway.LiteLLMSpendLog
	q := sdb.Table(gateway.LiteLLMSpendLog{}.TableName()).
		Where(`("api_key" IS NULL OR "api_key" = '' OR "api_key" <> ?)`, masterKeyToken).
		Where(`COALESCE("call_type",'') NOT IN ('list_mcp_tools','list_mcp_tool','mcp_list_tools','mcp_list_tool')`).
		Order(`COALESCE("endTime","startTime") ASC, request_id ASC`).
		Limit(limit)
	if state.LastRequestId != "" {
		// 复合游标：(COALESCE(endTime,startTime), request_id) > (last_t, last_rid)
		q = q.Where(`(COALESCE("endTime","startTime"), request_id) > (?, ?)`, state.LastSyncAt, state.LastRequestId)
	} else if !state.LastSyncAt.IsZero() {
		q = q.Where(`COALESCE("endTime","startTime") > ?`, state.LastSyncAt)
	}
	err := q.Find(&rows).Error
	return rows, err
}

// toLlmLog 单条 spend log → LlmLog（归因+成本）。master key/default_user_id 返回 nil 跳过。
func (s *UsageSyncService) toLlmLog(ctx context.Context, db *gorm.DB, r *gateway.LiteLLMSpendLog, aiKeyCache map[string]*gateway.AiKey, userCache map[string]*int64, depCache map[string]*gateway.ModelDeployment) *gateway.LlmLog {
	meta := jsonToMap(r.Metadata)
	// master key 跳过
	if r.ApiKey == masterKeyToken {
		return nil
	}

	aiKeyId := attributeAiKey(db, meta, r.ApiKey, aiKeyCache)
	userId := attributeUser(db, meta, r.User, userCache)
	depId, dep := attributeDeployment(db, meta, r.ModelId, r.Model, depCache)

	cacheRead, cacheCreation := parseCacheTokens(meta)
	// token 兜底：部分 LiteLLM 版本/provider 的 token 只在 metadata.usage_object(顶层列为 0)，
	// 兜底值同时供成本重算与落库(不兜则 token 计费模型的成本记 0)
	promptTokens, completionTokens, totalTokens := applyTokenFallback(r.PromptTokens, r.CompletionTokens, r.TotalTokens, meta)
	external, internal := calcCosts(dep, promptTokens, completionTokens, cacheRead, cacheCreation)

	started := r.StartTime.UTC()
	ended := r.EndTime.UTC()
	durationMs := 0
	if !ended.IsZero() && !started.IsZero() && ended.After(started) {
		durationMs = int(ended.Sub(started).Milliseconds())
	}
	return &gateway.LlmLog{
		RequestId:           r.RequestId,
		UserId:              userId,
		AiKeyId:             aiKeyId,
		DeploymentId:        depId,
		Model:               r.Model,
		Provider:            r.CustomLlmProvider,
		CallType:            r.CallType,
		PromptTokens:        promptTokens,
		CompletionTokens:    completionTokens,
		TotalTokens:         totalTokens,
		CacheReadTokens:     cacheRead,
		CacheCreationTokens: cacheCreation,
		ExternalCost:        external,
		InternalCost:        internal,
		DurationMs:          durationMs,
		StartedAt:           started,
		EndedAt:             ended,
		SessionId:           r.SessionId,
		Metadata:            datatypes.JSON(r.Metadata),
		SyncedAt:            time.Now().UTC(),
	}
}

// touchAiKeyLastUsed 回填 AiKey.last_used_at(僵尸 Key 治理)：按 Key 聚合本批最大
// started_at，仅前推不回退(幂等，回流/对账双路径复用)；失败仅告警不影响用量主流程。
func touchAiKeyLastUsed(ctx context.Context, db *gorm.DB, logs []gateway.LlmLog) {
	latest := map[int64]time.Time{}
	for i := range logs {
		if logs[i].AiKeyId == 0 || logs[i].StartedAt.IsZero() {
			continue
		}
		if t, ok := latest[logs[i].AiKeyId]; !ok || logs[i].StartedAt.After(t) {
			latest[logs[i].AiKeyId] = logs[i].StartedAt
		}
	}
	for id, t := range latest {
		if err := db.Model(&gateway.AiKey{}).
			Where("ai_key_id = ? AND (last_used_at IS NULL OR last_used_at < ?)", id, t).
			Update("last_used_at", t).Error; err != nil {
			logger.WithCtx(ctx).Mod("gateway").Warn(fmt.Sprintf("回填密钥 %d 最近使用时间失败: %v", id, err))
		}
	}
}

// attributeAiKey 归因 AiKey：metadata.user_api_key_alias→litellm_key_alias 或 api_key→litellm_key_id。
func attributeAiKey(db *gorm.DB, meta map[string]any, apiKey string, cache map[string]*gateway.AiKey) int64 {
	alias, _ := meta["user_api_key_alias"].(string)
	cacheKey := alias
	if cacheKey == "" {
		cacheKey = apiKey
	}
	if cacheKey == "" {
		return 0
	}
	if v, ok := cache[cacheKey]; ok {
		if v != nil {
			return v.AiKeyId
		}
		return 0
	}
	var k gateway.AiKey
	var err error
	if alias != "" {
		err = db.Where("litellm_key_alias = ?", alias).First(&k).Error
	} else {
		err = db.Where("litellm_key_id = ?", apiKey).First(&k).Error
	}
	if err != nil {
		cache[cacheKey] = nil // 缓存未命中，避免重复查询
		return 0
	}
	cache[cacheKey] = &k
	return k.AiKeyId
}

// attributeUser 归因 User：metadata.user_api_key_user_id 或 user 列 "devops_user_{id}"→SysUser.id。
// default_user_id 跳过，归因失败返回 0。
func attributeUser(db *gorm.DB, meta map[string]any, userField string, cache map[string]*int64) int64 {
	lookup := userField
	if v, ok := meta["user_api_key_user_id"].(string); ok && v != "" {
		lookup = v
	}
	if lookup == "" || lookup == defaultUserId {
		return 0
	}
	if v, ok := cache[lookup]; ok {
		if v != nil {
			return *v
		}
		return 0
	}
	// devops-admin 的 AiKey 推送时 UserID="devops_user_{ownerId}"，解析出 userId
	userId := parseDevopsUserId(lookup)
	if userId == 0 {
		// 兜底按 litellm_user_id 查（AIHelms 模式，devops-admin 无此字段则跳过）
		cache[lookup] = nil
		return 0
	}
	var cnt int64
	if err := db.Model(&system.SysUser{}).Where("id = ?", userId).Count(&cnt).Error; err != nil || cnt == 0 {
		cache[lookup] = nil
		return 0
	}
	id := userId
	cache[lookup] = &id
	return id
}

// parseDevopsUserId 从 "devops_user_{id}" 解析 userId。
func parseDevopsUserId(s string) int64 {
	const prefix = "devops_user_"
	if !strings.HasPrefix(s, prefix) {
		return 0
	}
	var id int64
	_, err := fmt.Sscanf(s[len(prefix):], "%d", &id)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

// attributeDeployment 归因 Deployment：model_id→litellm_model_id 优先，兜底 model_name 剥 (Anthropic)→Model.model_key。
func attributeDeployment(db *gorm.DB, meta map[string]any, modelId, modelName string, cache map[string]*gateway.ModelDeployment) (int64, *gateway.ModelDeployment) {
	litellmId := modelId
	if litellmId == "" {
		if v, ok := meta["model_id"].(string); ok {
			litellmId = v
		}
	}
	cacheKey := litellmId
	if cacheKey == "" {
		cacheKey = modelName
	}
	if cacheKey == "" {
		return 0, nil
	}
	if v, ok := cache[cacheKey]; ok {
		if v != nil {
			return v.DeploymentId, v
		}
		return 0, nil
	}
	var dep gateway.ModelDeployment
	var err error
	if litellmId != "" {
		err = db.Where("litellm_model_id = ?", litellmId).First(&dep).Error
	}
	if err != nil && modelName != "" {
		// 兜底按 model_name 剥 (Anthropic) 后缀匹配 Model.model_key → Deployment
		lookup := strings.TrimSuffix(modelName, gateway.ModelAnthropicSuffix)
		var m gateway.Model
		if e2 := db.Where("model_key = ?", lookup).First(&m).Error; e2 == nil {
			err = db.Where("model_id = ?", m.ModelId).First(&dep).Error
		}
	}
	if err != nil {
		cache[cacheKey] = nil
		return 0, nil
	}
	cache[cacheKey] = &dep
	return dep.DeploymentId, &dep
}

// parseCacheTokens 解析缓存 token：OpenAI(prompt_tokens_details.cached_tokens) / Anthropic(cache_read_input_tokens)。
func parseCacheTokens(meta map[string]any) (cacheRead, cacheCreation int) {
	usage, ok := meta["usage_object"].(map[string]any)
	if !ok {
		return
	}
	// OpenAI: prompt_tokens_details.cached_tokens
	if details, ok := usage["prompt_tokens_details"].(map[string]any); ok {
		if v, ok := details["cached_tokens"].(float64); ok {
			cacheRead = int(v)
		}
	}
	// Anthropic: cache_read_input_tokens（OpenAI 未命中时兜底）
	if cacheRead == 0 {
		if v, ok := usage["cache_read_input_tokens"].(float64); ok {
			cacheRead = int(v)
		}
	}
	// Anthropic 缓存创建
	if v, ok := usage["cache_creation_input_tokens"].(float64); ok {
		cacheCreation = int(v)
	}
	return
}

// extractTokensFromUsage 从 metadata.usage_object 提取 prompt/completion token（兜底）。
func extractTokensFromUsage(meta map[string]any) (prompt, completion int) {
	usage, ok := meta["usage_object"].(map[string]any)
	if !ok {
		return
	}
	if v, ok := usage["prompt_tokens"].(float64); ok {
		prompt = int(v)
	}
	if v, ok := usage["completion_tokens"].(float64); ok {
		completion = int(v)
	}
	return
}

// applyTokenFallback token 兜底纯函数(可单测)：顶层列为 0 时从 metadata.usage_object 补
// prompt/completion(仅零值补位，不覆盖非零列值)；total 为 0 时回落 prompt+completion 之和。
// 返回值供 calcCosts 与 LlmLog 落库共用——只兜一处则两侧口径分叉。
func applyTokenFallback(prompt, completion, total int, meta map[string]any) (int, int, int) {
	if prompt == 0 || completion == 0 {
		if pt, ct := extractTokensFromUsage(meta); pt > 0 || ct > 0 {
			if prompt == 0 {
				prompt = pt
			}
			if completion == 0 {
				completion = ct
			}
		}
	}
	if total == 0 {
		total = prompt + completion
	}
	return prompt, completion, total
}

// calcCosts 成本重算：external 用 deployment.model_info 四键（¥/百万token），internal P1 同 external。
// 本地重算不信任 LiteLLM 的 spend 列；deployment 为 nil（归因失败）返回 0。
func calcCosts(dep *gateway.ModelDeployment, prompt, completion, cacheRead, cacheCreation int) (external, internal float64) {
	if dep == nil {
		return 0, 0
	}
	info := jsonToMap(dep.ModelInfo)
	billing := dep.BillingType
	if billing == "" {
		billing = gateway.BillingTypeToken
	}
	if billing == gateway.BillingTypePerCall {
		if dep.CostPerCall != nil {
			external = *dep.CostPerCall
		}
		return external, external
	}
	const million = 1000000.0
	in := toFloatCost(info["input_cost"])
	out := toFloatCost(info["output_cost"])
	cr := toFloatCost(info["cache_read_cost"])
	cc := toFloatCost(info["cache_creation_cost"])
	external = (in*float64(prompt) + out*float64(completion) + cr*float64(cacheRead) + cc*float64(cacheCreation)) / million
	// 内部结算定价：有 internal_* 键则独立计算对内成本，未填回落 external（兼容历史数据）
	iin := toFloatCost(info["internal_input_cost"])
	iout := toFloatCost(info["internal_output_cost"])
	icr := toFloatCost(info["internal_cache_read_cost"])
	icc := toFloatCost(info["internal_cache_creation_cost"])
	if iin == 0 && iout == 0 && icr == 0 && icc == 0 {
		return external, external
	}
	internal = (iin*float64(prompt) + iout*float64(completion) + icr*float64(cacheRead) + icc*float64(cacheCreation)) / million
	return external, internal
}

// toFloatCost 成本键数值容错。
func toFloatCost(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()
		return f
	}
	return 0
}

// ----------------------------------------------------------------------------
// 游标状态与 DB 抽象
// ----------------------------------------------------------------------------

func (s *UsageSyncService) loadSyncState(db *gorm.DB) *gateway.SyncState {
	var state gateway.SyncState
	if err := db.Where("key = ?", syncStateKey).First(&state).Error; err != nil {
		// 首次同步：回退 1 小时启动（对齐 AIHelms）
		state = gateway.SyncState{
			Key:        syncStateKey,
			LastSyncAt: time.Now().UTC().Add(-time.Hour),
		}
		db.Create(&state)
	}
	return &state
}

func (s *UsageSyncService) saveSyncState(db *gorm.DB, state *gateway.SyncState) error {
	state.UpdatedAt = time.Now().UTC()
	return db.Save(state).Error
}

// spendDB 取 LiteLLM spend 只读连接，未配置则复用主库。
func spendDB(ctx context.Context) *gorm.DB {
	if global.OPS_SPEND_DB != nil {
		return global.OPS_SPEND_DB.WithContext(ctx)
	}
	return global.OPS_DB.WithContext(ctx)
}
