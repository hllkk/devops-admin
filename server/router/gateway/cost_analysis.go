package gateway

import "github.com/gin-gonic/gin"

// CostAnalysisRouter 成本分析路由(对齐前端 /gateway/cost/* 资源)
type CostAnalysisRouter struct{}

// InitCostAnalysisRouter 挂在 PrivateGroup，鉴权/操作日志由该组全局中间件统一处理。
// 接口权限走菜单 api_prefix(route.ai-audit_cost)：user 角色不授，管理员视角。
func (r *CostAnalysisRouter) InitCostAnalysisRouter(Router *gin.RouterGroup) {
	g := Router.Group("gateway/cost")
	{
		g.GET("overview", costAnalysisApi.GetCostOverview)             // KPI+趋势(LLM+MCP 合并口径)
		g.GET("detail", costAnalysisApi.GetCostDetail)                 // 七维明细(分页,含 MCP 维)
		g.GET("detail/scope-users", costAnalysisApi.GetCostScopeUsers) // 部门下钻成员
		g.GET("detail/mcp-tools", costAnalysisApi.GetCostMcpTools)     // MCP 维工具子表
	}
}
