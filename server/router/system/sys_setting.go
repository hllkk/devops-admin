package system

import "github.com/gin-gonic/gin"

// SettingRouter 系统设置路由(对齐前端 /system/setting)
type SettingRouter struct{}

// InitSettingRouter 私有路由挂在 PrivateGroup(管理员,鉴权+操作日志由该组全局中间件统一处理);
// 公开设置 GET system/setting/public 挂在 PublicGroup(登录页,免鉴权,脱敏:仅系统信息+验证码开关)
func (s *SettingRouter) InitSettingRouter(Router, PublicRouter *gin.RouterGroup) {
	settingRouter := Router.Group("system/setting")
	{
		settingRouter.GET("", settingApi.GetSetting)    // 获取系统设置(聚合 general+security+ldap+notify)
		settingRouter.PUT("", settingApi.UpdateSetting) // 更新系统设置(聚合保存)
	}
	notifyRouter := Router.Group("system/setting/notify")
	{
		notifyRouter.POST("test-email", settingApi.TestEmail) // 发送测试邮件(使用当前表单值)
	}
	publicSetting := PublicRouter.Group("system/setting")
	{
		publicSetting.GET("public", settingApi.GetPublicSetting) // 公开系统设置(登录页,免鉴权)
	}
}
