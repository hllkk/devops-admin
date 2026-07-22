package config

type Server struct {
	System       System          `mapstructure:"system" json:"system" yaml:"system"`
	JWT          JWT             `mapstructure:"jwt" json:"jwt" yaml:"jwt"`
	Zap          Zap             `mapstructure:"zap" json:"zap" yaml:"zap"`
	AutoCode     Autocode        `mapstructure:"autocode" json:"autocode" yaml:"autocode"`
	Mysql        Mysql           `mapstructure:"mysql" json:"mysql" yaml:"mysql"`
	Mssql        Mssql           `mapstructure:"mssql" json:"mssql" yaml:"mssql"`
	Pgsql        Pgsql           `mapstructure:"pgsql" json:"pgsql" yaml:"pgsql"`
	Oracle       Oracle          `mapstructure:"oracle" json:"oracle" yaml:"oracle"`
	Sqlite       Sqlite          `mapstructure:"sqlite" json:"sqlite" yaml:"sqlite"`
	DBList       []SpecializedDB `mapstructure:"db-list" json:"db-list" yaml:"db-list"`
	Media        Media           `mapstructure:"media" json:"media" yaml:"media"`
	Local        Local           `mapstructure:"local" json:"local" yaml:"local"`
	Qiniu        Qiniu           `mapstructure:"qiniu" json:"qiniu" yaml:"qiniu"`
	TencentCOS   TencentCOS      `mapstructure:"tencent-cos" json:"tencent-cos" yaml:"tencent-cos"`
	AliyunOSS    AliyunOSS       `mapstructure:"aliyun-oss" json:"aliyun-oss" yaml:"aliyun-oss"`
	HuaWeiObs    HuaWeiObs       `mapstructure:"hua-wei-obs" json:"hua-wei-obs" yaml:"hua-wei-obs"`
	AwsS3        AwsS3           `mapstructure:"aws-s3" json:"aws-s3" yaml:"aws-s3"`
	CloudflareR2 CloudflareR2    `mapstructure:"cloudflare-r2" json:"cloudflare-r2" yaml:"cloudflare-r2"`
	Minio        Minio           `mapstructure:"minio" json:"minio" yaml:"minio"`
	Redis        Redis           `mapstructure:"redis" json:"redis" yaml:"redis"`
	RedisList    []Redis         `mapstructure:"redis-list" json:"redis-list" yaml:"redis-list"`
	Mongo        Mongo           `mapstructure:"mongo" json:"mongo" yaml:"mongo"`
	Captcha      Captcha         `mapstructure:"captcha" json:"captcha" yaml:"captcha"`
	Cors         CORS            `mapstructure:"cors" json:"cors" yaml:"cors"`
	MCP          MCP             `mapstructure:"mcp" json:"mcp" yaml:"mcp"`
	App          App             `mapstructure:"app" json:"app" yaml:"app"`
	Ai           Ai              `mapstructure:"ai" json:"ai" yaml:"ai"`
	// Email        Email           `mapstructure:"email" json:"email" yaml:"email"`
	// Excel        Excel           `mapstructure:"excel" json:"excel" yaml:"excel"`
	// DiskList     []DiskList      `mapstructure:"disk-list" json:"disk-list" yaml:"disk-list"`

}
