package system

import (
	"github.com/gin-gonic/gin"
)

type TimedTaskRouter struct{}

// InitTimedTaskRouter 定时任务管理路由(CRUD/启停/触发/查询),在 PrivateGroup 下鉴权。
func (s *TimedTaskRouter) InitTimedTaskRouter(Router *gin.RouterGroup) {
	timedTaskRouter := Router.Group("timedTask")
	{
		timedTaskRouter.POST("createTimedTask", timedTaskApi.CreateTimedTask)         // 创建定时任务
		timedTaskRouter.PUT("updateTimedTask", timedTaskApi.UpdateTimedTask)          // 更新定时任务
		timedTaskRouter.DELETE("deleteTimedTask", timedTaskApi.DeleteTimedTask)       // 删除定时任务
		timedTaskRouter.POST("toggleTimedTask", timedTaskApi.ToggleTimedTask)         // 启用/停用
		timedTaskRouter.POST("triggerTimedTask", timedTaskApi.TriggerTimedTask)       // 手动触发
		timedTaskRouter.GET("getTimedTaskList", timedTaskApi.GetTimedTaskList)        // 任务列表
		timedTaskRouter.GET("getTimedTaskLogList", timedTaskApi.GetTimedTaskLogList)  // 执行日志
		timedTaskRouter.GET("getRegisteredMethods", timedTaskApi.GetRegisteredMethods) // 已注册方法
	}
}

// InitTimedTaskSSERouter SSE 告警订阅路由,挂载在提前于 AccessLog 的 SSE 专用组(避免缓冲破坏流式)。
// JWTAuth 已在该专用组上挂载。路径对齐 /resource/timedTask/alertStream。
func (s *TimedTaskRouter) InitTimedTaskSSERouter(Router *gin.RouterGroup) {
	Router.GET("resource/timedTask/alertStream", timedTaskApi.AlertStream)
}
