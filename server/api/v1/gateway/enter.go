package gateway

import "github.com/hllkk/devops-admin/server/service"

// ApiGroup AI 网关管理面 API 组(挂在 PrivateGroup，对齐前端 /gateway/* 资源)。
type ApiGroup struct {
	ProviderApi
	CredentialApi
	ModelApi
	DeploymentApi
	AiKeyApi
	UsageApi
	DashboardApi
	RouterSettingsApi
}

var (
	providerService        = service.ServiceGroupApp.GatewayServiceGroup.ProviderService
	credentialService      = service.ServiceGroupApp.GatewayServiceGroup.CredentialService
	modelService           = service.ServiceGroupApp.GatewayServiceGroup.ModelService
	deploymentService      = service.ServiceGroupApp.GatewayServiceGroup.DeploymentService
	aiKeyService           = service.ServiceGroupApp.GatewayServiceGroup.AiKeyService
	usageService           = service.ServiceGroupApp.GatewayServiceGroup.UsageSyncService
	usageAggregateService  = service.ServiceGroupApp.GatewayServiceGroup.UsageAggregateService
	dashboardService       = service.ServiceGroupApp.GatewayServiceGroup.DashboardService
	routerSettingsService  = service.ServiceGroupApp.GatewayServiceGroup.RouterSettingsService
)
