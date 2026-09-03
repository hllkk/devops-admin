package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// ReportGenerateParams 手动生成效能报告(body 传输)。
// weekly/monthly 忽略时间参数按上一完整周期取数；custom 必须带起止业务日。
type ReportGenerateParams struct {
	ReportType string `json:"reportType" binding:"required,oneof=weekly monthly custom"` // 报告类型
	StartDate  string `json:"startDate"`                                              // 开始业务日(custom 用)
	EndDate    string `json:"endDate"`                                                // 结束业务日(custom 用)
}

// ReportSearch 报告列表查询(query 传输)。
type ReportSearch struct {
	commonReq.PageInfo
	ReportType string `json:"reportType" form:"reportType"` // 类型筛选(空=全部)
}
