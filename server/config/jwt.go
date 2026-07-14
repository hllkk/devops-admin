package config

type JWT struct {
	SigningKey    string `mapstructure:"signing-key" json:"signing-key" yaml:"signing-key"`           // jwt 签名
	ExpiresTime   string `mapstructure:"expires-time" json:"expires-time" yaml:"expires-time"`        // access token 过期时间
	RefreshExTime string `mapstructure:"refresh-ex-time" json:"refresh-ex-time" yaml:"refresh-ex-time"` // refresh token 过期时间
	BufferTime    string `mapstructure:"buffer-time" json:"buffer-time" yaml:"buffer-time"`           // 已废弃（原滑动续期），保留字段以免 OtherInit 解析 panic
	Issuer        string `mapstructure:"issuer" json:"issuer" yaml:"issuer"`                          // 签发者
}
