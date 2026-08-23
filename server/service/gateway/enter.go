package gateway

// ServiceGroup AI 网关业务服务组(对齐 system 组的聚合方式)。
// 随 P1 推进追加 AiKey/Usage 等 Service。
type ServiceGroup struct {
	ProviderService
	CredentialService
	ModelService
	DeploymentService
}
