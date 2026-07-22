package system

import (
	"github.com/gin-gonic/gin"
)

// SysErrorRouter 错误日志路由(对齐前端 /log/sysError/* 资源)
type SysErrorRouter struct{}

// InitSysErrorRouter 错误日志路由:
//   - PrivateGroup(Router): 增删改 + 触发AI处理 + 查询(走鉴权)
//   - PublicGroup(PublicRouter): 前端错误上报 createSysError(无鉴权)
func (s *SysErrorRouter) InitSysErrorRouter(Router *gin.RouterGroup, PublicRouter *gin.RouterGroup) {
	sysErrorRouter := Router.Group("log/sysError")
	publicErrorRouter := PublicRouter.Group("log/sysError")
	{
		sysErrorRouter.DELETE("deleteSysError", sysErrorApi.DeleteSysError)               // 删除错误日志
		sysErrorRouter.POST("deleteSysErrorByIds", sysErrorApi.DeleteSysErrorByIds)       // 批量删除(POST body,避免query array序列化问题)
		sysErrorRouter.PUT("updateSysError", sysErrorApi.UpdateSysError)                  // 更新错误日志
		sysErrorRouter.GET("getSysErrorSolution", sysErrorApi.GetSysErrorSolution)        // 触发AI处理
	}
	{
		sysErrorRouter.GET("findSysError", sysErrorApi.FindSysError)       // 根据ID获取错误日志
		sysErrorRouter.GET("getSysErrorList", sysErrorApi.GetSysErrorList) // 获取错误日志列表
	}
	{
		publicErrorRouter.POST("createSysError", sysErrorApi.CreateSysError) // 前端上报错误日志(无鉴权)
	}
}
