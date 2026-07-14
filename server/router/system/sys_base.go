package system

import (
	"github.com/gin-gonic/gin"
)

type BaseRouter struct{}

// InitBaseRouter 注册 auth 路由：public 组挂 login/code/refreshToken/logout，
// private 组挂 getUserInfo（需鉴权）。
// public 或 private 为 nil 时跳过对应组的注册（ModuleRouter 分组调用时使用）。
func (s *BaseRouter) InitBaseRouter(public, private *gin.RouterGroup) {
	if public != nil {
		pub := public.Group("auth")
		{
			pub.POST("login", baseApi.Login)
			pub.POST("code", baseApi.Captcha)
			pub.POST("refreshToken", baseApi.RefreshToken)
			pub.POST("logout", baseApi.Logout)
		}
	}
	if private != nil {
		pri := private.Group("auth")
		{
			pri.GET("getUserInfo", baseApi.GetUserInfo)
		}
	}
}
