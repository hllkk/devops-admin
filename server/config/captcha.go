package config

type Captcha struct {
	Enable             bool   `mapstructure:"enable" json:"enable" yaml:"enable"`                                              // 验证码总开关：false 时 /auth/captcha 永远返回 captchaEnabled=false
	Type               string `mapstructure:"type" json:"type" yaml:"type"`                                                    // 验证码类型：image 传统图形验证码 | click/slide/rotate go-captcha 行为验证码
	KeyLong            int    `mapstructure:"key-long" json:"key-long" yaml:"key-long"`                                       // image 验证码长度 / click 文字点选主图字符数
	ImgWidth           int    `mapstructure:"img-width" json:"img-width" yaml:"img-width"`                                    // image 验证码宽度
	ImgHeight          int    `mapstructure:"img-height" json:"img-height" yaml:"img-height"`                                 // image 验证码高度
	Tolerance          int    `mapstructure:"tolerance" json:"tolerance" yaml:"tolerance"`                                    // 命中容差：click/slide 像素、rotate 角度，<=0 取默认 5
	OpenCaptcha        int    `mapstructure:"open-captcha" json:"open-captcha" yaml:"open-captcha"`                           // 触发阈值：0 每次都要 N 失败N次后触发(enable=true 时生效)
	OpenCaptchaTimeOut int    `mapstructure:"open-captcha-timeout" json:"open-captcha-timeout" yaml:"open-captcha-timeout"`   // 触发计数窗口(秒)，复用为验证码答案 TTL
}
