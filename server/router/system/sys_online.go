package system

import "github.com/gin-gonic/gin"

// OnlineRouter 在线设备路由(对齐前端 /monitor/online 资源,个人中心视角:仅当前用户自己)。
type OnlineRouter struct{}

// InitOnlineRouter 挂在 PrivateGroup 下,鉴权与操作日志由该组全局中间件统一处理。
func (o *OnlineRouter) InitOnlineRouter(Router *gin.RouterGroup) {
	g := Router.Group("monitor/online")
	{
		g.GET("", onlineApi.GetOnlineList)                      // 获取当前用户在线设备列表
		g.DELETE("myself/:tokenId", onlineApi.KickOnlineDevice) // 强制下线指定设备(仅自己)
	}
}
