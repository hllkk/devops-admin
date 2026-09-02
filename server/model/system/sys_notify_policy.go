package system

import (
	"gorm.io/datatypes"

	"github.com/hllkk/devops-admin/server/global"
)

// SysNotifyPolicy 通知策略(按场景 scene_key 一行,配置定时内容型通知的目标与参数)。
// 第一期场景:token_plan_morning(AI 网关 TokenPlan 工作日晨报)。
// 事件是否走外部渠道由 sys_notify_config 的事件开关控制,本表只管"发给谁+发不发"。
type SysNotifyPolicy struct {
	global.OPS_MODEL
	SceneKey   string         `json:"sceneKey" gorm:"uniqueIndex;size:64;comment:场景标识(token_plan_morning)"`
	Enabled    bool           `json:"enabled" gorm:"default:false;comment:启用该场景通知"`
	TargetType string         `json:"targetType" gorm:"size:16;default:users;comment:目标类型(all/depts/users,depts含子部门)"`
	TargetIds  datatypes.JSON `json:"targetIds" gorm:"type:jsonb;comment:目标ID列表(部门ID或用户ID, targetType=all时忽略)"`
	SendTime   string         `json:"sendTime" gorm:"size:5;default:'08:33';comment:发送时间(HH:mm工作日,保存时同步调度TokenPlanMorningReport)"`
	Params     datatypes.JSON `json:"params" gorm:"type:jsonb;comment:场景私有参数(预留)"`
}

// 通知策略场景
const (
	NotifySceneTokenPlanMorning = "token_plan_morning" // TokenPlan 工作日晨报
)

// 通知策略目标类型
const (
	NotifyPolicyTargetAll   = "all"
	NotifyPolicyTargetDepts = "depts"
	NotifyPolicyTargetUsers = "users"
)

func (SysNotifyPolicy) TableName() string { return "sys_notify_policy" }
