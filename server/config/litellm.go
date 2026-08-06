package config

// Litellm AI 网关 LiteLLM 代理底座配置（管理面调用）。
// 转发由 LiteLLM 容器承担：客户端直连 public-url 用模型；管理面 Go 后端经 base-url
// 调 LiteLLM 管理 API，用 master-key 鉴权，做凭证/模型部署/AI 密钥的同步与用量拉取。
// 详见 aiDoc/modules/business-modules.md「AI 网关模块」节。
type Litellm struct {
	BaseURL     string `mapstructure:"base-url" json:"base-url" yaml:"base-url"`           // LiteLLM 管理 API 地址（prod 容器间走服务名 http://litellm:4000）
	PublicURL   string `mapstructure:"public-url" json:"public-url" yaml:"public-url"`     // 客户端转发接入点（下发前端/客户端；prod 经 nginx /llm/ 反代或直暴露 4000）
	MasterKey   string `mapstructure:"master-key" json:"master-key" yaml:"master-key"`     // LiteLLM 管理面鉴权密钥（LITELLM_MASTER_KEY，生产由 env 覆盖）
	Timeout     int    `mapstructure:"timeout" json:"timeout" yaml:"timeout"`               // HTTP 调用超时（秒）
	SyncEnabled bool   `mapstructure:"sync-enabled" json:"sync-enabled" yaml:"sync-enabled"` // 是否启用与 LiteLLM 的双向同步（关闭则管理面单机运行，不写 LiteLLM 侧）
}
