package gateway

// ServiceGroup AI 网关业务服务组(对齐 system 组的聚合方式)。
// P1 五件套已齐：Provider/Credential/Model+Deployment/AiKey/Usage回流+聚合看板。
type ServiceGroup struct {
	ProviderService
	CredentialService
	ModelService
	DeploymentService
	AiKeyService
	KeyScenarioService
	UsageSyncService
	UsageAggregateService
	DashboardService
	CostAnalysisService
	RouterSettingsService
	ProviderBalanceService
	ResourceApplicationService
	McpService
	SkillService
	BudgetRuleService
}
