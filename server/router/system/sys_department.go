package system

import (
	"github.com/gin-gonic/gin"
)

// DeptRouter 部门管理路由(对齐前端 /system/dept/* 资源)
type DeptRouter struct{}

// InitDeptRouter 部门相关路由挂在 PrivateGroup 下,鉴权与操作日志由该组全局中间件统一处理。
func (d *DeptRouter) InitDeptRouter(Router *gin.RouterGroup) {
	deptRouter := Router.Group("system/dept")
	{
		deptRouter.GET("list", deptApi.GetDeptList)                        // 部门列表(平表,前端组装树)
		deptRouter.GET("list/exclude/:deptId", deptApi.GetExcludeDeptList) // 排除指定部门及子部门(选父级)
		deptRouter.GET("optionselect", deptApi.GetDeptOption)              // 部门下拉
		deptRouter.POST("", deptApi.CreateDept)                            // 新增部门
		deptRouter.PUT("", deptApi.UpdateDept)                             // 修改部门
		deptRouter.DELETE(":ids", deptApi.BatchDeleteDept)                 // 批量删除部门
	}
}
