package config

// ServerConfig 服务器模块配置（资产管理/采集端 node_exporter/Prometheus 查询）。
// 凭据值(BMC/SSH/数据库/SNMP)落库 AES-256-GCM 加密，密钥未配置则拒绝写入
// (对齐 litellm.credential-key 先例；轮换会使历史密文不可解)。
type ServerConfig struct {
	CredentialKey string      `mapstructure:"credential-key" json:"credential-key" yaml:"credential-key"` // 凭据值加密密钥（64 字符 hex=SERVER_CREDENTIAL_KEY，生产由 env 覆盖）
	Agent         AgentConfig `mapstructure:"agent" json:"agent" yaml:"agent"`                            // 采集端(node_exporter)安装/SSH 通道
	Prometheus    PrometheusConfig `mapstructure:"prometheus" json:"prometheus" yaml:"prometheus"`      // Prometheus 集成(查询/SD)
}

// AgentConfig 采集端(node_exporter)生命周期配置。
// 数据面走 Prometheus pull(node_exporter:9100 被刮取)，管理面(安装/重启/卸载)
// 走 SSH 公钥通道——平台 agent 进程已退役。
type AgentConfig struct {
	InstallTimeout int    `mapstructure:"install-timeout" json:"install-timeout" yaml:"install-timeout"` // 安装单步超时(秒,SSH 连接/传输等;默认 120)
	BinaryDir      string `mapstructure:"binary-dir" json:"binary-dir" yaml:"binary-dir"`               // node_exporter 二进制托管目录(按架构文件名 node_exporter-<arch>;默认 resource/node-exporter)
	SSHKeyDir      string `mapstructure:"ssh-key-dir" json:"ssh-key-dir" yaml:"ssh-key-dir"`            // 平台 SSH 密钥对目录(启动期自动生成 ed25519;默认 resource/ssh)
	// ExporterPort node_exporter 监听端口(默认 9100;与 Prometheus scrape 配置保持一致)
	ExporterPort int `mapstructure:"exporter-port" json:"exporter-port" yaml:"exporter-port"` // node_exporter 端口
}

// PrometheusConfig Prometheus 集成配置(平台代查数据/服务发现)。
type PrometheusConfig struct {
	BaseURL string `mapstructure:"base-url" json:"base-url" yaml:"base-url"` // Prometheus HTTP API 地址(prod 容器间走服务名 http://prometheus:9090;dev 本地容器 http://127.0.0.1:9090)
	SDToken string `mapstructure:"sd-token" json:"sd-token" yaml:"sd-token"` // HTTP SD 端点鉴权 token(Prometheus http_sd_configs 的 Bearer;留空=SD 端点拒绝)
}
