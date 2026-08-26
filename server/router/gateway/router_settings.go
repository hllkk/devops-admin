package gateway

import "github.com/gin-gonic/gin"

// RouterSettingsRouter 全局路由策略路由(GET/PUT /gateway/router/settings)。
// 挂在 PrivateGroup，鉴权/操作日志/数据权限由该组全局中间件统一处理。
type RouterSettingsRouter struct{}

func (r *RouterSettingsRouter) InitRouterSettingsRouter(Router *gin.RouterGroup) {
	g := Router.Group("gateway/router/settings")
	{
		g.GET("", routerSettingsApi.GetRouterSettings)
		g.PUT("", routerSettingsApi.UpdateRouterSettings)
	}
}
