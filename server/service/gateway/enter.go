package gateway

// ServiceGroup AI 网关业务服务组(对齐 system 组的聚合方式)。
// P1 五件套已齐：Provider/Credential/Model+Deployment/AiKey；用量看板见 slice5。
type ServiceGroup struct {
	ProviderService
	CredentialService
	ModelService
	DeploymentService
	AiKeyService
}
