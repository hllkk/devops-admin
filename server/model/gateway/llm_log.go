package gateway

import (
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"gorm.io/datatypes"
)

// LlmLog LLM 调用用量日志（从 LiteLLM_SpendLogs 回流 + 归因 + 成本重算）。
// request_id 唯一约束 + ON CONFLICT DO NOTHING 幂等；归因字段(ai_key_id/user_id/deployment_id)
// 可为 0（master key/default_user_id 跳过或归因失败）；成本本地重算不信任 LiteLLM 的 spend 列。
// started_at/ended_at 存 UTC（日桶按业务时区切是聚合层的事）。
type LlmLog struct {
	global.OPS_BASE
	LogId              int64          `json:"logId,string" gorm:"primarykey;comment:日志ID(雪花)"` // 日志ID(雪花)
	RequestId          string         `json:"requestId" gorm:"size:500;uniqueIndex;comment:请求ID(幂等键,ON CONFLICT DO NOTHING)"` // LiteLLM request_id
	UserId             int64          `json:"userId,string" gorm:"index;comment:归因用户ID(0=未归因)"`   // 归因用户(纯逻辑关联 sys_users)
	AiKeyId            int64          `json:"aiKeyId,string" gorm:"index;comment:归因密钥ID(0=未归因)"`  // 归因AiKey
	DeploymentId       int64          `json:"deploymentId,string" gorm:"index;comment:归因部署ID(0=未归因)"` // 归因ModelDeployment
	Model              string         `json:"model" gorm:"size:128;index;comment:请求时模型名"`       // 请求时 model_name(含前缀/变体)
	Provider           string         `json:"provider" gorm:"size:50;comment:供应商类型"`           // custom_llm_provider
	CallType           string         `json:"callType" gorm:"size:50;comment:调用类型"`           // call_type
	PromptTokens       int            `json:"promptTokens" gorm:"default:0;comment:输入token"`     // prompt_tokens
	CompletionTokens   int            `json:"completionTokens" gorm:"default:0;comment:输出token"`   // completion_tokens
	TotalTokens        int            `json:"totalTokens" gorm:"default:0;comment:总token"`        // total_tokens
	CacheReadTokens    int            `json:"cacheReadTokens" gorm:"default:0;comment:缓存读token"`   // cache_read
	CacheCreationTokens int           `json:"cacheCreationTokens" gorm:"default:0;comment:缓存创建token"` // cache_creation
	ExternalCost       float64        `json:"externalCost" gorm:"type:numeric(12,6);default:0;comment:外部成本(¥,对客定价重算)"` // external_cost
	InternalCost       float64        `json:"internalCost" gorm:"type:numeric(12,6);default:0;comment:内部成本(¥,P1同external,P3细化对内)"` // internal_cost
	DurationMs         int            `json:"durationMs" gorm:"comment:耗时(毫秒)"`             // duration_ms
	StartedAt          time.Time      `json:"startedAt" gorm:"index;comment:开始时间(UTC)"`       // started_at(UTC)
	EndedAt            time.Time      `json:"endedAt" gorm:"comment:结束时间(UTC)"`               // ended_at(UTC)
	SessionId          string         `json:"sessionId" gorm:"size:100;comment:会话ID"`           // session_id
	Metadata           datatypes.JSON `json:"metadata" gorm:"type:jsonb;comment:原始元数据" swaggertype:"object"` // metadata
	SyncedAt           time.Time      `json:"syncedAt" gorm:"comment:回流时间"`                // synced_at
}

func (LlmLog) TableName() string {
	return "gateway_llm_log"
}

// SyncState 用量回流游标 KV（复合游标 last_sync_at + last_request_id）。
// 聚合表无独立状态机，游标只存在原始同步层（约定）。
type SyncState struct {
	Key            string    `json:"key" gorm:"primarykey;size:64;comment:游标键(llm_logs)"` // "llm_logs"
	LastSyncAt     time.Time `json:"lastSyncAt" gorm:"comment:最后同步时间(UTC,COALESCE(endTime,startTime))"` // 游标第一列
	LastRequestId  string    `json:"lastRequestId" gorm:"size:500;comment:最后同步request_id(游标第二列)"` // 游标第二列
	UpdatedAt      time.Time `json:"updatedAt" gorm:"autoUpdateTime;comment:更新时间"`       // 自动更新
}

func (SyncState) TableName() string {
	return "gateway_sync_state"
}

// LiteLLMSpendLog LiteLLM 的 spend 日志表只读映射（不 AutoMigrate，表由 LiteLLM 管理）。
// 字段对齐 LiteLLM 1.98 的 LiteLLM_SpendLogs 列；startTime/endTime 是 timestamp without time zone，
// LiteLLM 落 naive UTC。Go 读取按 UTC 解释；SQL 比较侧显式 AT TIME ZONE 'UTC'（见 fetchSpendBatch），
// 连接会话时区不参与正确性——spend-dsn 建议仍配 TimeZone=UTC。
type LiteLLMSpendLog struct {
	RequestId           string         `gorm:"column:request_id"`
	CallType            string         `gorm:"column:call_type"`
	ApiKey              string         `gorm:"column:api_key"`
	Spend               float64        `gorm:"column:spend"`
	TotalTokens         int            `gorm:"column:total_tokens"`
	PromptTokens        int            `gorm:"column:prompt_tokens"`
	CompletionTokens   int            `gorm:"column:completion_tokens"`
	StartTime           time.Time      `gorm:"column:startTime"`
	EndTime             time.Time      `gorm:"column:endTime"`
	Model               string         `gorm:"column:model"`
	ModelId             string         `gorm:"column:model_id"`
	ModelGroup          string         `gorm:"column:model_group"`
	CustomLlmProvider   string         `gorm:"column:custom_llm_provider"`
	ApiBase             string         `gorm:"column:api_base"`
	User                string         `gorm:"column:user"`
	McpNamespacedToolName string       `gorm:"column:mcp_namespaced_tool_name"` // 非空=MCP 调用行(回流层与 LLM 行互斥分流)
	Status              string         `gorm:"column:status"`
	Metadata            datatypes.JSON `gorm:"column:metadata"`
	SessionId           string         `gorm:"column:session_id"`
}

func (LiteLLMSpendLog) TableName() string {
	return `"LiteLLM_SpendLogs"`
}

// SpendLogSelectColumns SpendLogs 查询显式列清单（与上方模型字段一一对应，模型加列须同步）。
// 表内 messages/response/proxy_server_request 等巨型 jsonb 不在模型映射内，SELECT * 会把
// 它们整体拖回（实测行均 ~157KB），单轮千行即 20MB+ IO 放大；查询一律显式选映射列。
func (LiteLLMSpendLog) SelectColumns() string {
	return `"request_id","call_type","api_key","spend","total_tokens","prompt_tokens","completion_tokens","startTime","endTime","model","model_id","model_group","custom_llm_provider","api_base","user","mcp_namespaced_tool_name","status","metadata","session_id"`
}
