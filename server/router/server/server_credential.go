package server

import "github.com/gin-gonic/gin"

// CredentialRouter 服务器模块采集凭据路由(对齐前端 /server/credential/* 资源)
type CredentialRouter struct{}

// InitCredentialRouter 挂在 PrivateGroup，鉴权/操作日志由该组全局中间件统一处理。
func (c *CredentialRouter) InitCredentialRouter(Router *gin.RouterGroup) {
	r := Router.Group("server/credential")
	{
		// 静态段(list/options)注册在 :id 前(gin 路由树静态段优先匹配)
		r.GET("list", credentialApi.GetCredentialList)        // 分页获取凭据列表
		r.GET("options", credentialApi.GetCredentialOptions)  // 凭据下拉选项(启用中,type 过滤)
		r.GET(":id", credentialApi.GetCredential)             // 凭据详情
		r.POST("", credentialApi.CreateCredential)            // 新增凭据
		r.PUT("", credentialApi.UpdateCredential)             // 修改凭据
		r.DELETE(":ids", credentialApi.BatchDeleteCredential) // 批量删除凭据(被资产关联时拒删)
	}
}
