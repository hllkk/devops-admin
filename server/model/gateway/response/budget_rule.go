package response

// BudgetRuleView 预算规则视图(scopeName 后端回填:部门名/用户名;budgetUsed/budgetUsedPercent 读时实时聚合)。
type BudgetRuleView struct {
	RuleId             int64   `json:"ruleId,string"`     // 规则ID
	ScopeType          string  `json:"scopeType"`          // 维度
	ScopeId            int64   `json:"scopeId,string"`     // 对象ID
	ScopeName          string  `json:"scopeName"`          // 对象名(部门名/用户名)
	BudgetLimit        float64 `json:"budgetLimit"`         // 预算上限(¥,0=不限)
	BudgetUsed         float64 `json:"budgetUsed"`          // 已用(¥,读时聚合)
	BudgetUsedPercent  float64 `json:"budgetUsedPercent"`   // 执行率(%,限>0)
	BudgetHardLimit    bool    `json:"budgetHardLimit"`     // 硬限
	BudgetDuration     string  `json:"budgetDuration"`      // 预算周期
	SoftWarnPercent    int     `json:"softWarnPercent"`     // 软限阈值(%)
	IsActive           bool    `json:"isActive"`            // 是否启用
	IsSoftWarn         bool    `json:"isSoftWarn"`          // 是否已触发软限预警(本周期)
	IsHardLimited      bool    `json:"isHardLimited"`       // 是否已触发硬限超限(本周期)
}
