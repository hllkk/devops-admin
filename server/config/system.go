package config

type System struct {
	DbType             string   `mapstructure:"db-type" json:"db-type" yaml:"db-type"`    // 数据库类型:mysql(默认)|sqlite|sqlserver|postgresql
	OssType            string   `mapstructure:"oss-type" json:"oss-type" yaml:"oss-type"` // Oss类型
	RouterPrefix       string   `mapstructure:"router-prefix" json:"router-prefix" yaml:"router-prefix"`
	Addr               int      `mapstructure:"addr" json:"addr" yaml:"addr"` // 端口值
	LimitCountIP       int      `mapstructure:"iplimit-count" json:"iplimit-count" yaml:"iplimit-count"`
	LimitTimeIP        int      `mapstructure:"iplimit-time" json:"iplimit-time" yaml:"iplimit-time"`
	UseMultipoint      bool     `mapstructure:"use-multipoint" json:"use-multipoint" yaml:"use-multipoint"`                   // 多点登录拦截
	UseRedis           bool     `mapstructure:"use-redis" json:"use-redis" yaml:"use-redis"`                                  // 使用redis
	UseMongo           bool     `mapstructure:"use-mongo" json:"use-mongo" yaml:"use-mongo"`                                  // 使用mongo
	UseStrictAuth      bool     `mapstructure:"use-strict-auth" json:"use-strict-auth" yaml:"use-strict-auth"`                // 使用树形角色分配模式
	DisableAutoMigrate bool     `mapstructure:"disable-auto-migrate" json:"disable-auto-migrate" yaml:"disable-auto-migrate"` // 自动迁移数据库表结构，生产环境建议设为false，手动迁移
	WorkerID           int      `mapstructure:"worker-id" json:"worker-id" yaml:"worker-id"`                                  // 雪花算法工作节点ID(0-63),多副本部署时各实例必须唯一
	TrustedProxies     []string `mapstructure:"trusted-proxies" json:"trusted-proxies" yaml:"trusted-proxies"`                // 受信任反代 CIDR(空=仅信任直连 peer,ClientIP 忽略 X-Forwarded-For;反代部署须显式配置)
	Ip2RegionDbPath    string   `mapstructure:"ip2region-db-path" json:"ip2region-db-path" yaml:"ip2region-db-path"`           // ip2region xdb 路径(登录/操作日志 IP→地点;空=resource/ip2region.xdb)
}
