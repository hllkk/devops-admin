package gateway

import (
	"time"

	"gorm.io/datatypes"

	"github.com/hllkk/devops-admin/server/global"
)

// MCPServer MCP 服务器（AI 市场 P2：企业内 MCP 工具统一注册/发布/授权）。
// LiteLLM MCP 网关是管理中心：server 经 /v1/mcp/server 同步，调用走 /{server_name}/mcp；
// 本表存管理元数据。server_name 是 LiteLLM 路由键（唯一、禁 '-'，拒绝改名，对齐
// credential_name）；litellm_server_id 是 LiteLLM 侧 server_id（归因锚点，永不重置）。
// credentials 为 AES-256-GCM 密文 JSON（json:"-" 不出网，规避 AIHelms 明文落库+回显坑）；
// 计费口径与模型部署一致：平台存 ¥，推送 LiteLLM 时按汇率换算 USD 写入
// mcp_info.mcp_server_cost_info（usage 回流的 MCP 成本统计留 P3）。
type MCPServer struct {
	global.OPS_AUDIT_MODEL
	McpServerId      int64      `json:"mcpServerId,string" gorm:"primarykey;comment:MCP服务器ID(雪花)"`                     // MCP服务器ID(雪花)
	Name             string     `json:"name" gorm:"size:128;comment:展示名称"`                                              // 展示名称
	ServerName       string     `json:"serverName" gorm:"size:128;comment:LiteLLM路由名(唯一,禁-)"`                          // LiteLLM 路由名/键
	Url              string     `json:"url" gorm:"type:text;comment:MCP端点URL"`                                          // MCP 端点
	Transport        string     `json:"transport" gorm:"size:20;default:streamable_http;comment:传输协议(sse/streamable_http)"` // 传输协议
	AuthType         string     `json:"authType" gorm:"size:20;default:none;comment:鉴权方式(none/api_key/bearer_token)"`       // 鉴权方式
	Credentials      string     `json:"-" gorm:"type:text;comment:鉴权凭据(AES-256-GCM密文JSON,不序列化出网)"`                    // 鉴权凭据(密文)
	Description      string     `json:"description" gorm:"type:text;comment:描述"`                                        // 描述
	Instructions     string     `json:"instructions" gorm:"type:text;comment:使用说明(接入页展示)"`                              // 使用说明
	Category         string     `json:"category" gorm:"size:50;default:general;comment:分类"`                             // 分类(广场筛选)
	Author           string     `json:"author" gorm:"size:128;comment:提供方"`                                             // 提供方/作者
	IconUrl          string     `json:"iconUrl" gorm:"size:500;comment:图标URL"`                                          // 图标
	DocumentationUrl string     `json:"documentationUrl" gorm:"size:500;comment:文档地址"`                                  // 文档地址
	BillingType      string     `json:"billingType" gorm:"size:20;default:free;comment:计费类型(per_call/free)"`            // 计费类型
	ExternalCostPerCall *float64 `json:"externalCostPerCall" gorm:"type:numeric(12,6);comment:单次调用外部价(¥,nil=免费)"`        // 单次调用价
	InternalCostPerCall *float64 `json:"internalCostPerCall" gorm:"type:numeric(12,6);comment:单次调用内部价(¥,nil=同外部价)"`     // 单次调用内部结算价(nil回落external,对齐部署internal_*语义)
	CallCount        int64      `json:"callCount" gorm:"default:0;comment:累计调用次数(回流任务按server增量维护)"`               // 累计调用次数
	IsActive         bool       `json:"isActive" gorm:"default:true;comment:是否启用"`                                      // 是否启用
	IsPublished      bool       `json:"isPublished" gorm:"default:false;comment:是否发布到用户端"`                              // 是否发布
	VisibilityType   string     `json:"visibilityType" gorm:"size:20;default:all;comment:可见范围(all/selected/user)"`      // 可见范围(与模型三档同口径)
	RequiresApproval bool       `json:"requiresApproval" gorm:"default:false;comment:接入是否需审批"`                          // 接入需审批
	HealthStatus     string     `json:"healthStatus" gorm:"size:20;default:unknown;comment:健康状态(unknown/healthy/unhealthy)"` // 健康状态
	LastHealthCheck  *time.Time `json:"lastHealthCheck" gorm:"comment:最近健康检查(nil=未检查)"`                                 // 最近健康检查
	HealthCheckError string     `json:"healthCheckError" gorm:"type:text;comment:最近健康检查错误"`                             // 健康检查错误
	LitellmServerId  string     `json:"litellmServerId" gorm:"size:100;index;comment:LiteLLM侧server_id(归因锚点,永不重置)"`     // LiteLLM server_id
	LitellmSynced    bool       `json:"litellmSynced" gorm:"default:false;comment:是否已同步LiteLLM"`                        // 同步状态
	LitellmSyncError string     `json:"litellmSyncError" gorm:"type:text;comment:最近同步错误"`                               // 同步错误
	ToolCount        int64      `json:"toolCount" gorm:"-"`                                                             // 工具数(service填充,不入库)
}

// MCP 传输协议(LiteLLM 只支持 sse/http，streamable_http 归一映射 http 下发)
const (
	MCPTransportSse            = "sse"             // SSE(服务端推送)
	MCPTransportStreamableHttp = "streamable_http" // Streamable HTTP(默认)
)

// MCP 鉴权方式(LiteLLM MCPAuth 枚举的常用子集，credentials 存 {"auth_value": "..."})
const (
	MCPAuthNone        = "none"         // 无鉴权
	MCPAuthApiKey      = "api_key"      // API Key(x-api-key)
	MCPAuthBearerToken = "bearer_token" // Bearer Token
)

// MCP 计费类型(per_call 与模型部署同语义；免费不写 cost_info)
const (
	MCPBillingPerCall = "per_call" // 按次计费
	MCPBillingFree    = "free"     // 免费
)

// MCP 健康状态
const (
	MCPHealthUnknown   = "unknown"   // 未检查
	MCPHealthHealthy   = "healthy"   // 可达
	MCPHealthUnhealthy = "unhealthy" // 不可达
)

func (MCPServer) TableName() string {
	return "gateway_mcp_server"
}

// MCPTool MCP 服务器工具（refresh-tools 从远端拉全量重建；按 tool_name 保留已有计费配置）。
// namespaced_name 是网关侧工具全名({server_name}_{tool_name})，调用日志/计费归因锚点。
type MCPTool struct {
	global.OPS_AUDIT_MODEL
	McpToolId      int64   `json:"mcpToolId,string" gorm:"primarykey;comment:工具ID(雪花)"`                                  // 工具ID(雪花)
	McpServerId    int64   `json:"mcpServerId,string" gorm:"uniqueIndex:idx_gateway_mcp_tool;comment:所属服务器ID"`            // 所属服务器
	ToolName       string  `json:"toolName" gorm:"uniqueIndex:idx_gateway_mcp_tool;size:200;comment:工具原始名"`               // 工具原始名
	NamespacedName string  `json:"namespacedName" gorm:"size:300;comment:网关全名(serverName_toolName)"`                     // 网关全名
	DisplayName    string  `json:"displayName" gorm:"size:200;comment:展示名"`                                              // 展示名
	Description    string  `json:"description" gorm:"type:text;comment:描述"`                                              // 描述
	InputSchema    datatypes.JSON `json:"inputSchema" gorm:"type:jsonb;comment:入参Schema" swaggertype:"object"`             // 入参 Schema
	BillingType    string  `json:"billingType" gorm:"size:20;comment:计费类型(空=继承服务器)"`                                     // 工具级计费类型
	ExternalCostPerCall *float64 `json:"externalCostPerCall" gorm:"type:numeric(12,6);comment:单次调用价(¥,nil=继承服务器)"`         // 工具级单次价
	InternalCostPerCall *float64 `json:"internalCostPerCall" gorm:"type:numeric(12,6);comment:单次调用内部价(¥,nil=继承服务器)"`      // 工具级内部结算价(nil=继承服务器/同外部价)
}

func (MCPTool) TableName() string {
	return "gateway_mcp_tool"
}

// MCPVisibility MCP 部门可见性(发布投影表，visibility_type=selected 时使用)。
// 与 gateway_model_visibility 同口径：非业务实体，重建时物理删除(Unscoped)，
// 软删行会占住唯一索引挡住同组合重新发布。
type MCPVisibility struct {
	global.OPS_MODEL
	McpServerId  int64 `json:"mcpServerId,string" gorm:"uniqueIndex:idx_gateway_mcp_visibility;comment:关联MCP服务器ID"`                            // 关联MCP服务器
	DepartmentId int64 `json:"departmentId,string" gorm:"uniqueIndex:idx_gateway_mcp_visibility;comment:关联部门ID(sys_departments.dept_id)"` // 关联部门
}

func (MCPVisibility) TableName() string {
	return "gateway_mcp_visibility"
}

// MCPVisibilityUser MCP 用户可见性(发布投影表，visibility_type=user 时使用)。
// 与 gateway_model_visibility_user 同口径：非业务实体，重建时物理删除(同唯一索引理由)。
type MCPVisibilityUser struct {
	global.OPS_MODEL
	McpServerId int64 `json:"mcpServerId,string" gorm:"uniqueIndex:idx_gateway_mcp_visibility_user;comment:关联MCP服务器ID"`           // 关联MCP服务器
	UserId      int64 `json:"userId,string" gorm:"uniqueIndex:idx_gateway_mcp_visibility_user;comment:关联用户ID(sys_users.id)"` // 关联用户
}

func (MCPVisibilityUser) TableName() string {
	return "gateway_mcp_visibility_user"
}
