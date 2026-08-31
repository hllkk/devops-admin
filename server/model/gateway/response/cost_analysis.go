package response

import "github.com/hllkk/devops-admin/server/model/gateway"

// CostKpi 成本分析 KPI(随筛选联动，含等长上一期环比)。
// 成本均为 ¥(库内口径：四键单价 × token / 1e6)。
type CostKpi struct {
	InternalCost     float64 `json:"internalCost"`     // 内部成本合计
	ExternalCost     float64 `json:"externalCost"`     // 外部成本合计
	CostDiff         float64 `json:"costDiff"`         // 结算差额(外部-内部)
	DailyAvgInternal float64 `json:"dailyAvgInternal"` // 日均内部成本(按期间天数)
	InternalChange   float64 `json:"internalChange"`   // 内部成本环比%(上期为0时给0)
	ExternalChange   float64 `json:"externalChange"`   // 外部成本环比%(上期为0时给0)
	TotalRequests    int     `json:"totalRequests"`    // 总请求数
	TotalTokens      int64   `json:"totalTokens"`      // 总token
	Days             int     `json:"days"`             // 期间天数(闭区间)
}

// CostTrendItem 成本趋势(按业务日，内/外双线)。
type CostTrendItem struct {
	Date         string  `json:"date"`         // 业务日(YYYY-MM-DD)
	InternalCost float64 `json:"internalCost"` // 内部成本
	ExternalCost float64 `json:"externalCost"` // 外部成本
	Requests     int     `json:"requests"`     // 请求数
	Tokens       int64   `json:"tokens"`       // 总token
}

// CostOverview 成本分析总览(KPI+趋势)。
type CostOverview struct {
	Kpi   CostKpi          `json:"kpi"`
	Trend []CostTrendItem `json:"trend"`
}

// CostDetailRow 成本明细行(六维共用：department/user/model/aiKey/provider/date)。
// Value 为维度原始值(部门/用户/Key 的 ID 字符串，其余同展示名)，供下钻/跳日志带参。
type CostDetailRow struct {
	Label            string  `json:"label"`            // 展示名(部门名/用户名/模型名/Key名/供应商/日期)
	Value            string  `json:"value"`            // 维度原始值
	Requests         int     `json:"requests"`         // 请求数
	PromptTokens     int64   `json:"promptTokens"`     // 输入token
	CompletionTokens int64   `json:"completionTokens"` // 输出token
	TotalTokens      int64   `json:"totalTokens"`      // 总token
	InternalCost     float64 `json:"internalCost"`     // 内部成本
	ExternalCost     float64 `json:"externalCost"`     // 外部成本
	CostDiff         float64 `json:"costDiff"`         // 结算差额(外部-内部)
	ActiveUsers      int     `json:"activeUsers"`      // 活跃用户数(去重，user_id>0)
	PerCapita        float64 `json:"perCapita"`        // 人均内部成本(分母=活跃用户数)
}

// CostScopeUserRow 部门下钻成员行(直挂口径，保证部门行=子和)。
// UserId=0 为该部门「部门Key消耗/未归因」合并行。
type CostScopeUserRow struct {
	UserId           int64   `json:"userId,string"`    // 用户ID(0=部门Key/未归因合并行)
	UserName         string  `json:"userName"`         // 用户昵称
	Requests         int     `json:"requests"`         // 请求数
	TotalTokens      int64   `json:"totalTokens"`      // 总token
	InternalCost     float64 `json:"internalCost"`     // 内部成本
	ExternalCost     float64 `json:"externalCost"`     // 外部成本
}

// McpLogView MCP 调用日志视图(userName/aiKeyName 后端回填;ServerName 落库已冗余无需回填)。
type McpLogView struct {
	gateway.McpLog
	UserName  string `json:"userName"`  // 归因用户昵称
	AiKeyName string `json:"aiKeyName"` // 归因密钥名
}
