package gateway

import (
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"gorm.io/datatypes"
)

// EfficiencyReport 效能报告(P3)：模板化数据报告——复用采用度/成本分析取数组装
// 结构化 JSON + Markdown 文本，不调 LLM 生成文字(AIHelms 的 LLM 报告为半成品,
// 永远停在"生成中";模板化保证真闭环与零成本)。周/月报由定时任务生成,
// custom 由管理员手动生成;同类型同起始周报/月报幂等(已存在直接返回)。
type EfficiencyReport struct {
	global.OPS_AUDIT_MODEL
	ReportId    int64          `json:"reportId,string" gorm:"primarykey;comment:报告ID(雪花)"`        // 报告ID(雪花)
	ReportType  string         `json:"reportType" gorm:"size:20;index;comment:类型(weekly/monthly/custom)"` // 报告类型
	PeriodStart string         `json:"periodStart" gorm:"size:10;comment:统计期开始(业务日 YYYY-MM-DD)"`   // 统计期开始
	PeriodEnd   string         `json:"periodEnd" gorm:"size:10;comment:统计期结束(业务日)"`          // 统计期结束
	Summary     string         `json:"summary" gorm:"size:500;comment:一句话摘要"`                  // 摘要
	Content     datatypes.JSON `json:"content" gorm:"type:jsonb;comment:结构化内容(KPI/部门/模型/Top用户)" swaggertype:"object"` // 结构化内容
	ContentMd   string         `json:"contentMd" gorm:"type:text;comment:Markdown 文本(复制/通知用)"`    // Markdown 文本
	CreatedBy   int64          `json:"createdBy,string" gorm:"comment:生成人(0=定时任务)"`             // 生成人
	CreatedAt   time.Time      `json:"createdAt" gorm:"comment:生成时间"`                          // 生成时间
}

func (EfficiencyReport) TableName() string {
	return "gateway_efficiency_report"
}

// 效能报告类型
const (
	ReportTypeWeekly  = "weekly"  // 周报(上周一~上周日)
	ReportTypeMonthly = "monthly" // 月报(上月1号~月末)
	ReportTypeCustom  = "custom"  // 自定义区间(手动)
)
