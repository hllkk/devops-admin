package gateway

import "github.com/hllkk/devops-admin/server/service"

// ApiGroup AI 网关管理面 API 组(挂在 PrivateGroup，对齐前端 /gateway/* 资源)。
type ApiGroup struct {
	ProviderApi
}

var (
	providerService = service.ServiceGroupApp.GatewayServiceGroup.ProviderService
)
