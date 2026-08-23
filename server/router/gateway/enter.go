package gateway

import v1 "github.com/hllkk/devops-admin/server/api/v1"

// RouterGroup AI 网关路由组(挂在 PrivateGroup，鉴权与操作日志由该组全局中间件统一处理)。
type RouterGroup struct {
	ProviderRouter
	CredentialRouter
	ModelRouter
	DeploymentRouter
}

var (
	providerApi    = v1.ApiGroupApp.GatewayApiGroup.ProviderApi
	credentialApi  = v1.ApiGroupApp.GatewayApiGroup.CredentialApi
	modelApi       = v1.ApiGroupApp.GatewayApiGroup.ModelApi
	deploymentApi  = v1.ApiGroupApp.GatewayApiGroup.DeploymentApi
)
