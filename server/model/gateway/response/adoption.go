package response

// AdoptionKpi 覆盖率/采用度 KPI(口径集中定义见 service/gateway/adoption.go 头注释)。
type AdoptionKpi struct {
	TotalUsers      int     `json:"totalUsers"`      // 启用用户总数(分母,含从未使用者)
	ActiveUsers     int     `json:"activeUsers"`     // 激活用户数(期内有 LLM/MCP 调用)
	Coverage        float64 `json:"coverage"`        // 总覆盖率%(激活/启用总数)
	CoverageChange  float64 `json:"coverageChange"`  // 覆盖率环比(百分点,当期-上期;分母同为当前用户快照)
	NewActiveUsers  int     `json:"newActiveUsers"`  // 新增活跃(当期激活且上期未激活)
	PrevActiveUsers int     `json:"prevActiveUsers"` // 上期激活用户数(等长上一期)
	TotalRequests   int     `json:"totalRequests"`   // 总调用数(LLM+MCP)
	DailyRequests   float64 `json:"dailyRequests"`   // 日均调用(按期间天数)
	PerCapitaTokens int64   `json:"perCapitaTokens"` // 人均 token(分母=激活用户数,活跃人均)
	Days            int     `json:"days"`            // 期间天数(闭区间)
}

// AdoptionTrendItem DAU 趋势(按业务日)。
type AdoptionTrendItem struct {
	Date        string `json:"date"`        // 业务日(YYYY-MM-DD)
	ActiveUsers int    `json:"activeUsers"` // 当日活跃用户数(LLM∪MCP 去重)
	Requests    int    `json:"requests"`    // 当日调用数(LLM+MCP)
}

// AdoptionOverview 覆盖率总览(KPI+DAU 趋势)。
type AdoptionOverview struct {
	Kpi   AdoptionKpi          `json:"kpi"`
	Trend []AdoptionTrendItem `json:"trend"`
}

// AdoptionDeptRow 部门覆盖率行(含零调用部门,覆盖率视角与成本明细的差异点)。
// 直挂口径:成员分母按 sys_users.dept_id 计数,消耗含部门 Key(锚点同成本分析)。
type AdoptionDeptRow struct {
	DeptId       int64   `json:"deptId,string"`
	DeptName     string  `json:"deptName"`
	MemberCount  int     `json:"memberCount"`  // 启用成员数(直挂)
	ActiveCount  int     `json:"activeCount"`  // 激活成员数
	Coverage     float64 `json:"coverage"`     // 覆盖率%
	Requests     int     `json:"requests"`     // 调用数(含部门 Key 消耗)
	TotalTokens  int64   `json:"totalTokens"`  // 总 token
	InternalCost float64 `json:"internalCost"` // 内部成本(¥)
}

// AdoptionUserRow 部门成员行(含未激活成员,激活在前——覆盖率下钻兼「未使用人员」清单)。
type AdoptionUserRow struct {
	UserId       int64   `json:"userId,string"`
	UserName     string  `json:"userName"`
	Active       bool    `json:"active"`       // 期内是否有调用
	Requests     int     `json:"requests"`     // 调用数(LLM+MCP)
	TotalTokens  int64   `json:"totalTokens"`  // 总 token
	InternalCost float64 `json:"internalCost"` // 内部成本(¥)
	LastActiveAt string  `json:"lastActiveAt"` // 最后活跃(本地时区 YYYY-MM-DD HH:mm,空=期内无调用)
}

// AdoptionModelRow 模型分布行(LLM 调用;MCP 无模型概念不参与)。
type AdoptionModelRow struct {
	Model        string  `json:"model"`
	Requests     int     `json:"requests"`     // 调用数
	RequestShare float64 `json:"requestShare"` // 调用占比%
	TotalTokens  int64   `json:"totalTokens"`
	InternalCost float64 `json:"internalCost"` // 内部成本(¥)
	CostShare    float64 `json:"costShare"`    // 内部成本占比%
	ActiveUsers  int     `json:"activeUsers"`  // 活跃用户数(去重)
}
