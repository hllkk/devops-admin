package system

import (
	"time"

	"github.com/hllkk/devops-admin/server/config"
	"github.com/hllkk/devops-admin/server/global"
)

// SysSecurityConfig 安全配置(单行表 固定 id=1 启动加载入内存缓存 保存即热更新;对齐前端 SecuritySettingConfig)
//
// 字段对齐策略:
//   - 六段(验证码 Captcha*/密码复杂度 Password*/失败锁定 LoginFailLock*/访问控制 IpValidation*/限流 Limit*/密码过期 PwdExpire*)
//     的 json tag 均对齐前端 SecuritySettingConfig,前端「安全配置」页 8 个 tab 直接消费
//   - 登录链路(captcha/限流中间件/失败锁定/密码过期/密码复杂度校验)同样读这张表
type SysSecurityConfig struct {
	global.OPS_MODEL
	// 验证码(登录链路验证码生成用,前端「安全配置-验证码」tab 直接消费本段)
	CaptchaEnabled   bool   `json:"captchaEnabled" gorm:"default:true;comment:验证码总开关"`
	CaptchaType      string `json:"captchaType" gorm:"default:click;comment:验证码类型 image|click|slide|rotate"`
	CaptchaOpen      int    `json:"captchaOpen" gorm:"default:0;comment:错误N次后出验证码 0=每次都要"`
	CaptchaTimeout   int    `json:"captchaTimeout" gorm:"default:3600;comment:防爆破计数缓存超时(秒)"`
	CaptchaTolerance int    `json:"captchaTolerance" gorm:"default:5;comment:命中容差 click/slide像素 rotate角度"`
	KeyLong          int    `json:"keyLong" gorm:"default:4;comment:image验证码长度/click文字点选字符数"`
	ImgWidth         int    `json:"imgWidth" gorm:"default:240;comment:验证码宽度"`
	ImgHeight        int    `json:"imgHeight" gorm:"default:80;comment:验证码高度"`
	// 密码复杂度(对齐前端 SecuritySettingConfig.password*)
	PasswordMinLength      int  `json:"passwordMinLength" gorm:"default:8;comment:密码最小长度"`
	PasswordRequireUpper   bool `json:"passwordRequireUppercase" gorm:"default:false;comment:需大写字母"`
	PasswordRequireLower   bool `json:"passwordRequireLowercase" gorm:"default:false;comment:需小写字母"`
	PasswordRequireDigit   bool `json:"passwordRequireDigit" gorm:"default:false;comment:需数字"`
	PasswordRequireSpecial bool `json:"passwordRequireSpecial" gorm:"default:false;comment:需特殊字符"`
	// 登录失败锁定(对齐前端 SecuritySettingConfig.loginFailLock*)
	LoginFailLockCount int `json:"loginFailLockCount" gorm:"default:5;comment:失败次数阈值"`
	LoginFailLockTime  int `json:"loginFailLockTime" gorm:"default:30;comment:锁定时长(分钟)"`
	// IP 校验(对齐前端 SecuritySettingConfig.ipValidation*)
	IpValidationEnabled bool   `json:"ipValidationEnabled" gorm:"default:false;comment:是否开启IP校验"`
	IpValidationMode    string `json:"ipValidationMode" gorm:"default:blacklist;comment:IP校验模式 blacklist/whitelist"`
	IpBlacklist         string `json:"ipBlacklist" gorm:"comment:IP黑名单(逗号/换行分隔)"`
	IpWhitelist         string `json:"ipWhitelist" gorm:"comment:IP白名单(逗号/换行分隔)"`
	// 限流(后端登录链路用,前端不直接消费)
	LimitEnable bool `json:"limitEnable" gorm:"default:false;comment:是否开启限流"`
	LimitWindow int  `json:"limitWindow" gorm:"default:60;comment:限流窗口(秒)"`
	LimitCount  int  `json:"limitCount" gorm:"default:30;comment:窗口内最大次数"`
	// 密码过期(后端登录链路用,前端不直接消费)
	PwdExpireEnable bool `json:"pwdExpireEnable" gorm:"default:false;comment:是否开启密码过期"`
	PwdExpireDays   int  `json:"pwdExpireDays" gorm:"default:90;comment:密码有效天数"`
}

func (SysSecurityConfig) TableName() string {
	return "sys_security_config"
}

// CaptchaTimeoutDuration 防爆破计数缓存超时
func (c SysSecurityConfig) CaptchaTimeoutDuration() time.Duration {
	return time.Duration(c.CaptchaTimeout) * time.Second
}

// LockDurationTimeout 锁定时长
func (c SysSecurityConfig) LockDurationTimeout() time.Duration {
	return time.Duration(c.LoginFailLockTime) * time.Minute
}

// LimitWindowDuration 限流窗口
func (c SysSecurityConfig) LimitWindowDuration() time.Duration {
	return time.Duration(c.LimitWindow) * time.Second
}

// DefaultSecurityConfig 由 config.yaml 的 captcha 生成默认单行配置 调用方负责设 id=1
func DefaultSecurityConfig(captcha config.Captcha) SysSecurityConfig {
	return SysSecurityConfig{
		CaptchaEnabled:         captcha.Enable,
		CaptchaType:            captcha.Type,
		CaptchaOpen:            captcha.OpenCaptcha,
		CaptchaTimeout:         captcha.OpenCaptchaTimeOut,
		CaptchaTolerance:       captcha.Tolerance,
		KeyLong:                captcha.KeyLong,
		ImgWidth:               captcha.ImgWidth,
		ImgHeight:              captcha.ImgHeight,
		PasswordMinLength:      8,
		PasswordRequireUpper:   false,
		PasswordRequireLower:   false,
		PasswordRequireDigit:   false,
		PasswordRequireSpecial: false,
		LoginFailLockCount:     5,
		LoginFailLockTime:      30,
		IpValidationEnabled:    false,
		IpValidationMode:       "blacklist",
		IpBlacklist:            "",
		IpWhitelist:            "",
		LimitEnable:            false,
		LimitWindow:            60,
		LimitCount:             30,
		PwdExpireEnable:        false,
		PwdExpireDays:          90,
	}
}
