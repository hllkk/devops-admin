package system

import "github.com/hllkk/devops-admin/server/global"

// SysNotifyConfig 通知配置(单行表 id=1,启动加载入内存缓存,保存即热更新;对齐前端 NotifySettingConfig)
//
// 字段分两段:
//   - 邮件通知:emailEnabled/emailHost/emailPort/emailUsername/emailPassword/emailFromAddr/emailFromName/emailSSLMode
//   - Webhook:webhookEnabled/webhookUrl/webhookSecret
//   - 企微应用消息/群机器人:渠道可用性开关(群机器人多群清单在 sys_wecom_bot_group 表)
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
	// 企业微信应用消息(凭证复用 sys_auth_config 企微段,此处不重复存)
	WecomPushEnabled     bool   `json:"wecomPushEnabled" gorm:"default:false;comment:启用企微应用消息推送"`
	WecomPushRedirectBase string `json:"wecomPushRedirectBase" gorm:"comment:企微消息跳转基础地址(空则textcard降级纯文本)"`
	WecomPushMaxTargets  int    `json:"wecomPushMaxTargets" gorm:"default:1000;comment:企微推送单次人数上限(超出截断)"`
	// 企业微信群机器人(webhook,markdown 进群;群清单在 sys_wecom_bot_group 表,渠道是否可用由本开关控制)
	WecomBotEnabled bool `json:"wecomBotEnabled" gorm:"default:false;comment:启用企微群机器人"`
	// 事件开关(控制该事件是否走外部渠道;站内通知不受控)。
	// 晨报渠道选择已移至 sys_notify_policy.params(按场景勾选应用消息/群机器人)
	PushBudgetAlertEnabled bool `json:"pushBudgetAlertEnabled" gorm:"default:true;comment:事件开关-预算告警外部推送"`
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
		// 企微应用消息
		WecomPushEnabled:      false,
		WecomPushRedirectBase: "",
		WecomPushMaxTargets:   1000,
		// 群机器人(群清单在 sys_wecom_bot_group)
		WecomBotEnabled: false,
		// 事件开关
		PushBudgetAlertEnabled: true,
	}
}
