package system

import "github.com/gin-gonic/gin"

// LoginLogRouter 登录日志路由（private 组）。
type LoginLogRouter struct{}

// InitLoginLogRouter 注册前端契约 /log/loginlog/*：
//   - GET  list              登录日志列表
//   - GET  unlock/:username  解锁用户（清失败计数）
//   - DELETE :action         action="clean" 清空，否则逗号分隔 infoId 批量删除
//
// 注：gin 同层不允许静态段 clean 与参数段 :ids 共存，故 DELETE 合并到 :action。
func (s *LoginLogRouter) InitLoginLogRouter(r *gin.RouterGroup) {
	log := r.Group("log/loginlog")
	{
		log.GET("list", loginLogApi.GetLoginLogList)
		log.GET("unlock/:username", loginLogApi.UnlockLoginLog)
		log.DELETE(":action", loginLogApi.HandleLoginLogDelete)
	}
}
