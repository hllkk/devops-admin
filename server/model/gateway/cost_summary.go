package gateway

import (
	"time"

	"github.com/hllkk/devops-admin/server/global"
)

// CostSummaryDaily 用量日汇总缓存（滚动重建：近60天，每5分钟 DELETE+INSERT 自愈）。
// 聚合表不带独立状态机（游标只在原始同步层 slice5a gateway_sync_state）；
// summary_date 按业务时区(Asia/Shanghai)切日桶，规避 UTC date_trunc 与上海0点错位8h。
// 部门维度不写进聚合表（人员调岗不污染历史成本），看板按部门过滤时读时归因 EXISTS sys_user_departments。
type CostSummaryDaily struct {
	global.OPS_BASE
	SummaryId           uint      `json:"summaryId,string" gorm:"primarykey;autoIncrement;comment:汇总ID(自增,派生缓存表不走雪花)"` // 自增(聚合 DELETE+INSERT 重建,Raw INSERT SELECT 不走雪花回调)
	SummaryDate         time.Time `json:"summaryDate" gorm:"index:idx_gateway_cost_summary;type:date;comment:业务日(Asia/Shanghai切日桶)"` // 业务日
	UserId              int64   `json:"userId,string" gorm:"index:idx_gateway_cost_summary;comment:归因用户ID(0=未归因)"` // 归因用户
	AiKeyId             int64   `json:"aiKeyId,string" gorm:"index:idx_gateway_cost_summary;comment:归因密钥ID(0=未归因)"` // 归因AiKey
	Model               string  `json:"model" gorm:"index:idx_gateway_cost_summary;size:128;comment:模型名"` // 模型名
	Provider            string  `json:"provider" gorm:"size:50;comment:供应商类型"`           // 供应商
	RequestCount        int     `json:"requestCount" gorm:"default:0;comment:请求数"`        // 请求数
	PromptTokens        int     `json:"promptTokens" gorm:"default:0;comment:输入token"`     // prompt_tokens
	CompletionTokens    int     `json:"completionTokens" gorm:"default:0;comment:输出token"`   // completion_tokens
	TotalTokens         int     `json:"totalTokens" gorm:"default:0;comment:总token"`        // total_tokens
	CacheReadTokens    int     `json:"cacheReadTokens" gorm:"default:0;comment:缓存读token"`   // cache_read
	CacheCreationTokens int    `json:"cacheCreationTokens" gorm:"default:0;comment:缓存创建token"` // cache_creation
	ExternalCost        float64 `json:"externalCost" gorm:"type:numeric(12,6);default:0;comment:外部成本(¥)"` // external_cost
	InternalCost        float64 `json:"internalCost" gorm:"type:numeric(12,6);default:0;comment:内部成本(¥,P1同external)"` // internal_cost
}

func (CostSummaryDaily) TableName() string {
	return "gateway_cost_summary_daily"
}
