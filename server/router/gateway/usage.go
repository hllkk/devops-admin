package gateway

import "github.com/gin-gonic/gin"

// UsageRouter 用量统计路由(对齐前端 /gateway/usage/* 资源)
type UsageRouter struct{}

// InitUsageRouter 挂在 PrivateGroup，鉴权/操作日志/数据权限由该组全局中间件统一处理。
// 静态段(sync/reconcile/list)书写在 :id 之前注册。
func (r *UsageRouter) InitUsageRouter(Router *gin.RouterGroup) {
	g := Router.Group("gateway/usage")
	{
		g.POST("sync", usageApi.SyncLLMLogs)            // 手动触发用量回流
		g.POST("reconcile", usageApi.ReconcileLLMLogs)  // 手动触发对账回灌
		g.GET("list", usageApi.GetUsageLogList)         // 分页查用量日志
	}
}
