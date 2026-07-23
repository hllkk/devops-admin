package system

import "github.com/gin-gonic/gin"

// SocialRouter 第三方账号绑定/社交登录路由(对齐前端契约)
//
// 公开组(免鉴权,登录页可调):GET /auth/binding/:source、POST /auth/social/callback
// 私有组(需 JWTAuth):    GET /system/social/list、DELETE /auth/unlock/:id
type SocialRouter struct{}

func (s *SocialRouter) InitSocialRouter(Router, PublicRouter *gin.RouterGroup) {
	// 公开:授权 URL 获取 + OAuth 回调(交换 code + 绑定/登录)
	pub := PublicRouter.Group("auth")
	{
		pub.GET("/binding/:source", socialApi.GetAuthURL)
		pub.POST("/social/callback", socialApi.Callback)
	}
	// 私有:已绑定列表 + 解绑
	{
		Router.GET("/system/social/list", socialApi.List)
		Router.DELETE("/auth/unlock/:id", socialApi.Unbind)
	}
}
