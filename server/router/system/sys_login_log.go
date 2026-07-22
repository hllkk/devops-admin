package system

import (
	"github.com/gin-gonic/gin"
)

// LoginLogRouter 登录日志路由(对齐前端 /log/loginlog/* 资源)
type LoginLogRouter struct{}

// InitLoginLogRouter 登录日志路由挂在 PrivateGroup 下,鉴权与操作日志由该组全局中间件统一处理。
// DELETE ":ids"(参数段)与 "clean"(静态段)同层共存,gin static 优先匹配,注册顺序无关。
func (l *LoginLogRouter) InitLoginLogRouter(Router *gin.RouterGroup) {
	loginLogRouter := Router.Group("log/loginlog")
	{
		loginLogRouter.GET("list", loginLogApi.GetLoginLogList)            // 分页获取登录日志列表
		loginLogRouter.POST("export", loginLogApi.ExportLoginLog)          // 导出登录日志(Excel)
		loginLogRouter.GET("unlock/:username", loginLogApi.UnlockLoginLog) // 解锁账号(清失败计数与锁)
		loginLogRouter.DELETE(":ids", loginLogApi.BatchDeleteLoginLog)     // 批量删除登录日志
		loginLogRouter.DELETE("clean", loginLogApi.CleanLoginLog)          // 清空登录日志
	}
}
