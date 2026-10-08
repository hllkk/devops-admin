package config

// ServerConfig 服务器模块配置（资产管理/agent/IPMI/Docker/数据库/网络监控）。
// 凭据值(BMC/SSH/数据库/SNMP)落库 AES-256-GCM 加密，密钥未配置则拒绝写入
// (对齐 litellm.credential-key 先例；轮换会使历史密文不可解)。
type ServerConfig struct {
	CredentialKey string      `mapstructure:"credential-key" json:"credential-key" yaml:"credential-key"` // 凭据值加密密钥（64 字符 hex=SERVER_CREDENTIAL_KEY，生产由 env 覆盖）
	Agent         AgentConfig `mapstructure:"agent" json:"agent" yaml:"agent"`                            // agent 安装/心跳/二进制托管
}

// AgentConfig agent 生命周期配置(安装/注册/心跳/失联判定/二进制托管)。
type AgentConfig struct {
	HeartbeatInterval int    `mapstructure:"heartbeat-interval" json:"heartbeat-interval" yaml:"heartbeat-interval"` // agent 心跳间隔(秒,注册时下发;默认 30)
	LostThreshold     int    `mapstructure:"lost-threshold" json:"lost-threshold" yaml:"lost-threshold"`             // 失联判定阈值(秒,超过未心跳置 lost;默认 90=3 次心跳)
	InstallTimeout    int    `mapstructure:"install-timeout" json:"install-timeout" yaml:"install-timeout"`          // 安装单步超时(秒,SSH 连接/传输等;默认 120)
	BinaryDir         string `mapstructure:"binary-dir" json:"binary-dir" yaml:"binary-dir"`                         // agent 二进制托管目录(按架构文件名 aiops-agent-<ver>-linux-<arch>;默认 resource/agent)
	SSHKeyDir         string `mapstructure:"ssh-key-dir" json:"ssh-key-dir" yaml:"ssh-key-dir"`                      // 平台 SSH 密钥对目录(启动期自动生成 ed25519;默认 resource/ssh)
	ServerURL         string `mapstructure:"server-url" json:"server-url" yaml:"server-url"`                         // agent 回连接入地址(http://host:port;留空=按本机出网IP+端口推导,跨网段部署显式配置)
}
