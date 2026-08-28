package response

// DashboardOverview 看板总览（按时间范围汇总）。
type DashboardOverview struct {
	TotalRequests    int     `json:"totalRequests"`    // 总请求数
	TotalCost         float64 `json:"totalCost"`        // 总成本(¥,external)
	InternalCost      float64 `json:"internalCost"`     // 内部成本(¥,P1同external)
	TotalTokens       int64   `json:"totalTokens"`     // 总token
	InputTokens       int64   `json:"inputTokens"`      // 输入token
	OutputTokens      int64   `json:"outputTokens"`    // 输出token
	CacheReadTokens   int64   `json:"cacheReadTokens"`  // 缓存读token
	BudgetUsedTotal   float64 `json:"budgetUsedTotal"`  // 预算已用总额
	BudgetLimitTotal  float64 `json:"budgetLimitTotal"` // 预算限额总额(有预算的Key)
}

// TrendItem 成本趋势（按日）。
type TrendItem struct {
	Date     string  `json:"date"`     // 业务日(YYYY-MM-DD)
	Cost     float64 `json:"cost"`     // 当日成本(¥)
	Requests int     `json:"requests"` // 当日请求数
	Tokens   int64   `json:"tokens"`   // 当日总token
}

// TopItem 排行项（按维度 user/model/aiKey，排序键 cost/requests/tokens）。
type TopItem struct {
	Name     string  `json:"name"`     // 维度值(用户名/模型名/Key名)
	Cost     float64 `json:"cost"`     // 成本(¥)
	Requests int     `json:"requests"` // 请求数
	Tokens   int64   `json:"tokens"`   // 总token
}

// BudgetItem 预算执行项（按 Key）。
type BudgetItem struct {
	AiKeyId      int64   `json:"aiKeyId,string"` // 密钥ID
	Name         string  `json:"name"`           // 密钥名称
	OwnerName    string  `json:"ownerName"`       // 归属(用户名/部门名)
	BudgetLimit  float64 `json:"budgetLimit"`     // 预算限额(¥,0=不限)
	BudgetUsed   float64 `json:"budgetUsed"`      // 已用(¥)
	UsageRate    float64 `json:"usageRate"`       // 执行率(%)
	HardLimit    bool    `json:"hardLimit"`       // 硬限
	IsActive     bool    `json:"isActive"`        // 是否启用
}
