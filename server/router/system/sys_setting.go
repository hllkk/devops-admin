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
		notifyRouter.POST("test-email", settingApi.TestEmail)        // 发送测试邮件(使用当前表单值)
		notifyRouter.POST("test-wecom-app", settingApi.TestWecomApp) // 企微应用消息测试(选人实测)
		notifyRouter.POST("test-wecom-bot", settingApi.TestWecomBot) // 企微群机器人测试(按已录入群实测)
		notifyRouter.GET("wecom-bot-group", settingApi.WecomBotGroupList)
		notifyRouter.POST("wecom-bot-group", settingApi.WecomBotGroupCreate)
		notifyRouter.DELETE("wecom-bot-group/:id", settingApi.WecomBotGroupDelete)
	}
	publicSetting := PublicRouter.Group("system/setting")
	{
		publicSetting.GET("public", settingApi.GetPublicSetting) // 公开系统设置(登录页,免鉴权)
	}
	// 企业微信可信域名校验：企业微信会请求 /WW_verify_*.txt，无需鉴权
	// 注意路由不能写成 /WW_verify_:name.txt——gin 的 param 名会吞掉段内 ".txt" 后缀
	// (param key 变成 "name.txt"，c.Param("name") 恒空，v1.12.0 探针实测)，
	// 让 :name 吃整段(含 .txt)，handler 侧拼回完整文件名
	PublicRouter.GET("/WW_verify_:name", settingApi.WecomDomainVerify)
}
