package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// CostAnalysisSearch 成本分析查询(P3，读聚合表 cost_summary_daily，query 传输)。
// 时间为业务日(YYYY-MM-DD，summary_date 按 Asia/Shanghai 切桶)；部门筛选含子树。
type CostAnalysisSearch struct {
	commonReq.PageInfo
	StartDate    string `json:"startDate" form:"startDate"`                // 开始业务日(缺省本月首日)
	EndDate      string `json:"endDate" form:"endDate"`                    // 结束业务日(缺省今天)
	Dimension    string `json:"dimension" form:"dimension"`                // 维度(department/user/model/aiKey/provider/date，默认department)
	Sort         string `json:"sort" form:"sort"`                          // 排序键(internal/external/requests/tokens，默认internal)
	DepartmentId int64  `json:"departmentId,string" form:"departmentId"`   // 部门筛选(含子树，0=不限)
	UserId       int64  `json:"userId,string" form:"userId"`               // 用户筛选(0=不限)
	AiKeyId      int64  `json:"aiKeyId,string" form:"aiKeyId"`             // 密钥筛选(0=不限)
	Model        string `json:"model" form:"model"`                        // 模型名(精确，聚合表粒度即模型名)
	Provider     string `json:"provider" form:"provider"`                  // 供应商(精确)
}
