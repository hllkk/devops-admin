package gateway

import "github.com/gin-gonic/gin"

// CredentialRouter 凭证管理路由(对齐前端 /gateway/credential/* 资源)
type CredentialRouter struct{}

// InitCredentialRouter 挂在 PrivateGroup，鉴权/操作日志/数据权限由该组全局中间件统一处理。
// 静态段(provider-fields/resync)书写在 :id 之前注册。
func (c *CredentialRouter) InitCredentialRouter(Router *gin.RouterGroup) {
	r := Router.Group("gateway/credential")
	{
		r.GET("list", credentialApi.GetCredentialList)           // 分页获取凭证列表
		r.GET("provider-fields", credentialApi.GetProviderFields) // 供应商凭证表单字段定义(动态表单)
		r.POST("resync", credentialApi.ResyncCredentials)         // 手动重同步 LiteLLM(漂移兜底)
		r.GET(":id", credentialApi.GetCredential)                 // 凭证详情
		r.POST("", credentialApi.CreateCredential)                // 新增凭证
		r.PUT("", credentialApi.UpdateCredential)                 // 修改凭证
		r.DELETE(":ids", credentialApi.BatchDeleteCredential)     // 批量删除凭证
	}
}
