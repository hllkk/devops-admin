package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// BudgetRuleSearch 预算规则分页查询(P3，query 传输)。
type BudgetRuleSearch struct {
	commonReq.PageInfo
	ScopeType string `json:"scopeType" form:"scopeType"` // 维度(dept/user,空=全部)
	IsActive  *bool  `json:"isActive" form:"isActive"`   // 启停状态(nil=全部)
}

// BudgetRuleOperateParams 预算规则新增/修改(P3)。
type BudgetRuleOperateParams struct {
	RuleId          int64   `json:"ruleId,string"`         // 规则ID(修改时必填)
	ScopeType       string  `json:"scopeType"`             // 维度(dept/user)
	ScopeId         int64   `json:"scopeId,string"`        // 对象ID(部门ID/用户ID)
	BudgetLimit     float64 `json:"budgetLimit"`            // 预算上限(¥,0=不限)
	BudgetHardLimit bool    `json:"budgetHardLimit"`        // 硬限
	BudgetDuration  string  `json:"budgetDuration"`         // 预算周期(1d/7d/30d)
	SoftWarnPercent int     `json:"softWarnPercent"`        // 软限预警阈值(%)
	IsActive        *bool   `json:"isActive"`               // 启停(nil=不改)
}
