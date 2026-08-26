package gateway

import (
	"github.com/hllkk/devops-admin/server/global"
	"gorm.io/datatypes"
)

// Provider AI 供应商（管理元数据：名称/类型/接入格式，不直接同步 LiteLLM）。
// 其下 Credential 才携带密钥并同步 LiteLLM /credentials；计费与预算口径在部署(Deployment)级，
// 供应商级不再持有计费/预算字段(对齐 AIHelms 设计，成本计算见 usage_sync)。
// 字段对齐 aiDoc/modules/business-modules.md「四层数据模型」的 Provider。
type Provider struct {
	global.OPS_AUDIT_MODEL
	ProviderId    int64    `json:"providerId,string" gorm:"primarykey;comment:供应商ID"`        // 供应商ID(雪花)
	Name          string   `json:"name" gorm:"index;comment:供应商名称"`                       // 供应商名称(如 OpenAI/Anthropic)
	ProviderType  string   `json:"providerType" gorm:"size:50;comment:供应商类型(openai/anthropic/vllm...)"` // 供应商类型(对应 LiteLLM custom_llm_provider)
	IsActive      bool     `json:"isActive" gorm:"default:true;comment:是否启用"`               // 是否启用
	Description   string   `json:"description" gorm:"type:text;comment:描述"`                  // 描述
	SupportedFormats datatypes.JSON `json:"supportedFormats" gorm:"type:jsonb;comment:支持的接入格式(openai/anthropic/lmstudio/ollama)" swaggertype:"object"` // 支持的接入格式(凭证 format 从中选)
	CredentialCount  int64           `json:"credentialCount" gorm:"-"`                                // 凭证数(列表展示,service 层填充,不入库)
}

// 计费类型(部署级口径，usage_sync 按 Deployment.BillingType 计算成本)
const (
	BillingTypeToken        = "token"         // 按 token 计费(默认)
	BillingTypePerCall      = "per_call"      // 按次计费
	BillingTypeMonthlyQuota = "monthly_quota" // 月配额
)

func (Provider) TableName() string {
	return "gateway_provider"
}
