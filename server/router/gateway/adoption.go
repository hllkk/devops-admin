package gateway

import "github.com/gin-gonic/gin"

// AdoptionRouter 覆盖率/采用度路由(对齐前端 /gateway/adoption/* 资源)
type AdoptionRouter struct{}

// InitAdoptionRouter 挂在 PrivateGroup，鉴权/操作日志由该组全局中间件统一处理。
// 接口权限走菜单 api_prefix(route.ai-audit_adoption)：user 角色不授，决策层/管理员视角。
func (r *AdoptionRouter) InitAdoptionRouter(Router *gin.RouterGroup) {
	g := Router.Group("gateway/adoption")
	{
		g.GET("overview", adoptionApi.GetAdoptionOverview)                 // KPI+DAU 趋势
		g.GET("departments", adoptionApi.GetAdoptionDepartments)           // 部门覆盖率明细(含零调用部门)
		g.GET("departments/:id/users", adoptionApi.GetAdoptionDeptUsers)   // 部门成员下钻(含未激活)
		g.GET("models", adoptionApi.GetAdoptionModels)                     // 模型分布
	}
}
