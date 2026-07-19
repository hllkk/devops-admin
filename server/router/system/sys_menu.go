package system

import (
	"github.com/gin-gonic/gin"
)

// MenuRouter 菜单管理路由(对齐前端 /system/menu/* 资源)
type MenuRouter struct{}

// InitMenuRouter 菜单相关路由挂在 PrivateGroup 下,鉴权与操作日志由该组全局中间件统一处理。
// 注:DELETE ":menuId"(参数段)与 "cascade/:menuIds"(静态段+参数段)在同层共存,
// gin 允许 static+param 同层(static 优先匹配),注册顺序无关。
func (m *MenuRouter) InitMenuRouter(Router *gin.RouterGroup) {
	menuRouter := Router.Group("system/menu")
	{
		menuRouter.GET("list", menuApi.GetMenuList)                                 // 菜单列表(平表,前端组装树)
		menuRouter.GET("treeselect", menuApi.GetMenuTreeSelect)                     // 菜单树选择
		menuRouter.GET("roleMenuTreeselect/:roleId", menuApi.GetRoleMenuTreeSelect) // 角色菜单权限树
		menuRouter.POST("", menuApi.CreateMenu)                                     // 新增菜单
		menuRouter.PUT("", menuApi.UpdateMenu)                                      // 修改菜单
		menuRouter.DELETE(":menuId", menuApi.DeleteMenu)                            // 删除单个菜单
		menuRouter.DELETE("cascade/:menuIds", menuApi.CascadeDeleteMenu)            // 级联删除菜单(含子孙)
	}
}
