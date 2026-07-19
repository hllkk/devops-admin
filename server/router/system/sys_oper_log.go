package system

import (
	"github.com/gin-gonic/gin"
)

// OperLogRouter 操作日志路由(对齐前端 /log/operlog/* 资源)
type OperLogRouter struct{}

// InitOperLogRouter 操作日志路由挂在 PrivateGroup 下,鉴权与操作日志由该组全局中间件统一处理。
// DELETE ":ids"(参数段)与 "clean"(静态段)同层共存,gin static 优先匹配,注册顺序无关。
func (o *OperLogRouter) InitOperLogRouter(Router *gin.RouterGroup) {
	operLogRouter := Router.Group("log/operlog")
	{
		operLogRouter.GET("list", operLogApi.GetOperLogList)        // 分页获取操作日志列表
		operLogRouter.DELETE(":ids", operLogApi.BatchDeleteOperLog) // 批量删除操作日志
		operLogRouter.DELETE("clean", operLogApi.CleanOperLog)      // 清空操作日志
	}
}
