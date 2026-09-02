package system

import (
	"github.com/gin-gonic/gin"
)

// WecomContactRouter 企业微信通讯录同步路由(私有,管理员)。
//
// 与扫码登录的 WecomRouter(全公开)区分:通讯录同步是管理操作,挂 PrivateGroup,
// 自动走 JWTAuth→OperationRecord→CasbinHandler→DataScope,仅持有该接口权限点的角色可触发。
type WecomContactRouter struct{}

// InitWecomContactRouter 通讯录同步路由。
// 路由清单(前缀 RouterPrefix,通常 /api/v1):
//   - POST /system/wecom/syncStructure   手动触发通讯录同步(异步启动)
//   - GET  /system/wecom/syncStatus      查询同步状态(进度/最近结果)
func (r *WecomContactRouter) InitWecomContactRouter(Router *gin.RouterGroup) {
	w := Router.Group("system/wecom")
	{
		w.POST("syncStructure", wecomContactApi.SyncStructure)
		w.GET("syncStatus", wecomContactApi.SyncStatus)
	}
}
