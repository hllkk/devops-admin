package system

import (
	"github.com/gin-gonic/gin"
)

type BaseRouter struct{}

// InitBaseRouter 注册 auth 路由：public 组挂 login/code/refreshToken/logout，
// private 组挂 getUserInfo（需鉴权）。
func (s *BaseRouter) InitBaseRouter(public, private *gin.RouterGroup) {
	pub := public.Group("auth")
	{
		pub.POST("login", baseApi.Login)
		pub.POST("code", baseApi.Captcha)
		pub.POST("refreshToken", baseApi.RefreshToken)
		pub.POST("logout", baseApi.Logout)
	}
	// private 组的 /auth/getUserInfo（见权限基座闭环 Step 5）
	pri := private.Group("auth")
	{
		pri.GET("getUserInfo", baseApi.GetUserInfo)
	}
}
