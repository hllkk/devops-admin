package system

import (
	"github.com/gin-gonic/gin"
	api "github.com/hllkk/devops-admin/server/api/v1"
)

type RouterGroup struct {
	InitRouter
	BaseRouter
	LoginLogRouter
}

var (
	dbApi       = api.ApiGroupApp.SystemApiGroup.DBApi
	baseApi     = api.ApiGroupApp.SystemApiGroup.BaseApi
	loginLogApi = api.ApiGroupApp.SystemApiGroup.LoginLogApi
)

// RegisterPublic 实现 router.ModuleRouter —— 注册公开路由（无需认证）
func (rg *RouterGroup) RegisterPublic(r *gin.RouterGroup) {
	rg.InitRouter.InitInitRouter(r)
	rg.BaseRouter.InitBaseRouter(r, nil) // 只注册 public 部分
}

// RegisterPrivate 实现 router.ModuleRouter —— 注册需认证路由（JWT）
func (rg *RouterGroup) RegisterPrivate(r *gin.RouterGroup) {
	rg.BaseRouter.InitBaseRouter(nil, r)    // 只注册 private 部分
	rg.LoginLogRouter.InitLoginLogRouter(r) // 登录日志（需认证）
}

// RegisterAdmin 实现 router.ModuleRouter —— 注册管理员路由（JWT + RequireAdmin）
func (rg *RouterGroup) RegisterAdmin(r *gin.RouterGroup) {
	// 当前 system 模块暂无仅限管理员的路由；后续新增用户/角色/菜单管理路由时在此注册
}
