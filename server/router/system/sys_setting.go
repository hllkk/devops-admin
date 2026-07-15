package system

import "github.com/gin-gonic/gin"

// SettingRouter 系统设置路由：公开路由供登录页读取展示配置，管理路由供管理员读写完整配置。
type SettingRouter struct{}

// InitSettingPublicRouter 公开路由（无需认证，登录页使用）。
func (r *SettingRouter) InitSettingPublicRouter(Router *gin.RouterGroup) {
	g := Router.Group("system/setting")
	{
		g.GET("/public", settingApi.GetPublicSystemSettings)
	}
}

// InitSettingRouter 管理路由（挂载于 admin 组：JWT + RequireAdmin）。
func (r *SettingRouter) InitSettingRouter(Router *gin.RouterGroup) {
	g := Router.Group("system/setting")
	{
		g.GET("", settingApi.GetSystemSettings)
		g.PUT("", settingApi.UpdateSystemSettings)
	}
}
