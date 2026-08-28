package gateway

import "github.com/gin-gonic/gin"

// MCPRouter MCP 服务器管理路由(对齐前端 /gateway/mcp/* 资源)
type MCPRouter struct{}

// InitMCPRouter 挂在 PrivateGroup，鉴权/操作日志/数据权限由该组全局中间件统一处理。
// active 与 connect-config 为用户侧接口(casbin 登录白名单，见 middleware/casbin_rbac.go)。
func (m *MCPRouter) InitMCPRouter(Router *gin.RouterGroup) {
	r := Router.Group("gateway/mcp")
	{
		r.GET("list", mcpApi.GetMCPServerList)                  // 分页获取服务器列表
		r.GET(":id", mcpApi.GetMCPServer)                       // 服务器详情(含工具)
		r.POST("", mcpApi.CreateMCPServer)                      // 注册服务器
		r.PUT("", mcpApi.UpdateMCPServer)                       // 修改服务器
		r.DELETE(":ids", mcpApi.DeleteMCPServers)               // 批量删除
		r.PUT("publish", mcpApi.PublishMCPServer)               // 发布设置(三档可见性+审批)
		r.GET("publish/:id", mcpApi.GetMCPPublish)              // 发布设置回显(含可见部门/用户)
		r.POST(":id/refresh-tools", mcpApi.RefreshMCPTools)     // 刷新工具列表
		r.POST(":id/health-check", mcpApi.HealthCheckMCPServer) // 健康检查
		r.PUT("tool/:toolId/billing", mcpApi.UpdateMCPToolBilling) // 工具计费
		// 用户侧(广场/接入)：登录白名单，不经菜单授权
		r.GET("available", mcpApi.GetAvailableMcps)                     // 管理端授权下拉
		r.GET("active", mcpApi.GetActiveMcps)                           // 用户侧可见列表(广场)
		r.GET("connect-config/:id", mcpApi.GetMCPConnectConfig)         // 接入配置(用户)
	}
}
