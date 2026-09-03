package system

import "github.com/gin-gonic/gin"

type UpgradeRouter struct{}

// InitUpgradeRouter 在线升级路由(版本/检查/状态为登录即可——rbacWhitelistPrivate 白名单;
// start 触发升级为管理操作,走 casbin 菜单授权——系统设置菜单 ApiPrefix 含 /system/upgrade/start)
func (u *UpgradeRouter) InitUpgradeRouter(Router *gin.RouterGroup) {
	upgradeRouter := Router.Group("system/upgrade")
	{
		upgradeRouter.GET("version", upgradeApi.GetVersion)      // 版本信息(关于弹窗)
		upgradeRouter.GET("check", upgradeApi.CheckUpdate)       // 检查更新
		upgradeRouter.POST("start", upgradeApi.StartUpgrade)     // 触发在线升级
		upgradeRouter.GET("status", upgradeApi.GetUpgradeStatus) // 升级状态(轮询)
	}
}
