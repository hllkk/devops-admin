package request

import (
	"time"

	"github.com/hllkk/devops-admin/server/model/common"
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// AiKeySearch 密钥分页查询(管理员视角，query 传输)。
type AiKeySearch struct {
	commonReq.PageInfo
	KeyType   string `json:"keyType" form:"keyType"`     // 密钥类型(精确)
	OwnerType string `json:"ownerType" form:"ownerType"` // 归属类型(精确)
	OwnerId   int64  `json:"ownerId,string" form:"ownerId"` // 归属ID(0=不限)
	Name      string `json:"name" form:"name"`           // 名称(模糊)
	ScenarioId int64 `json:"scenarioId,string" form:"scenarioId"` // 场景ID(0=不限)
	IsActive  *bool  `json:"isActive" form:"isActive"`   // 是否启用(精确,nil=不限)
}

// AiKeyOperateParams 密钥新增/修改(管理员视角)。
// create 时 aiKeyId 为空；keyType/ownerType/ownerId 创建后不可改(改=删旧建新)。
// models 为授权 modelKey 列表(推送时自动追加 anthropic 变体)；
// budgetHardLimit=true 时 budgetLimit 下发为 LiteLLM max_budget；停用=max_budget=0。
type AiKeyOperateParams struct {
	AiKeyId          int64           `json:"aiKeyId,string" form:"aiKeyId"`    // 密钥ID(新增为空)
	KeyType          string          `json:"keyType" form:"keyType"`          // 密钥类型(创建必填)
	OwnerType        string          `json:"ownerType" form:"ownerType"`      // 归属类型(创建必填)
	OwnerId          int64           `json:"ownerId,string" form:"ownerId"`   // 归属ID(创建必填)
	Name             string          `json:"name" form:"name"`                // 名称
	Description      string          `json:"description" form:"description"`  // 描述
	ScenarioId       *int64          `json:"scenarioId,string" form:"scenarioId"` // 场景ID(场景Key可填;nil=清空/主Key恒0;须为未软删且启用的场景)
	Models           []string        `json:"models" form:"models"`           // 授权模型(modelKey列表)
	Mcps             []string        `json:"mcps" form:"mcps"`               // 授权MCP(serverName列表,nil=不改,空=清空)
	Skills           []string        `json:"skills"`                         // 授权Skill(skillId字符串列表,nil=不改,空=清空)
	ModelBudgets     map[string]any   `json:"modelBudgets" form:"modelBudgets"` // 按模型预算
	BudgetLimit      *float64        `json:"budgetLimit" form:"budgetLimit"`  // 预算上限
	BudgetHardLimit  *bool           `json:"budgetHardLimit" form:"budgetHardLimit"` // 硬限
	BudgetDuration   string          `json:"budgetDuration" form:"budgetDuration"`   // 预算周期
	RateLimitMode    string          `json:"rateLimitMode" form:"rateLimitMode"`     // 限流模式
	TpmLimit         *int            `json:"tpmLimit" form:"tpmLimit"`         // 全局TPM
	RpmLimit         *int            `json:"rpmLimit" form:"rpmLimit"`         // 全局RPM
	ModelLimits      map[string]any   `json:"modelLimits" form:"modelLimits"`  // per-model限流
	IsActive         *bool           `json:"isActive" form:"isActive"`         // 是否启用
	ExpiresAt        *time.Time      `json:"expiresAt" form:"expiresAt"`       // 过期时间(nil=永不过期,覆盖式更新)
}

// AiKeyBatchCreateParams 批量开通个人主 Key(管理员创建制的效率件)：按用户ID列表或按部门
// (取部门下全部用户)二选一；对每个目标用户：已有 personal_main 跳过，无则按主 Key 默认
// 语义创建(公开模型、name=main)。部分失败不中断，结果经 data 标记返回。
type AiKeyBatchCreateParams struct {
	UserIds common.Int64StringSlice `json:"userIds"`              // 目标用户ID列表(与 deptId 至少一项;雪花id字符串/数字元素兼容)
	DeptId  *int64                  `json:"deptId,string" form:"deptId"` // 按部门开通(优先,取部门下全部用户)
}

// AiKeyBatchSceneCreateParams 批量建个人场景 Key(管理员效率件)：目标同批量开通(deptId ∪
// userIds)；nameTemplate 必填、支持 {username}/{nickname} 占位符逐用户渲染；资源配置
// (models/mcps/skills/预算/限流/过期)整体作为模板套到每个目标用户——前端「复制主 Key 模板」
// 即以某主 Key 的配置预填本表单。同名/停用用户计入失败明细，部分成功语义。
type AiKeyBatchSceneCreateParams struct {
	UserIds          common.Int64StringSlice `json:"userIds"`              // 目标用户ID列表(与 deptId 至少一项)
	DeptId           *int64                  `json:"deptId,string" form:"deptId"` // 按部门(优先,取部门下全部用户)
	NameTemplate     string                  `json:"nameTemplate"`         // 名称模板(必填,{username}/{nickname}占位)
	Description      string                  `json:"description"`          // 描述
	ScenarioId       *int64                  `json:"scenarioId,string"`    // 场景ID(可选,须为启用中场景)
	Models           []string                `json:"models"`               // 授权模型(空=场景Key无默认,显式空数组)
	Mcps             []string                `json:"mcps"`                 // 授权MCP(serverName列表)
	Skills           []string                `json:"skills"`               // 授权Skill(skillId字符串列表)
	ModelBudgets     map[string]any          `json:"modelBudgets"`         // 按模型预算
	BudgetLimit      *float64                `json:"budgetLimit"`          // 预算上限
	BudgetHardLimit  *bool                   `json:"budgetHardLimit"`      // 硬限
	BudgetDuration   string                  `json:"budgetDuration"`       // 预算周期
	RateLimitMode    string                  `json:"rateLimitMode"`        // 限流模式
	TpmLimit         *int                    `json:"tpmLimit"`             // 全局TPM
	RpmLimit         *int                    `json:"rpmLimit"`             // 全局RPM
	ModelLimits      map[string]any          `json:"modelLimits"`          // per-model限流
	IsActive         *bool                   `json:"isActive"`             // 是否启用(nil=默认启用)
	ExpiresAt        *time.Time              `json:"expiresAt"`            // 过期时间(nil=永不过期)
}
