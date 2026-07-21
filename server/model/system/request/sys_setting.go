package request

import "github.com/hllkk/devops-admin/server/model/system"

// SettingConfig 系统设置聚合配置(GET/PUT /system/setting 请求与响应体,对齐前端 Api.System.Setting)
// general 段落表 sys_general_config(系统信息/账户默认/日志清理);security 段落表 sys_security_config(六段安全策略)
// general 与 security 任一可选:前端聚合页一次性提交两段,后端分发到两张配置表
type SettingConfig struct {
	General  *system.SysGeneralConfig  `json:"general,omitempty"`
	Security *system.SysSecurityConfig `json:"security,omitempty"`
}

// PublicSetting 公开系统设置(GET /system/setting/public 响应体,登录页用,免鉴权脱敏)。
// 只暴露无风险字段:常规配置的系统信息 + 安全配置的验证码段;不含密码策略/IP名单/限流/密码过期等。
type PublicSetting struct {
	// 系统信息(sys_general_config)
	SystemName        string `json:"systemName"`
	SystemDescription string `json:"systemDescription"`
	LogoUrl           string `json:"logoUrl"`
	FaviconUrl        string `json:"faviconUrl"`
	// 验证码(sys_security_config.Captcha*,登录页验证码渲染用)
	CaptchaEnabled bool   `json:"captchaEnabled"`
	CaptchaType    string `json:"captchaType"`
	CaptchaOpen    int    `json:"captchaOpen"`
	KeyLong        int    `json:"keyLong"`
	ImgWidth       int    `json:"imgWidth"`
	ImgHeight      int    `json:"imgHeight"`
}
