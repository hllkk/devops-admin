package response

// HealthCard 状态卡(MCP 上游/模型部署两卡通用，分母=启用实体数)。
type HealthCard struct {
	Total     int `json:"total"`     // 启用总数
	Healthy   int `json:"healthy"`   // 健康
	Unhealthy int `json:"unhealthy"` // 异常
	Unknown   int `json:"unknown"`   // 未检测(从未巡检)
}

// HealthComponentItem 基础组件状态项(LiteLLM/PostgreSQL/Redis)。
type HealthComponentItem struct {
	Name      string `json:"name"`      // 组件名(litellm/postgresql/redis)
	Status    string `json:"status"`    // healthy/unhealthy/unknown(未配置)
	LatencyMs int64  `json:"latencyMs"` // 探测耗时
	Message   string `json:"message"`  // 失败信息(成功为空)
}

// HealthFreshness 数据回流新鲜度(游标 updated_at 判定，反映回流管道活性)。
type HealthFreshness struct {
	Status            string `json:"status"`            // healthy(≤10m)/warning(≤60m)/danger(>60m)/unknown(无记录)
	LlmSyncAt         string `json:"llmSyncAt"`         // llm_logs 游标最后推进(本地时区,空=无记录)
	McpSyncAt         string `json:"mcpSyncAt"`         // mcp_logs 游标最后推进
	LastSyncAt        string `json:"lastSyncAt"`        // 两游标较新者
	ThresholdMinutes  int    `json:"thresholdMinutes"`  // 判 healthy 的阈值(分钟)
	StaleWarnMinutes  int    `json:"staleWarnMinutes"`  // 判 warning 的阈值(分钟)
}

// HealthMcpItem MCP 上游明细行(读现有巡检落库)。
type HealthMcpItem struct {
	McpServerId      int64   `json:"mcpServerId,string"`
	Name             string  `json:"name"`
	ServerName       string  `json:"serverName"`
	HealthStatus     string  `json:"healthStatus"`
	LastHealthCheck  string  `json:"lastHealthCheck"` // 本地时区,空=从未
	HealthCheckError string  `json:"healthCheckError"`
}

// HealthDeploymentItem 模型部署明细行(路由组级结论,同组部署状态一致)。
type HealthDeploymentItem struct {
	DeploymentId     int64  `json:"deploymentId,string"`
	ModelName        string `json:"modelName"`
	ModelKey         string `json:"modelKey"`
	DeployName       string `json:"deployName"`
	HealthStatus     string `json:"healthStatus"`
	LastHealthCheck  string `json:"lastHealthCheck"` // 本地时区,空=从未
	HealthCheckError string `json:"healthCheckError"`
}

// HealthSummary 健康检查汇总(四卡+三块明细+生成时间)。
type HealthSummary struct {
	Mcp             HealthCard            `json:"mcp"`
	Deployment      HealthCard            `json:"deployment"`
	Components      []HealthComponentItem `json:"components"`
	Freshness       HealthFreshness       `json:"freshness"`
	McpItems        []HealthMcpItem       `json:"mcpItems"`
	DeploymentItems []HealthDeploymentItem `json:"deploymentItems"`
	CheckedAt       string                `json:"checkedAt"` // 汇总生成时间(本地时区)
}
