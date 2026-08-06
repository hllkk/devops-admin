package gateway

import "github.com/hllkk/devops-admin/server/global"

// Provider AI 供应商（管理元数据：名称/类型/计费/月预算，不直接同步 LiteLLM）。
// 其下 Credential 才携带密钥并同步 LiteLLM /credentials；本表只做供应商维度的计费与预算跟踪。
// 字段对齐 aiDoc/modules/business-modules.md「四层数据模型」的 Provider。
type Provider struct {
	global.OPS_AUDIT_MODEL
	ProviderId    int64    `json:"providerId,string" gorm:"primarykey;comment:供应商ID"`        // 供应商ID(雪花)
	Name          string   `json:"name" gorm:"index;comment:供应商名称"`                       // 供应商名称(如 OpenAI/Anthropic)
	ProviderType  string   `json:"providerType" gorm:"size:50;comment:供应商类型(openai/anthropic/vllm...)"` // 供应商类型(对应 LiteLLM custom_llm_provider)
	BillingType   string   `json:"billingType" gorm:"size:20;default:token;comment:计费类型 token/per_call/monthly_quota"` // 计费类型
	MonthlyBudget *float64 `json:"monthlyBudget" gorm:"type:numeric(12,4);comment:月预算(USD)"`  // 月预算(USD,nil=不限)
	MonthlyUsed   float64  `json:"monthlyUsed" gorm:"type:numeric(12,4);default:0;comment:月已用(USD)"` // 月已用(USD,用量统计定时回填)
	IsActive      bool     `json:"isActive" gorm:"default:true;comment:是否启用"`               // 是否启用
	Description   string   `json:"description" gorm:"type:text;comment:描述"`                  // 描述
}

// 供应商计费类型
const (
	BillingTypeToken        = "token"         // 按 token 计费(默认)
	BillingTypePerCall      = "per_call"      // 按次计费
	BillingTypeMonthlyQuota = "monthly_quota" // 月配额
)

func (Provider) TableName() string {
	return "gateway_provider"
}
