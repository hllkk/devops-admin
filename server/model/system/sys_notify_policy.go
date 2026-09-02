package system

import (
	"encoding/json"

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

// MorningReportParams 晨报场景私有参数(存 sys_notify_policy.params)。
// 渠道按场景勾选:勾 wecomApp 只发企微应用消息,勾 wecomBot 只发群机器人(可同时勾),
// 均不勾则仅站内通知;渠道总开关(sys_notify_config)仍是前置条件(未启用渠道勾了也不发)。
// 正文模板为 Go text/template 语法,变量见 MorningTemplateVars;留空用默认模板,渲染失败降级默认并记日志。
type MorningReportParams struct {
	WecomApp         bool     `json:"wecomApp"`                    // 勾选企微应用消息渠道
	WecomBot         bool     `json:"wecomBot"`                    // 勾选企微群机器人渠道
	BotGroupIds      []string `json:"botGroupIds,omitempty"`      // 群机器人目标群(sys_wecom_bot_group 主键字符串,前端 IdType)
	ContentTemplate  string   `json:"contentTemplate,omitempty"`  // 纯文本正文自定义模板(站内+企微应用消息)
	MarkdownTemplate string   `json:"markdownTemplate,omitempty"` // markdown 正文自定义模板(群机器人)
}

// ParseMorningReportParams 解析 params jsonb 为晨报参数;空 params 返回零值(仅站内)。
// 解析失败返回错误由调用方感知(坏数据静默降级会导致勾选的渠道不发而无感知)。
func ParseMorningReportParams(raw datatypes.JSON) (MorningReportParams, error) {
	var p MorningReportParams
	if len(raw) == 0 || string(raw) == "null" {
		return p, nil
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, err
	}
	return p, nil
}

func (SysNotifyPolicy) TableName() string { return "sys_notify_policy" }
