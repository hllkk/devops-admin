package system

import "github.com/hllkk/devops-admin/server/global"

// SysWecomBotGroup 企业微信群机器人群登记表(群聊名称+webhook,支持多群)。
// 所有走群机器人的通知场景(晨报/后续预算告警等)共用本表;暂不提供编辑(录错删除重录),
// 软删即时生效(发送按 id 查询自动排除)。
type SysWecomBotGroup struct {
	global.OPS_MODEL
	GroupName  string `json:"groupName" gorm:"size:128;comment:群聊名称"`
	WebhookUrl string `json:"webhookUrl" gorm:"size:512;comment:群机器人webhook地址"`
}

func (SysWecomBotGroup) TableName() string { return "sys_wecom_bot_group" }
