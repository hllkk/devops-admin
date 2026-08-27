package gateway

import "github.com/gin-gonic/gin"

// DashboardRouter 看板路由(对齐前端 /gateway/dashboard/* 资源)
type DashboardRouter struct{}

// InitDashboardRouter 挂在 PrivateGroup，鉴权/操作日志/数据权限由该组全局中间件统一处理。
func (r *DashboardRouter) InitDashboardRouter(Router *gin.RouterGroup) {
	g := Router.Group("gateway/dashboard")
	{
		g.GET("overview", dashboardApi.GetOverview)    // 看板总览
		g.GET("trend", dashboardApi.GetTrend)           // 成本趋势
		g.GET("top", dashboardApi.GetTop)              // 成本Top10
		g.GET("budget", dashboardApi.GetBudget)        // 预算执行率
		g.GET("balance-summary", providerBalanceApi.GetBalanceSummary) // 跨供应商套餐余量汇总(旁路只读,超管)
		g.POST("aggregate", dashboardApi.AggregateUsage) // 手动触发聚合
	}
}
