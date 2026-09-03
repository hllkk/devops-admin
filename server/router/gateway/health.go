package gateway

import "github.com/gin-gonic/gin"

// HealthRouter 健康检查路由(对齐前端 /gateway/health/* 资源)
type HealthRouter struct{}

// InitHealthRouter 挂在 PrivateGroup，鉴权/操作日志由该组全局中间件统一处理。
// 接口权限走菜单 api_prefix(route.ai-audit_health)：user 角色不授，管理员/运维视角。
func (r *HealthRouter) InitHealthRouter(Router *gin.RouterGroup) {
	g := Router.Group("gateway/health")
	{
		g.GET("summary", healthApi.GetHealthSummary)                 // 四卡汇总+明细
		g.POST("check-deployments", healthApi.HealthCheckDeployments) // 手动巡检模型部署
	}
}
