package system

import (
	"github.com/gin-gonic/gin"
)

// RouteRouter 路由下发(/route)。前端 Soybean 静态/动态模式均由此取路由。
// getConstantRoutes 挂 PublicGroup(登录前即可取,constant 路由含登录页/404/init 本就该公开);
// getUserRoutes/isRouteExist 挂 PrivateGroup(需登录态,按当前用户角色过滤)。
type RouteRouter struct{}

func (s *RouteRouter) InitRouteRouter(privateRouter, publicRouter *gin.RouterGroup) {
	{
		pub := publicRouter.Group("route")
		pub.GET("getConstantRoutes", routeApi.GetConstantRoutes) // 常量路由(免鉴权 / 前端静态 fallback)
	}
	{
		priv := privateRouter.Group("route")
		priv.GET("getUserRoutes", routeApi.GetUserRoutes) // 用户动态路由(dynamic 模式,需登录)
		priv.GET("isRouteExist", routeApi.IsRouteExist)   // 路由是否存在(前端守卫用,需登录)
	}
}
