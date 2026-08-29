package gateway

import "github.com/gin-gonic/gin"

// ResourceApplicationRouter 资源申请审批路由(对齐前端 /gateway/application/* 资源)。
// apply/my 在 casbin 登录白名单(用户侧,数据范围由 JWT 锁定);list/approve/reject/batch-*
// 走菜单 ApiPrefix → casbin(管理端,审批管理页菜单 route.ai-audit_approval)。
type ResourceApplicationRouter struct{}

// InitResourceApplicationRouter 挂在 PrivateGroup，鉴权/操作日志由该组全局中间件统一处理。
func (r *ResourceApplicationRouter) InitResourceApplicationRouter(Router *gin.RouterGroup) {
	g := Router.Group("gateway/application")
	{
		g.POST("apply", applicationApi.CreateApplication)     // 提交申请(用户侧)
		g.GET("my", applicationApi.GetMyApplications)        // 我的申请(用户侧)
		g.GET("list", applicationApi.GetApplicationList)     // 审批列表(管理端)
		g.PUT("approve", applicationApi.ApproveApplication)  // 通过(管理端)
		g.PUT("reject", applicationApi.RejectApplication)    // 驳回(管理端)
		g.PUT("batch-approve", applicationApi.BatchApproveApplications)  // 批量通过(管理端)
		g.PUT("batch-reject", applicationApi.BatchRejectApplications)    // 批量驳回(管理端)
	}
}
