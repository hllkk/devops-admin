package config

type System struct {
	DbType              string   `mapstructure:"db-type" json:"db-type" yaml:"db-type"`    // 数据库类型:mysql(默认)|sqlite|sqlserver|postgresql
	OssType             string   `mapstructure:"oss-type" json:"oss-type" yaml:"oss-type"` // Oss类型
	RouterPrefix        string   `mapstructure:"router-prefix" json:"router-prefix" yaml:"router-prefix"`
	Addr                int      `mapstructure:"addr" json:"addr" yaml:"addr"` // 端口值
	LimitCountIP        int      `mapstructure:"iplimit-count" json:"iplimit-count" yaml:"iplimit-count"`
	LimitTimeIP         int      `mapstructure:"iplimit-time" json:"iplimit-time" yaml:"iplimit-time"`
	UseMultipoint       bool     `mapstructure:"use-multipoint" json:"use-multipoint" yaml:"use-multipoint"`                      // 多点登录拦截
	UseRedis            bool     `mapstructure:"use-redis" json:"use-redis" yaml:"use-redis"`                                     // 使用redis
	UseMongo            bool     `mapstructure:"use-mongo" json:"use-mongo" yaml:"use-mongo"`                                     // 使用mongo
	UseStrictAuth       bool     `mapstructure:"use-strict-auth" json:"use-strict-auth" yaml:"use-strict-auth"`                   // 使用树形角色分配模式
	DisableAutoMigrate  bool     `mapstructure:"disable-auto-migrate" json:"disable-auto-migrate" yaml:"disable-auto-migrate"`    // 自动迁移数据库表结构，生产环境建议设为true，手动迁移
	InitKey             string   `mapstructure:"init-key" json:"init-key" yaml:"init-key"`                                        // 系统初始化密钥，为空则禁用初始化 API
	SecretEncryptKey    string   `mapstructure:"secret-encrypt-key" json:"secret-encrypt-key" yaml:"secret-encrypt-key"`          // 数据库敏感字段加密密钥；空则回退到 JWT.SigningKey
	TrustedProxies      []string `mapstructure:"trusted-proxies" json:"trusted-proxies" yaml:"trusted-proxies"`                   // 可信反代 CIDR/IP；空则不信任任何代理
	StrictDeviceBinding bool     `mapstructure:"strict-device-binding" json:"strict-device-binding" yaml:"strict-device-binding"` // 严格设备绑定：UA 变化时强制重新登录而非仅告警
	LoginLimitCount     int      `mapstructure:"login-limit-count" json:"login-limit-count" yaml:"login-limit-count"`             // 登录端点单 IP 限流：窗口内最大尝试次数；<=0 走默认 10
	LoginLimitWindow    int      `mapstructure:"login-limit-window" json:"login-limit-window" yaml:"login-limit-window"`          // 登录限流窗口（秒）；<=0 走默认 60
}
