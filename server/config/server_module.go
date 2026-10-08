package config

// ServerConfig 服务器模块配置（资产管理/agent/IPMI/Docker/数据库/网络监控）。
// 凭据值(BMC/SSH/数据库/SNMP)落库 AES-256-GCM 加密，密钥未配置则拒绝写入
// (对齐 litellm.credential-key 先例；轮换会使历史密文不可解)。
type ServerConfig struct {
	CredentialKey string `mapstructure:"credential-key" json:"credential-key" yaml:"credential-key"` // 凭据值加密密钥（64 字符 hex=SERVER_CREDENTIAL_KEY，生产由 env 覆盖）
}
