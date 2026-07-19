package system

import (
	"github.com/gin-gonic/gin"
)

// RoleRouter 角色管理路由(对齐前端 /system/role/* 资源)
type RoleRouter struct{}

// InitRoleRouter 角色相关路由挂在 PrivateGroup 下,鉴权与操作日志由该组全局中间件统一处理。
func (r *RoleRouter) InitRoleRouter(Router *gin.RouterGroup) {
	roleRouter := Router.Group("system/role")
	{
		roleRouter.GET("list", roleApi.GetRoleList)                            // 角色列表
		roleRouter.GET("authUser/allocatedList", roleApi.GetAllocatedUserList) // 角色已分配用户
		roleRouter.POST("", roleApi.CreateRole)                                // 新增角色(含分配菜单)
		roleRouter.PUT("", roleApi.UpdateRole)                                 // 修改角色(全量替换菜单)
		roleRouter.PUT("changeStatus", roleApi.UpdateRoleStatus)               // 修改角色状态
		roleRouter.PUT("authUser/selectAll", roleApi.AuthUserSelectAll)        // 批量授权用户
		roleRouter.PUT("authUser/cancelAll", roleApi.AuthUserCancelAll)        // 批量取消授权
		roleRouter.DELETE(":ids", roleApi.BatchDeleteRole)                     // 批量删除角色
	}
}
