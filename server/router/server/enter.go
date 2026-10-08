package server

import v1 "github.com/hllkk/devops-admin/server/api/v1"

// RouterGroup 服务器模块路由组(挂在 PrivateGroup，鉴权与操作日志由该组全局中间件统一处理)。
type RouterGroup struct {
	AssetRouter
	CredentialRouter
}

var (
	assetApi      = v1.ApiGroupApp.ServerApiGroup.AssetApi
	credentialApi = v1.ApiGroupApp.ServerApiGroup.CredentialApi
)
