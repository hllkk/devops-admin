package gateway

import v1 "github.com/hllkk/devops-admin/server/api/v1"

// RouterGroup AI 网关路由组(挂在 PrivateGroup，鉴权与操作日志由该组全局中间件统一处理)。
type RouterGroup struct {
	ProviderRouter
}

var providerApi = v1.ApiGroupApp.GatewayApiGroup.ProviderApi
