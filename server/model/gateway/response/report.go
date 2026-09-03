package response

import (
	"time"
)

// ReportKpi 报告 KPI = 采用度 KPI + 成本三数(平铺序列化)。
type ReportKpi struct {
	AdoptionKpi
	InternalCost float64 `json:"internalCost"` // 内部成本合计(¥)
	ExternalCost float64 `json:"externalCost"` // 外部成本合计(¥)
	CostDiff     float64 `json:"costDiff"`     // 结算差额(¥)
}

// ReportContent 报告结构化内容(落库 content 列的 JSON 形态)。
type ReportContent struct {
	Kpi       ReportKpi          `json:"kpi"`
	DeptRows  []AdoptionDeptRow  `json:"deptRows"`  // 部门覆盖率 Top20
	ModelRows []AdoptionModelRow `json:"modelRows"` // 模型分布 Top20
	TopUsers  []CostDetailRow    `json:"topUsers"`  // 用户 Top10(成本口径)
}

// EfficiencyReportView 报告视图。
// 列表不带 Content/ContentMd(大字段),详情才解析填充;CreatorName 批量回填。
type EfficiencyReportView struct {
	ReportId    int64          `json:"reportId,string"`
	ReportType  string         `json:"reportType"`
	PeriodStart string         `json:"periodStart"`
	PeriodEnd   string         `json:"periodEnd"`
	Summary     string         `json:"summary"`
	Content     *ReportContent `json:"content,omitempty"` // 详情才带
	ContentMd   string         `json:"contentMd,omitempty"` // 详情才带
	CreatedBy   int64          `json:"createdBy,string"`
	CreatorName string         `json:"creatorName"` // 生成人昵称(空=定时任务)
	CreatedAt   time.Time      `json:"createdAt"`
}
