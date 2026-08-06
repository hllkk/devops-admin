package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// ProviderSearch 供应商分页查询(对齐前端 GET /gateway/provider/list，query 传输)。
// name 模糊匹配；providerType/billingType 精确；isActive 精确(指针区分未传与 false)。
type ProviderSearch struct {
	commonReq.PageInfo
	Name         string `json:"name" form:"name"`                     // 供应商名称(模糊)
	ProviderType string `json:"providerType" form:"providerType"`     // 供应商类型(精确)
	BillingType  string `json:"billingType" form:"billingType"`        // 计费类型(精确)
	IsActive     *bool  `json:"isActive" form:"isActive"`             // 是否启用(精确,nil=不限)
}

// ProviderOperateParams 供应商新增/修改(对齐前端 POST/PUT /gateway/provider)。
// create 时 providerId 为空(雪花主键由回调填充)；update 时必填 providerId。
// monthlyBudget/isActive 用指针以显式区分零值(nil=不改/用默认)。
type ProviderOperateParams struct {
	ProviderId    int64    `json:"providerId,string" form:"providerId"`     // 供应商ID(新增为空)
	Name          string   `json:"name" form:"name"`                       // 供应商名称
	ProviderType  string   `json:"providerType" form:"providerType"`       // 供应商类型
	BillingType   string   `json:"billingType" form:"billingType"`         // 计费类型(空=默认 token)
	MonthlyBudget *float64 `json:"monthlyBudget" form:"monthlyBudget"`     // 月预算(USD)
	IsActive      *bool    `json:"isActive" form:"isActive"`              // 是否启用
	Description   string   `json:"description" form:"description"`        // 描述
}
