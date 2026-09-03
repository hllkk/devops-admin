package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"

	"github.com/hllkk/devops-admin/server/model/common"
)

// MCPServerSearch MCP 服务器分页查询(对齐前端 GET /gateway/mcp/list，query 传输)。
// name 模糊匹配展示名/路由名；category/healthStatus 精确；isActive/isPublished
// 指针区分未传与 false(query 空串 *bool 绑 false 的坑由 NormalizeEmptyBoolQuery 归一)。
type MCPServerSearch struct {
	commonReq.PageInfo
	Name         string `json:"name" form:"name"`                 // 展示名/路由名(模糊)
	Category     string `json:"category" form:"category"`         // 分类(精确)
	IsActive     *bool  `json:"isActive" form:"isActive"`         // 是否启用(nil=不限)
	IsPublished  *bool  `json:"isPublished" form:"isPublished"`   // 是否发布(nil=不限)
	HealthStatus string `json:"healthStatus" form:"healthStatus"` // 健康状态(精确)
}

// MCPServerOperateParams MCP 服务器新增/修改(对齐前端 POST/PUT /gateway/mcp)。
// create 时 mcpServerId 为空(雪花主键回调填充)；update 时必填。serverName 拒绝改名
// (LiteLLM 路由键)；credentials 明文入网、密文落库——编辑回传掩码值=保留旧明文；
// http/sse 型 url 必填，stdio 型 command 必填(白名单运行时)、url 留空、authType 恒 none，
// credentials 此时承载 env 变量(env 即凭据，同套加密/掩码/合并语义)。
type MCPServerOperateParams struct {
	McpServerId         int64          `json:"mcpServerId,string" form:"mcpServerId"` // 服务器ID(新增为空)
	Name                string         `json:"name" form:"name"`                      // 展示名称
	ServerName          string         `json:"serverName" form:"serverName"`          // LiteLLM 路由名(唯一,禁-)
	Url                 string         `json:"url" form:"url"`                        // MCP 端点(stdio 型留空)
	Transport           string         `json:"transport" form:"transport"`            // 传输协议(sse/streamable_http/stdio)
	Command             string         `json:"command" form:"command"`                // stdio 启动命令(白名单运行时)
	Args                []string       `json:"args"`                                  // stdio 启动参数
	AuthType            string         `json:"authType" form:"authType"`              // 鉴权方式(none/api_key/bearer_token;stdio 恒 none)
	Credentials         map[string]any `json:"credentials"`                           // 鉴权凭据/stdio env变量(明文入网,掩码回传=保留旧值)
	Description         string         `json:"description" form:"description"`        // 描述
	Instructions        string         `json:"instructions" form:"instructions"`      // 使用说明
	Category            string         `json:"category" form:"category"`              // 分类
	Author              string         `json:"author" form:"author"`                  // 提供方
	IconUrl             string         `json:"iconUrl" form:"iconUrl"`                // 图标URL
	DocumentationUrl    string         `json:"documentationUrl" form:"documentationUrl"` // 文档地址
	BillingType         string         `json:"billingType" form:"billingType"`        // 计费类型(per_call/free)
	ExternalCostPerCall *float64       `json:"externalCostPerCall"`                   // 单次调用价(¥,nil=免费)
	InternalCostPerCall *float64       `json:"internalCostPerCall"`                   // 单次调用内部结算价(¥,nil=同外部价)
	IsActive            *bool          `json:"isActive"`                              // 是否启用(nil=不改)
}

// MCPToolBillingParams 工具级计费配置(对齐前端 PUT /gateway/mcp/tool/:toolId/billing)。
// billingType 空/externalCostPerCall nil = 清为继承服务器默认。
type MCPToolBillingParams struct {
	BillingType         string   `json:"billingType"`                  // 计费类型(per_call/free/空=继承)
	ExternalCostPerCall *float64 `json:"externalCostPerCall"`          // 单次调用价(¥,nil=继承)
	InternalCostPerCall *float64 `json:"internalCostPerCall"`          // 单次调用内部价(¥,nil=继承/同外部价)
}

// MCPPublishParams MCP 发布设置(对齐前端 PUT /gateway/mcp/publish)。
// visibilityType=selected 且 isPublished=true 时 departmentIds 必填；=user 时 userIds 必填；
// =mixed 时两列表合计至少一项(两张投影行并集)。
// ID 列表用 Int64StringSlice：前端 IdType 混 string/number，元素级兼容反序列化。
type MCPPublishParams struct {
	McpServerId      int64                   `json:"mcpServerId,string" form:"mcpServerId"` // 服务器ID
	IsPublished      bool                    `json:"isPublished" form:"isPublished"`        // 是否发布
	VisibilityType   string                  `json:"visibilityType" form:"visibilityType"`  // 可见范围(all/selected/user/mixed)
	RequiresApproval bool                    `json:"requiresApproval" form:"requiresApproval"` // 接入需审批
	DepartmentIds    common.Int64StringSlice `json:"departmentIds" form:"departmentIds"`    // 可见部门(selected/mixed 模式)
	UserIds          common.Int64StringSlice `json:"userIds" form:"userIds"`                // 可见用户(user/mixed 模式)
}
