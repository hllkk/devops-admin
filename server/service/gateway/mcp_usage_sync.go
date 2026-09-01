package gateway

import (
	"context"
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// MCP 调用日志回流（P3）：与 SyncLLMLogs 同源同管线（LiteLLM_SpendLogs），
// 按 mcp_namespaced_tool_name 非空行互斥分流到 gateway_mcp_log。
// 规避 AIHelms mcp_tasks 坑：request_id 唯一约束+ON CONFLICT 幂等(其无索引 N+1 查重)、
// 复合游标增量(其固定 10min 回看窗口停摆丢数)、每小时对账兜底(其全无)、
// namespaced_name 整串精确匹配工具表(其 split '_' 切名歧义)、status 透传 LiteLLM(其硬编码 success)。

// mcpToolRef MCP 调用归因结果缓存条目：namespaced_name → 工具/服务器定位与计费单价。
type mcpToolRef struct {
	mcpServerId int64
	serverName  string // 路由名(LiteLLM 锚点,展示用)
	toolName    string
	tool        *gateway.MCPTool // nil=平台无此工具(继承服务器计费)
	server      *gateway.MCPServer
}

// SyncMcpLogs 从 LiteLLM_SpendLogs 增量回流 MCP 调用日志：复合游标(key=mcp_logs,
// 与 llm_logs 各自独立) keyset 分页 → namespaced_name 归因工具/服务器 → per_call 成本
// 自算 → ON CONFLICT(request_id) 幂等落库 → server.call_count 增量 → 推进游标。
func (s *UsageSyncService) SyncMcpLogs(ctx context.Context) (map[string]int, error) {
	result := map[string]int{"scanned": 0, "inserted": 0, "batches": 0}
	sdb := spendDB(ctx)
	mdb := global.OPS_DB.WithContext(ctx)

	state := loadSyncStateByKey(mdb, syncStateKeyMcp)
	batchSize := global.OPS_CONFIG.Litellm.LogSyncBatch
	if batchSize <= 0 {
		batchSize = defaultSpendBatchSize
	}

	aiKeyCache := map[string]*gateway.AiKey{}
	userCache := map[string]*int64{}
	toolCache := map[string]*mcpToolRef{}

	for batch := 0; batch < defaultMaxBatchesPerRun; batch++ {
		rows, err := s.fetchMcpSpendBatch(sdb, state, batchSize)
		if err != nil {
			logger.WithCtx(ctx).Mod("gateway").Err(err).Error("MCP回流查 LiteLLM_SpendLogs 失败")
			return result, err
		}
		if len(rows) == 0 {
			break
		}
		result["batches"]++

		logs := make([]gateway.McpLog, 0, len(rows))
		for i := range rows {
			result["scanned"]++
			log := s.toMcpLog(ctx, mdb, &rows[i], aiKeyCache, userCache, toolCache)
			if log == nil {
				continue // master key 跳过
			}
			logs = append(logs, *log)
		}
		if len(logs) > 0 {
			if err := insertMcpLogs(ctx, mdb, logs); err != nil {
				return result, err
			}
			result["inserted"] += len(logs)
		}

		nextCursor := lastSpendCursor(rows)
		if nextCursor == nil || cursorStalled(nextCursor, state.LastSyncAt, state.LastRequestId) {
			logger.WithCtx(ctx).Mod("gateway").Error("MCP回流游标未推进，中止本轮（防无效循环）")
			break
		}
		state.LastSyncAt, state.LastRequestId = nextCursor.t, nextCursor.rid
		if err := s.saveSyncState(mdb, state); err != nil {
			return result, fmt.Errorf("MCP回流游标保存失败: %w", err)
		}

		if len(rows) < batchSize {
			break // 不足一批，已到底
		}
	}
	logger.WithCtx(ctx).Mod("gateway").Info(fmt.Sprintf("MCP回流完成: scanned=%d inserted=%d batches=%d",
		result["scanned"], result["inserted"], result["batches"]))
	return result, nil
}

// ReconcileMcpLogs MCP 漏单对账兜底：每小时回灌近 N 天（两步查询避跨库 NOT EXISTS）。
func (s *UsageSyncService) ReconcileMcpLogs(ctx context.Context) (map[string]int, error) {
	result := map[string]int{"reconciled": 0}
	sdb := spendDB(ctx)
	mdb := global.OPS_DB.WithContext(ctx)
	window := global.OPS_CONFIG.Litellm.LogReconcileWindow
	if window <= 0 {
		window = defaultReconcileWindowDays
	}
	since := time.Now().UTC().AddDate(0, 0, -window)

	// 1. sdb 查近 N 天 SpendLogs 的 MCP 行（跳过 master key）
	var rows []gateway.LiteLLMSpendLog
	if err := sdb.Table(gateway.LiteLLMSpendLog{}.TableName()).
		Select(gateway.LiteLLMSpendLog{}.SelectColumns()).
		Where(`"startTime" >= ?`, since).
		Where(`("api_key" IS NULL OR "api_key" = '' OR "api_key" <> ?)`, masterKeyToken).
		Where(`"mcp_namespaced_tool_name" IS NOT NULL AND "mcp_namespaced_tool_name" <> ''`).
		Order(`COALESCE("endTime","startTime") DESC, request_id DESC`).
		Limit(defaultSpendBatchSize).
		Find(&rows).Error; err != nil {
		return result, fmt.Errorf("MCP对账查 SpendLogs 失败: %w", err)
	}
	if len(rows) == 0 {
		return result, nil
	}

	// 2. mdb 查已回流的 request_id 集合 → Go 算差集漏单
	reqIds := make([]string, 0, len(rows))
	for i := range rows {
		if rows[i].RequestId != "" {
			reqIds = append(reqIds, rows[i].RequestId)
		}
	}
	var existing []string
	if len(reqIds) > 0 {
		mdb.Model(&gateway.McpLog{}).Where("request_id IN ?", reqIds).Pluck("request_id", &existing)
	}
	existingSet := map[string]bool{}
	for _, r := range existing {
		existingSet[r] = true
	}

	aiKeyCache := map[string]*gateway.AiKey{}
	userCache := map[string]*int64{}
	toolCache := map[string]*mcpToolRef{}
	logs := make([]gateway.McpLog, 0, len(rows))
	for i := range rows {
		if existingSet[rows[i].RequestId] {
			continue
		}
		log := s.toMcpLog(ctx, mdb, &rows[i], aiKeyCache, userCache, toolCache)
		if log == nil {
			continue
		}
		logs = append(logs, *log)
	}
	if len(logs) > 0 {
		if err := insertMcpLogs(ctx, mdb, logs); err != nil {
			return result, err
		}
		result["reconciled"] = len(logs)
	}
	return result, nil
}

// fetchMcpSpendBatch 复合游标 keyset 分页查 LiteLLM_SpendLogs 的 MCP 行
// （mcp_namespaced_tool_name 非空；排除 list_mcp_tools 类管理动作）。
func (s *UsageSyncService) fetchMcpSpendBatch(sdb *gorm.DB, state *gateway.SyncState, limit int) ([]gateway.LiteLLMSpendLog, error) {
	var rows []gateway.LiteLLMSpendLog
	q := sdb.Table(gateway.LiteLLMSpendLog{}.TableName()).
		Select(gateway.LiteLLMSpendLog{}.SelectColumns()).
		Where(`("api_key" IS NULL OR "api_key" = '' OR "api_key" <> ?)`, masterKeyToken).
		Where(`COALESCE("call_type",'') NOT IN ('list_mcp_tools','list_mcp_tool','mcp_list_tools','mcp_list_tool')`).
		Where(`"mcp_namespaced_tool_name" IS NOT NULL AND "mcp_namespaced_tool_name" <> ''`).
		Order(`COALESCE("endTime","startTime") ASC, request_id ASC`).
		Limit(limit)
	if state.LastRequestId != "" {
		q = q.Where(`(COALESCE("endTime","startTime"), request_id) > (?, ?)`, state.LastSyncAt, state.LastRequestId)
	} else if !state.LastSyncAt.IsZero() {
		q = q.Where(`COALESCE("endTime","startTime") > ?`, state.LastSyncAt)
	}
	err := q.Find(&rows).Error
	return rows, err
}

// toMcpLog 单条 spend log → McpLog（归因+成本自算）。master key 返回 nil 跳过。
func (s *UsageSyncService) toMcpLog(ctx context.Context, db *gorm.DB, r *gateway.LiteLLMSpendLog, aiKeyCache map[string]*gateway.AiKey, userCache map[string]*int64, toolCache map[string]*mcpToolRef) *gateway.McpLog {
	if r.ApiKey == masterKeyToken {
		return nil
	}
	meta := jsonToMap(r.Metadata)

	aiKeyId := attributeAiKey(db, meta, r.ApiKey, aiKeyCache)
	userId := attributeUser(db, meta, r.User, userCache)
	ref := attributeMcpTool(db, r.McpNamespacedToolName, toolCache)
	external, internal := calcMcpCosts(ref)

	started := r.StartTime.UTC()
	ended := r.EndTime.UTC()
	durationMs := 0
	if !ended.IsZero() && !started.IsZero() && ended.After(started) {
		durationMs = int(ended.Sub(started).Milliseconds())
	}
	status := r.Status
	if status == "" {
		status = gateway.McpCallStatusSuccess
	}
	serverName := r.McpNamespacedToolName
	if ref != nil {
		serverName = ref.serverName
	}
	return &gateway.McpLog{
		RequestId:      r.RequestId,
		UserId:         userId,
		AiKeyId:        aiKeyId,
		McpServerId:    ref.mcpServerIdOf(),
		ServerName:     serverName,
		NamespacedName: r.McpNamespacedToolName,
		ToolName:       ref.toolNameOf(),
		ExternalCost:   external,
		InternalCost:   internal,
		DurationMs:     durationMs,
		Status:         status,
		StartedAt:      started,
		EndedAt:        ended,
		SessionId:      r.SessionId,
		Metadata:       datatypes.JSON(r.Metadata),
		SyncedAt:       time.Now().UTC(),
	}
}

// insertMcpLogs 幂等落库 + call_count 增量 + last_used_at 回填（同一事务语义内尽力而为）。
func insertMcpLogs(ctx context.Context, mdb *gorm.DB, logs []gateway.McpLog) error {
	tx := mdb.Begin()
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&logs).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("MCP调用日志落库失败: %w", err)
	}
	// call_count 按已归因 server 分组增量（未归因/冲突跳过的行不精确计数，计数器仅展示参考）
	counts := map[int64]int{}
	for i := range logs {
		if logs[i].McpServerId != 0 {
			counts[logs[i].McpServerId]++
		}
	}
	for serverId, n := range counts {
		if err := tx.Model(&gateway.MCPServer{}).
			Where("mcp_server_id = ?", serverId).
			Update("call_count", gorm.Expr("call_count + ?", n)).Error; err != nil {
			logger.WithCtx(ctx).Mod("gateway").Warn(fmt.Sprintf("MCP server %d call_count 增量失败: %v", serverId, err))
		}
	}
	tx.Commit()
	touchMcpKeyLastUsed(ctx, mdb, logs)
	return nil
}

// touchMcpKeyLastUsed 回填 AiKey.last_used_at（MCP 调用同样视为 Key 最近使用）。
func touchMcpKeyLastUsed(ctx context.Context, db *gorm.DB, logs []gateway.McpLog) {
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
			logger.WithCtx(ctx).Mod("gateway").Warn(fmt.Sprintf("MCP回填密钥 %d 最近使用时间失败: %v", id, err))
		}
	}
}

// attributeMcpTool namespaced_name 整串精确匹配 gateway_mcp_tool（唯一索引）→ 联动定位
// 服务器与计费单价；匹配不到工具时按前缀 serverName 反查服务器（split 切名有歧义，
// 仅用"最短候选前缀"兜底），全 miss 返回 nil(未归因,成本 0)。
func attributeMcpTool(db *gorm.DB, namespaced string, cache map[string]*mcpToolRef) *mcpToolRef {
	if namespaced == "" {
		return nil
	}
	if v, ok := cache[namespaced]; ok {
		return v
	}
	ref := &mcpToolRef{serverName: namespaced}
	var tool gateway.MCPTool
	if err := db.Where("namespaced_name = ?", namespaced).First(&tool).Error; err == nil {
		ref.tool = &tool
		ref.toolName = tool.ToolName
		var server gateway.MCPServer
		if err := db.Where("mcp_server_id = ?", tool.McpServerId).First(&server).Error; err == nil {
			ref.server = &server
			ref.mcpServerId = server.McpServerId
			ref.serverName = server.ServerName
		}
	} else {
		// 工具未登记：按 server_name 前缀反查（candidate = 最短前缀命中的 server；
		// serverName 本身禁 '-' 但可含 '_'，前缀歧义时取更长前缀即更具体 server）
		var servers []gateway.MCPServer
		if err := db.Where("server_name <> ''").Find(&servers).Error; err == nil {
			for i := range servers {
				sn := servers[i].ServerName
				if len(sn) > 0 && len(namespaced) > len(sn) && namespaced[:len(sn)] == sn {
					if ref.server == nil || len(sn) > len(ref.server.ServerName) {
						ref.server = &servers[i]
					}
				}
			}
			if ref.server != nil {
				ref.mcpServerId = ref.server.McpServerId
				ref.serverName = ref.server.ServerName
			}
		}
	}
	cache[namespaced] = ref
	return ref
}

// calcMcpCosts per_call 成本自算：工具级计费优先（BillingType 非空即覆盖）→ 服务器级 → 0；
// internal 单价 nil 回落 external（对齐部署 internal_* 语义）；free/无单价 = 0。
// 不采信 LiteLLM spend 列（其按推送的 USD 单价记，平台以 ¥ 自算为准）。
func calcMcpCosts(ref *mcpToolRef) (external, internal float64) {
	if ref == nil {
		return 0, 0
	}
	billing, extPerCall, intPerCall := "", (*float64)(nil), (*float64)(nil)
	switch {
	case ref.tool != nil && ref.tool.BillingType != "":
		billing, extPerCall, intPerCall = ref.tool.BillingType, ref.tool.ExternalCostPerCall, ref.tool.InternalCostPerCall
	case ref.tool != nil && (ref.tool.ExternalCostPerCall != nil || ref.tool.InternalCostPerCall != nil):
		// 工具级未声明 billing_type 但配了单价：视为 per_call 覆盖
		billing, extPerCall, intPerCall = gateway.MCPBillingPerCall, ref.tool.ExternalCostPerCall, ref.tool.InternalCostPerCall
	case ref.server != nil:
		billing, extPerCall, intPerCall = ref.server.BillingType, ref.server.ExternalCostPerCall, ref.server.InternalCostPerCall
	}
	if billing != gateway.MCPBillingPerCall {
		return 0, 0
	}
	if extPerCall != nil {
		external = *extPerCall
	}
	if intPerCall != nil {
		internal = *intPerCall
	} else {
		internal = external
	}
	return external, internal
}

// mcpServerIdOf/toolNameOf nil 安全取值。
func (r *mcpToolRef) mcpServerIdOf() int64 {
	if r == nil {
		return 0
	}
	return r.mcpServerId
}

func (r *mcpToolRef) toolNameOf() string {
	if r == nil {
		return ""
	}
	return r.toolName
}
