package gateway

import "github.com/gin-gonic/gin"

// AiKeyRouter AI 密钥路由(对齐前端 /gateway/ai-key/* 资源)
type AiKeyRouter struct{}

// InitAiKeyRouter 挂在 PrivateGroup，鉴权/操作日志/数据权限由该组全局中间件统一处理。
// 静态段(identity/my、identity/available-models、list、scenario/*、resync)书写在 :id 之前注册。
func (r *AiKeyRouter) InitAiKeyRouter(Router *gin.RouterGroup) {
	g := Router.Group("gateway/ai-key")
	{
		g.GET("identity/my", aiKeyApi.GetMyIdentity)                      // 我的 AI 身份(管理员创建制,未开通 opened=false)
		g.GET("identity/available-models", aiKeyApi.GetAvailableModels)   // 可授权模型列表
		g.GET("list", aiKeyApi.GetAiKeyList)                              // 密钥分页列表(管理员)
		g.POST("batch", aiKeyApi.BatchCreateMainKeys)                     // 批量开通个人主Key(按部门/按用户)
		g.POST("resync", aiKeyApi.ResyncAiKeys)                           // 全量重推密钥投影到LiteLLM(漂移兜底)
		g.GET("scenario/list", keyScenarioApi.GetKeyScenarioList)         // 使用场景分页列表
		g.GET("scenario/all", keyScenarioApi.GetAllScenarios)             // 启用中场景全量(建Key下拉)
		g.POST("scenario", keyScenarioApi.CreateKeyScenario)              // 新增使用场景
		g.PUT("scenario", keyScenarioApi.UpdateKeyScenario)               // 修改使用场景
		g.DELETE("scenario/:ids", keyScenarioApi.BatchDeleteKeyScenarios) // 批量删除使用场景
		g.GET(":id", aiKeyApi.GetAiKey)                                   // 密钥详情
		g.GET("value/:id", aiKeyApi.RevealAiKey)                          // 查看密钥完整明文(仅管理员/超管,审计)
		g.POST("rotate/:id", aiKeyApi.RotateAiKey)                        // 轮换密钥(原地换Key值保归因)
		g.POST("", aiKeyApi.CreateSceneKey)                               // 创建密钥
		g.PUT("", aiKeyApi.UpdateAiKey)                                   // 修改密钥
		g.DELETE(":ids", aiKeyApi.BatchDeleteAiKeys)                      // 批量删除密钥
	}
}
