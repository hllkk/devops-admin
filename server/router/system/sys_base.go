package system

import (
	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/middleware"
)

type BaseRouter struct{}

func (s *BaseRouter) InitBaseRouter(Router *gin.RouterGroup) (R gin.IRoutes) {
	baseRouter := Router.Group("base")
	{
		baseRouter.POST("login", middleware.SecurityLimit(), baseApi.Login)
		baseRouter.POST("captcha", middleware.SecurityLimit(), baseApi.Captcha)
	}
	return baseRouter
}
