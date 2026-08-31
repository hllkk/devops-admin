package gateway

import (
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"gorm.io/datatypes"
)

// McpLog MCP 调用日志（P3：从 LiteLLM_SpendLogs 的 mcp_namespaced_tool_name 非空行回流）。
// 与 LlmLog 同源同风格：request_id 唯一约束 + ON CONFLICT DO NOTHING 幂等；归因字段
// (ai_key_id/user_id/mcp_server_id)可为 0（未归因/平台无此 server）；成本平台自算
// per_call（工具级优先→server 级→0），不采信 LiteLLM spend 列。started_at/ended_at 存 UTC。
// namespaced_name 整串精确匹配 gateway_mcp_tool.namespaced_name 归因（不 split 切名，
// 规避 serverName 含 '_' 时切错位的歧义）。
type McpLog struct {
	global.OPS_BASE
	LogId          int64          `json:"logId,string" gorm:"primarykey;comment:日志ID(雪花)"`       // 日志ID(雪花)
	RequestId      string         `json:"requestId" gorm:"size:500;uniqueIndex;comment:请求ID(幂等键,ON CONFLICT DO NOTHING)"` // LiteLLM request_id
	UserId         int64          `json:"userId,string" gorm:"index;comment:归因用户ID(0=未归因)"`  // 归因用户
	AiKeyId        int64          `json:"aiKeyId,string" gorm:"index;comment:归因密钥ID(0=未归因)"` // 归因AiKey
	McpServerId    int64          `json:"mcpServerId,string" gorm:"index;comment:归因MCP服务器ID(0=未匹配)"` // 归因MCPServer
	ServerName     string         `json:"serverName" gorm:"size:300;comment:服务器展示名(未匹配时为原始名)"` // 服务器名
	NamespacedName string         `json:"namespacedName" gorm:"size:400;index;comment:网关工具全名(LiteLLM原始锚点)"` // mcp_namespaced_tool_name 原始值
	ToolName       string         `json:"toolName" gorm:"size:200;comment:工具名(匹配MCPTool后回填,未匹配为空)"` // 工具名
	ExternalCost   float64        `json:"externalCost" gorm:"type:numeric(12,6);default:0;comment:外部成本(¥,per_call自算)"` // external_cost
	InternalCost   float64        `json:"internalCost" gorm:"type:numeric(12,6);default:0;comment:内部成本(¥,nil单价回落external)"` // internal_cost
	DurationMs     int            `json:"durationMs" gorm:"comment:耗时(毫秒)"`               // duration_ms
	Status         string         `json:"status" gorm:"size:20;default:success;comment:调用状态(success/error)"` // LiteLLM status
	StartedAt      time.Time      `json:"startedAt" gorm:"index;comment:开始时间(UTC)"`        // started_at(UTC)
	EndedAt        time.Time      `json:"endedAt" gorm:"comment:结束时间(UTC)"`                // ended_at(UTC)
	SessionId      string         `json:"sessionId" gorm:"size:100;comment:会话ID"`           // session_id
	Metadata       datatypes.JSON `json:"metadata" gorm:"type:jsonb;comment:原始元数据" swaggertype:"object"` // metadata
	SyncedAt       time.Time      `json:"syncedAt" gorm:"comment:回流时间"`                  // synced_at
}

func (McpLog) TableName() string {
	return "gateway_mcp_log"
}

// MCP 调用状态
const (
	McpCallStatusSuccess = "success" // 成功
	McpCallStatusError   = "error"   // 失败
)
