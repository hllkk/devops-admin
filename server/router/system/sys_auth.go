package system

import (
	"github.com/gin-gonic/gin"
)

// AuthRouter 鉴权路由组(/auth,挂 PrivateGroup,需登录)
type AuthRouter struct{}

func (s *AuthRouter) InitAuthRouter(Router *gin.RouterGroup) {
	authRouter := Router.Group("auth")
	{
		authRouter.GET("getUserInfo", baseApi.GetUserInfo)    // 当前登录用户信息(roles/permissions)
		authRouter.POST("logout", baseApi.Logout)             // 退出登录
		authRouter.POST("refreshToken", baseApi.RefreshToken) // 刷新 token
	}
}
