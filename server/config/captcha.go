package config

type Captcha struct {
	KeyLong            int `mapstructure:"key-long" json:"key-long" yaml:"key-long"`                                     // 验证码长度
	ImgWidth           int `mapstructure:"img-width" json:"img-width" yaml:"img-width"`                                  // 验证码宽度
	ImgHeight          int `mapstructure:"img-height" json:"img-height" yaml:"img-height"`                               // 验证码高度
	OpenCaptcha        int `mapstructure:"open-captcha" json:"open-captcha" yaml:"open-captcha"`                         // 防爆破验证码开启此数，0代表每次登录都需要验证码，其他数字代表错误密码次数，如3代表错误三次后出现验证码
	OpenCaptchaTimeOut int `mapstructure:"open-captcha-timeout" json:"open-captcha-timeout" yaml:"open-captcha-timeout"` // 防爆破验证码超时时间，单位：s(秒)
	GoCaptcha          GoCaptcha `mapstructure:"go-captcha" json:"go-captcha" yaml:"go-captcha"`                         // go-captcha 行为验证码配置
}

// GoCaptcha go-captcha 行为验证码配置（click 点选 / slide 滑动 / rotate 旋转，可通过 type 切换）
type GoCaptcha struct {
	Enabled       bool             `mapstructure:"enabled" json:"enabled" yaml:"enabled"`                       // 总开关；false 时登录不要求验证码
	Type          string           `mapstructure:"type" json:"type" yaml:"type"`                                 // 当前启用的类型：click | slide | rotate
	Store         string           `mapstructure:"store" json:"store" yaml:"store"`                             // 答案存储介质：redis（不可用时自动降级 memory）| memory
	KeyPrefix     string           `mapstructure:"key-prefix" json:"key-prefix" yaml:"key-prefix"`             // redis key 前缀
	ExpireSeconds int              `mapstructure:"expire-seconds" json:"expire-seconds" yaml:"expire-seconds"` // 验证码有效期（秒）
	Trigger       GoCaptchaTrigger `mapstructure:"trigger" json:"trigger" yaml:"trigger"`                       // 触发策略
	Slide         GoCaptchaSlide   `mapstructure:"slide" json:"slide" yaml:"slide"`                             // 滑动拼图参数
	Click         GoCaptchaClick   `mapstructure:"click" json:"click" yaml:"click"`                             // 文字点选参数
	Rotate        GoCaptchaRotate  `mapstructure:"rotate" json:"rotate" yaml:"rotate"`                          // 旋转参数
}

// GoCaptchaTrigger 触发策略：决定登录时是否要求验证码
type GoCaptchaTrigger struct {
	Mode          string `mapstructure:"mode" json:"mode" yaml:"mode"`                     // threshold(阈值触发) | always(始终要求) | off(关闭)
	FailThreshold int    `mapstructure:"fail-threshold" json:"fail-threshold" yaml:"fail-threshold"` // 阈值模式：同一账号/IP 在 fail-window 内连续失败达到此次数后必须验证码
	FailWindow    int    `mapstructure:"fail-window" json:"fail-window" yaml:"fail-window"`       // 失败计数窗口（秒）
}

// GoCaptchaSlide 滑动拼图参数
type GoCaptchaSlide struct {
	Tolerance int `mapstructure:"tolerance" json:"tolerance" yaml:"tolerance"` // 拖动到位容差（像素）
}

// GoCaptchaClick 文字点选参数
type GoCaptchaClick struct {
	Padding int `mapstructure:"padding" json:"padding" yaml:"padding"` // 点选命中容差（像素）
}

// GoCaptchaRotate 旋转参数
type GoCaptchaRotate struct {
	Tolerance int `mapstructure:"tolerance" json:"tolerance" yaml:"tolerance"` // 旋转归位容差（角度）
}
