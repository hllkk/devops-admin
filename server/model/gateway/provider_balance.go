package gateway

import (
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"gorm.io/datatypes"
)

// ProviderBalance 套餐余量快照（旁路只读，不进成本链路）。
// 语义：每 (provider_id, item_type, item_key) 一行"当前余量现状"，同步时整批 DELETE+INSERT 重建，
// 不堆历史行（同 CostSummaryDaily 的派生缓存模式）。
// 硬边界（aiDoc/modules/ai-gateway-billing-integration.md）：不进 calcCosts、不触发预算停用、不并入 cost_summary_daily。
type ProviderBalance struct {
	global.OPS_BASE
	BalanceId    int64          `json:"balanceId,string" gorm:"primarykey;comment:余量ID(自增,快照表不走雪花)"` // 余量ID
	ProviderId   int64          `json:"providerId,string" gorm:"index:idx_gateway_provider_balance;comment:关联供应商ID(雪花)"` // 供应商ID
	PlanType     string         `json:"planType" gorm:"size:32;default:token_plan;comment:套餐类型(token_plan/subscription/credit)"` // 套餐类型
	ItemType     string         `json:"itemType" gorm:"index:idx_gateway_provider_balance;size:32;comment:条目类型(seat=坐席/shared_package=共享包)"` // 条目类型
	ItemKey      string         `json:"itemKey" gorm:"size:128;comment:条目键(坐席SeatId/共享包InstanceCode)"` // 条目键
	ItemName     string         `json:"itemName" gorm:"size:128;comment:条目名称(坐席成员名/共享包说明)"` // 条目名称
	SpecType     string         `json:"specType" gorm:"size:32;comment:坐席档位(standard/pro/max)"` // 坐席档位
	Status       string         `json:"status" gorm:"size:32;comment:条目状态(NORMAL/LIMIT/RELEASE/STOP/...)"` // 条目状态
	EquityType   string         `json:"equityType" gorm:"size:32;comment:权益类型(CREDITS)"` // 权益类型
	CycleStart   *time.Time     `json:"cycleStart" gorm:"comment:当前计费周期开始"` // 周期开始
	CycleEnd     *time.Time     `json:"cycleEnd" gorm:"comment:当前计费周期结束"` // 周期结束
	TotalValue   float64        `json:"totalValue" gorm:"type:numeric(16,4);default:0;comment:周期总额度(Credits)"` // 总额度
	SurplusValue float64        `json:"surplusValue" gorm:"type:numeric(16,4);default:0;comment:周期剩余额度(Credits)"` // 剩余额度
	UsedValue    float64        `json:"usedValue" gorm:"type:numeric(16,4);default:0;comment:周期已用额度(Credits,总-剩余)"` // 已用额度
	SyncedAt     time.Time      `json:"syncedAt" gorm:"comment:同步时间(UTC)"` // 同步时间
	Raw          datatypes.JSON `json:"raw" gorm:"type:jsonb;comment:厂商原始返回(排障用)" swaggertype:"object"` // 原始返回
}

// 条目类型
const (
	BalanceItemTypeSeat    = "seat"           // 坐席
	BalanceItemTypePackage = "shared_package" // 共享用量包
)

// TableName 表名。
func (ProviderBalance) TableName() string { return "gateway_provider_balance" }

// BalanceSyncConfig 供应商余量采集配置（明文结构，落库 AES-256-GCM 密文）。
// 百炼 Token Plan 走阿里云 OpenAPI AK/SK（RAM 凭证，非模型 Key），region 固定 cn-beijing。
type BalanceSyncConfig struct {
	AccessKeyId     string `json:"accessKeyId"`     // 阿里云 AccessKey ID
	AccessKeySecret string `json:"accessKeySecret"` // 阿里云 AccessKey Secret
	Region          string `json:"region"`          // 服务地域(默认 cn-beijing,Token Plan 仅支持北京)
}

// 支持余量采集的供应商类型（采集入口按此白名单校验）。
var BalanceSyncProviderTypes = map[string]string{
	"dashscope": "百炼 Token Plan", // 阿里云百炼
}
