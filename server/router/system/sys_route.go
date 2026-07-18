package system

import (
	"github.com/gin-gonic/gin"
)

// RouteRouter 路由下发(/route,挂 PrivateGroup)
type RouteRouter struct{}

func (s *RouteRouter) InitRouteRouter(Router *gin.RouterGroup) {
	routeRouter := Router.Group("route")
	{
		routeRouter.GET("getConstantRoutes", baseApi.GetConstantRoutes) // 常量路由(static 模式)
	}
}
