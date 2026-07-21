package system

import "github.com/gin-gonic/gin"

// SettingRouter 系统设置路由(对齐前端 /system/setting)
type SettingRouter struct{}

// InitSettingRouter 挂在 PrivateGroup 下(管理员,鉴权+操作日志由该组全局中间件统一处理)
func (s *SettingRouter) InitSettingRouter(Router *gin.RouterGroup) {
	settingRouter := Router.Group("system/setting")
	{
		settingRouter.GET("", settingApi.GetSetting)  // 获取系统设置(聚合 general+security)
		settingRouter.PUT("", settingApi.UpdateSetting) // 更新系统设置(聚合保存)
	}
}
