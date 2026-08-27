package response

import (
	"time"

	"github.com/hllkk/devops-admin/server/model/gateway"
)

// ProviderBalanceDetail 供应商余量明细（汇总 + 坐席/共享包快照行）。
type ProviderBalanceDetail struct {
	Summary ProviderBalanceSummary     `json:"summary"` // 汇总
	Items   []gateway.ProviderBalance  `json:"items"`   // 快照明细
}

// ProviderBalanceSummary 套餐余量汇总（供应商页面板头 + 看板汇总卡共用）。
// 口径：来自厂商侧快照（旁路只读），与网关标价成本口径不同，不参与预算卡口。
type ProviderBalanceSummary struct {
	ProviderId   int64      `json:"providerId,string"` // 供应商ID
	ProviderName string     `json:"providerName"`      // 供应商名称
	PlanLabel    string     `json:"planLabel"`         // 套餐标签(如"百炼 Token Plan")
	TotalValue   float64    `json:"totalValue"`        // 周期总额度(Credits,坐席+共享包)
	UsedValue    float64    `json:"usedValue"`         // 已用额度(Credits)
	SurplusValue float64    `json:"surplusValue"`      // 剩余额度(Credits)
	SeatCount    int        `json:"seatCount"`         // 坐席数
	PackageCount int        `json:"packageCount"`      // 共享包数
	SyncedAt     *time.Time `json:"syncedAt"`          // 最近同步时间(UTC,空=从未同步)
}
