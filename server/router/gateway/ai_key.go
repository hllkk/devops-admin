package gateway

import "github.com/gin-gonic/gin"

// AiKeyRouter AI 密钥路由(对齐前端 /gateway/ai-key/* 资源)
type AiKeyRouter struct{}

// InitAiKeyRouter 挂在 PrivateGroup，鉴权/操作日志/数据权限由该组全局中间件统一处理。
// 静态段(identity/my、identity/available-models、list)书写在 :id 之前注册。
func (r *AiKeyRouter) InitAiKeyRouter(Router *gin.RouterGroup) {
	g := Router.Group("gateway/ai-key")
	{
		g.GET("identity/my", aiKeyApi.GetMyIdentity)               // 我的 AI 身份(管理员创建制,未开通 opened=false)
		g.GET("identity/available-models", aiKeyApi.GetAvailableModels) // 可授权模型列表
		g.GET("list", aiKeyApi.GetAiKeyList)                       // 密钥分页列表(管理员)
		g.GET(":id", aiKeyApi.GetAiKey)                           // 密钥详情
		g.POST("", aiKeyApi.CreateSceneKey)                        // 创建密钥
		g.PUT("", aiKeyApi.UpdateAiKey)                           // 修改密钥
		g.DELETE(":ids", aiKeyApi.BatchDeleteAiKeys)              // 批量删除密钥
	}
}
