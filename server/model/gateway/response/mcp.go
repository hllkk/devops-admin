package response

import (
	"github.com/hllkk/devops-admin/server/model/common"
	"github.com/hllkk/devops-admin/server/model/gateway"
)

// MCPServerView MCP 服务器出网视图：模型本体(credentials 密文列 json:"-" 不出网) +
// 解密后掩码的 credentials(敏感值掩码，非敏感键值明文)。
type MCPServerView struct {
	gateway.MCPServer
	Credentials map[string]any `json:"credentials"` // 鉴权凭据(敏感值已掩码,编辑回传掩码=保留旧明文)
}

// MCPToolView MCP 工具视图(入参 Schema 按需展示，工具列表页不回显大 Schema 时置空)。
type MCPToolView struct {
	gateway.MCPTool
}

// MCPServerDetail MCP 服务器详情：视图 + 工具列表。
type MCPServerDetail struct {
	MCPServerView
	Tools []MCPToolView `json:"tools"` // 工具列表(refresh-tools 后有值)
}

// MCPConnectConfigView MCP 接入配置(用户侧"查看接入"弹窗)：可直接复制到客户端的
// mcpServers JSON + 工具清单 + 使用说明。keyValue 为当前用户个人主 Key 明文
// (未开通主 Key 时空串，前端提示先开通)。
type MCPConnectConfigView struct {
	Name             string          `json:"name"`             // 展示名
	ServerName       string          `json:"serverName"`       // 路由名(URL 段)
	McpUrl           string          `json:"mcpUrl"`           // 网关接入点 {publicUrl}/{serverName}/mcp
	Description      string          `json:"description"`      // 描述
	Instructions     string          `json:"instructions"`     // 使用说明
	DocumentationUrl string          `json:"documentationUrl"` // 文档地址
	Tools            []MCPToolBrief  `json:"tools"`            // 工具清单(名称+描述)
	Config           map[string]any  `json:"config"`           // 客户端接入配置 JSON
}

// MCPToolBrief 工具简要(接入配置弹窗展示用)。
type MCPToolBrief struct {
	Name        string `json:"name"`        // 工具名
	Description string `json:"description"` // 描述
}

// MCPPublishView MCP 发布设置视图(含 selected/user 模式的可见部门与可见用户回显)。
type MCPPublishView struct {
	McpServerId      int64                   `json:"mcpServerId,string"` // 服务器ID
	IsPublished      bool                    `json:"isPublished"`        // 是否发布
	VisibilityType   string                  `json:"visibilityType"`     // 可见范围(all/selected/user)
	RequiresApproval bool                    `json:"requiresApproval"`   // 接入需审批
	DepartmentIds    common.Int64StringSlice `json:"departmentIds"`      // 可见部门(selected 模式,string[] 雪花id)
	UserIds          common.Int64StringSlice `json:"userIds"`            // 可见用户(user 模式,string[] 雪花id)
}

// AvailableMcpView 可授权 MCP 服务器(精简版，供 Key 授权选择与广场卡片)。
type AvailableMcpView struct {
	McpServerId      int64  `json:"mcpServerId,string"` // 服务器ID
	ServerName       string `json:"serverName"`         // 路由名(授权/LiteLLM 锚点)
	Name             string `json:"name"`               // 展示名
	Description      string `json:"description"`        // 描述
	Category         string `json:"category"`           // 分类
	Author           string `json:"author"`             // 提供方
	IconUrl          string `json:"iconUrl"`            // 图标
	DocumentationUrl string `json:"documentationUrl"`   // 文档地址
	RequiresApproval bool   `json:"requiresApproval"`   // 接入需审批
	ToolCount        int64  `json:"toolCount"`          // 工具数
}
