package config

// Social 第三方社交登录配置。
//
// 当前仅承载 token 加解密密钥;provider 的 clientId/clientSecret/callbackUrl/enabled
// 存于 sys_auth_config 表(运行时热更),不走本配置文件。
type Social struct {
	// TokenKey sys_social.accessToken/refreshToken 的 AES-256-GCM 加解密密钥。
	// 32 字节,以 64 字符 hex 字符串配置。未配置时启动 warn,实际加密时报错。
	// 严禁硬编码,必须从 config.yaml 读取。
	TokenKey string `mapstructure:"token-key" json:"tokenKey" yaml:"token-key"`
}
