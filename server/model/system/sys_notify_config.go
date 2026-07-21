package system

import "github.com/hllkk/devops-admin/server/global"

// SysNotifyConfig 通知配置(单行表 id=1,启动加载入内存缓存,保存即热更新;对齐前端 NotifySettingConfig)
//
// 字段分两段:
//   - 邮件通知:emailEnabled/emailHost/emailPort/emailUsername/emailPassword/emailFromAddr/emailFromName/emailSSLMode
//   - Webhook:webhookEnabled/webhookUrl/webhookSecret
type SysNotifyConfig struct {
	global.OPS_MODEL
	// 邮件通知
	EmailEnabled  bool   `json:"emailEnabled" gorm:"default:false;comment:启用邮件通知"`
	EmailHost     string `json:"emailHost" gorm:"comment:SMTP服务器"`
	EmailPort     int    `json:"emailPort" gorm:"default:465;comment:SMTP端口"`
	EmailUsername string `json:"emailUsername" gorm:"comment:SMTP认证用户名"`
	EmailPassword string `json:"emailPassword" gorm:"comment:SMTP认证密码"`
	EmailFromAddr string `json:"emailFromAddr" gorm:"comment:发件人邮箱地址"`
	EmailFromName string `json:"emailFromName" gorm:"comment:发件人显示名称"`
	// none / ssl / starttls
	EmailSSLMode string `json:"emailSSLMode" gorm:"default:ssl;comment:加密方式:none/ssl/starttls"`
	// Webhook
	WebhookEnabled bool   `json:"webhookEnabled" gorm:"default:false;comment:启用Webhook通知"`
	WebhookUrl     string `json:"webhookUrl" gorm:"comment:Webhook推送地址"`
	WebhookSecret  string `json:"webhookSecret" gorm:"comment:Webhook签名密钥(可选)"`
}

func (SysNotifyConfig) TableName() string {
	return "sys_notify_config"
}

// DefaultNotifyConfig 返回默认通知配置,调用方负责设 id=1
func DefaultNotifyConfig() SysNotifyConfig {
	return SysNotifyConfig{
		EmailEnabled:  false,
		EmailHost:     "",
		EmailPort:     465,
		EmailUsername: "",
		EmailPassword: "",
		EmailFromAddr: "",
		EmailFromName: "",
		EmailSSLMode:  "ssl",
		// Webhook
		WebhookEnabled: false,
		WebhookUrl:     "",
		WebhookSecret:  "",
	}
}
