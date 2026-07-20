package system

import (
	"github.com/gin-gonic/gin"
)

// NoticeRouter 通知公告路由(对齐前端 /system/notice/* 资源)
type NoticeRouter struct{}

// InitNoticeRouter 通知公告路由挂在 PrivateGroup 下,鉴权与操作日志由该组全局中间件统一处理。
func (n *NoticeRouter) InitNoticeRouter(Router *gin.RouterGroup) {
	noticeRouter := Router.Group("system/notice")
	{
		noticeRouter.GET("list", noticeApi.GetNoticeList)        // 分页获取通知公告列表
		noticeRouter.POST("", noticeApi.CreateNotice)            // 新增通知公告
		noticeRouter.PUT("", noticeApi.UpdateNotice)             // 修改通知公告
		noticeRouter.DELETE(":ids", noticeApi.BatchDeleteNotice) // 批量删除通知公告
		noticeRouter.GET("unread", noticeApi.GetUnreadNotice)    // 当前用户通知列表(未读/历史)
		noticeRouter.PUT("read", noticeApi.MarkNoticeRead)       // 标记通知已读
	}
}

// InitNoticeSSERouter SSE 接入路由,挂在专用组(不经 AccessLog/OperationRecord,避免缓冲破坏流式)。
// JWTAuth 已在专用组上挂载,从 httpOnly cookie 解析身份。路径对齐前端 /resource/sse。
func (n *NoticeRouter) InitNoticeSSERouter(Router *gin.RouterGroup) {
	Router.GET("resource/sse", noticeApi.Stream)
}
