package request

import (
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
	ModelBudgets     map[string]any   `json:"modelBudgets" form:"modelBudgets"` // 按模型预算
	BudgetLimit      *float64        `json:"budgetLimit" form:"budgetLimit"`  // 预算上限
	BudgetHardLimit  *bool           `json:"budgetHardLimit" form:"budgetHardLimit"` // 硬限
	BudgetDuration   string          `json:"budgetDuration" form:"budgetDuration"`   // 预算周期
	RateLimitMode    string          `json:"rateLimitMode" form:"rateLimitMode"`     // 限流模式
	TpmLimit         *int            `json:"tpmLimit" form:"tpmLimit"`         // 全局TPM
	RpmLimit         *int            `json:"rpmLimit" form:"rpmLimit"`         // 全局RPM
	ModelLimits      map[string]any   `json:"modelLimits" form:"modelLimits"`  // per-model限流
	IsActive         *bool           `json:"isActive" form:"isActive"`         // 是否启用
}
