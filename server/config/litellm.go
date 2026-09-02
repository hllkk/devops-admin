package config

// Litellm AI 网关 LiteLLM 代理底座配置（管理面调用）。
// 转发由 LiteLLM 容器承担：客户端直连 public-url 用模型；管理面 Go 后端经 base-url
// 调 LiteLLM 管理 API，用 master-key 鉴权，做凭证/模型部署/AI 密钥的同步与用量拉取。
// 详见 aiDoc/modules/business-modules.md「AI 网关模块」节。
type Litellm struct {
	BaseURL     string `mapstructure:"base-url" json:"base-url" yaml:"base-url"`           // LiteLLM 管理 API 地址（prod 容器间走服务名 http://litellm:4000）
	PublicURL   string `mapstructure:"public-url" json:"public-url" yaml:"public-url"`     // 客户端转发接入点（下发前端/客户端；prod 经 nginx /llm/ 反代或直暴露 4000）
	MasterKey   string `mapstructure:"master-key" json:"master-key" yaml:"master-key"`     // LiteLLM 管理面鉴权密钥（LITELLM_MASTER_KEY，生产由 env 覆盖）
	CredentialKey string `mapstructure:"credential-key" json:"credential-key" yaml:"credential-key"` // 凭证值加密密钥（AES-256-GCM，64 字符 hex=LITELLM_CREDENTIAL_KEY，生产由 env 覆盖；轮换会使历史密文不可解）
	Timeout     int    `mapstructure:"timeout" json:"timeout" yaml:"timeout"`               // HTTP 调用超时（秒）
	UsdToCnyRate float64 `mapstructure:"usd-to-cny-rate" json:"usd-to-cny-rate" yaml:"usd-to-cny-rate"` // 美元兑人民币汇率（部署定价 ¥/百万token → LiteLLM USD/token 换算；0/负值运行时按 7.0 兜底）
	SpendDSN    string `mapstructure:"spend-dsn" json:"spend-dsn" yaml:"spend-dsn"` // LiteLLM spend logs 库连接（留空复用主库；dev litellm 独立库需配，prod 共享库留空）
	LogSyncInterval int `mapstructure:"log-sync-interval" json:"log-sync-interval" yaml:"log-sync-interval"` // 用量回流间隔（分钟，默认5，种子调度用）
	LogSyncBatch int    `mapstructure:"log-sync-batch" json:"log-sync-batch" yaml:"log-sync-batch"` // 用量回流批大小（默认1000）
	LogReconcileWindow int `mapstructure:"log-reconcile-window" json:"log-reconcile-window" yaml:"log-reconcile-window"` // 对账回灌窗口（天，默认30）
	LogRetentionDays int `mapstructure:"log-retention-days" json:"log-retention-days" yaml:"log-retention-days"` // 用量日志保留天数（llm+mcp，0=不清理；生效值不低于对账窗口+7，防删了又被对账重灌）
	SyncEnabled bool   `mapstructure:"sync-enabled" json:"sync-enabled" yaml:"sync-enabled"` // 是否启用与 LiteLLM 的双向同步（关闭则管理面单机运行，不写 LiteLLM 侧）
}
