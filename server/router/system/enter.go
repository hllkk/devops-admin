package system

import (
	"github.com/gin-gonic/gin"
	api "github.com/hllkk/devops-admin/server/api/v1"
)

type RouterGroup struct {
	InitRouter
	BaseRouter
	LoginLogRouter
	SettingRouter
}

var (
	dbApi       = api.ApiGroupApp.SystemApiGroup.DBApi
	baseApi     = api.ApiGroupApp.SystemApiGroup.BaseApi
	loginLogApi = api.ApiGroupApp.SystemApiGroup.LoginLogApi
	settingApi  = api.ApiGroupApp.SystemApiGroup.SettingApi
)

// RegisterPublic 实现 router.ModuleRouter —— 注册公开路由（无需认证）
func (rg *RouterGroup) RegisterPublic(r *gin.RouterGroup) {
	rg.InitRouter.InitInitRouter(r)
	rg.BaseRouter.InitBaseRouter(r, nil)               // 只注册 public 部分
	rg.SettingRouter.InitSettingPublicRouter(r)        // 系统设置公开接口（登录页读取展示配置）
}

// RegisterPrivate 实现 router.ModuleRouter —— 注册需认证路由（JWT）
func (rg *RouterGroup) RegisterPrivate(r *gin.RouterGroup) {
	rg.BaseRouter.InitBaseRouter(nil, r)    // 只注册 private 部分
	rg.LoginLogRouter.InitLoginLogRouter(r) // 登录日志（需认证）
}

// RegisterAdmin 实现 router.ModuleRouter —— 注册管理员路由（JWT + RequireAdmin）
func (rg *RouterGroup) RegisterAdmin(r *gin.RouterGroup) {
	rg.SettingRouter.InitSettingRouter(r) // 系统设置（管理员读写）
}
