package gateway

import (
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"gorm.io/datatypes"
)

// AiKey AI 密钥（LiteLLM 虚拟 Key 的管理面投影）。
// key_value 加密存储(偏离 AIHelms 不存的设计：devops-admin home 需明文展示主 Key、
// LiteLLM /key/generate 只返回一次、已有 AES-GCM 基座)，仅 owner 本人经 identity/my 可查明文。
// owner_id 纯逻辑关联(sys_users.user_id / sys_departments.dept_id)，不建外键。
// models 存 modelKey 列表(含 anthropic 变体扩展)；推送 LiteLLM 时经 expandModelsWithAnthropic 补变体。
type AiKey struct {
	global.OPS_AUDIT_MODEL
	AiKeyId         int64          `json:"aiKeyId,string" gorm:"primarykey;comment:密钥ID(雪花)"`         // 密钥ID(雪花)
	Name            string         `json:"name" gorm:"size:128;comment:密钥名称"`                  // 密钥名称(主Key固定"主Key")
	Description     string         `json:"description" gorm:"type:text;comment:描述"`               // 描述
	KeyType         string         `json:"keyType" gorm:"size:20;index;comment:密钥类型(personal_main/personal_scene/dept_main/dept_scene)"` // 密钥类型
	OwnerType       string         `json:"ownerType" gorm:"size:20;comment:归属类型(user/dept)"`       // 归属类型
	OwnerId         int64          `json:"ownerId,string" gorm:"index;comment:归属ID(用户ID/部门ID)"`    // 归属ID(纯逻辑关联)
	KeyValue        string         `json:"-" gorm:"type:text;comment:Key明文(AES-256-GCM密文,仅owner经identity/my可查)"` // Key明文密文(不出网)
	KeyPrefix       string         `json:"keyPrefix" gorm:"size:20;comment:Key前缀(列表展示用)"`        // Key前缀(明文前8位+****)
	LitellmKeyId    string         `json:"litellmKeyId" gorm:"size:100;index;comment:LiteLLM密钥ID"`    // LiteLLM key_id
	LitellmKeyAlias string         `json:"litellmKeyAlias" gorm:"size:200;comment:LiteLLM密钥别名"`      // LiteLLM key_alias({ownerType}:{ownerId}/{name})
	ScenarioId      int64          `json:"scenarioId,string" gorm:"index;comment:场景ID(逻辑关联gateway_key_scenario,0=无;仅场景Key有值)"` // 场景ID(逻辑关联)
	Models          datatypes.JSON `json:"models" gorm:"type:jsonb;comment:授权模型列表(modelKey,含anthropic变体)" swaggertype:"object"` // 授权模型(modelKey 列表)
	ModelBudgets   datatypes.JSON `json:"modelBudgets" gorm:"type:jsonb;comment:按模型预算({modelKey:金额})" swaggertype:"object"` // 按模型预算
	Mcps            datatypes.JSON `json:"mcps" gorm:"type:jsonb;comment:授权MCP(P2预留)" swaggertype:"object"` // 授权MCP(P2)
	Skills          datatypes.JSON `json:"skills" gorm:"type:jsonb;comment:授权Skill(P2预留)" swaggertype:"object"` // 授权Skill(P2)
	BudgetLimit     *float64       `json:"budgetLimit" gorm:"type:numeric(12,4);comment:预算上限(nil=不限)"` // 预算上限
	BudgetUsed      float64        `json:"budgetUsed" gorm:"type:numeric(12,4);default:0;comment:已用预算"` // 已用(slice5聚合回填)
	BudgetHardLimit bool           `json:"budgetHardLimit" gorm:"default:false;comment:硬限(超支停用,max_budget=0下发LiteLLM)"` // 硬限
	BudgetDuration  string         `json:"budgetDuration" gorm:"size:10;default:30d;comment:预算周期(1d/7d/30d)"` // 预算周期
	RateLimitMode   string         `json:"rateLimitMode" gorm:"size:20;default:none;comment:限流模式(none/total/per_model)"` // 限流模式
	TpmLimit        *int           `json:"tpmLimit" gorm:"comment:全局TPM限流(total模式)"`        // 全局TPM
	RpmLimit        *int           `json:"rpmLimit" gorm:"comment:全局RPM限流(total模式)"`        // 全局RPM
	ModelLimits     datatypes.JSON `json:"modelLimits" gorm:"type:jsonb;comment:按模型限流({modelKey:{tpm,rpm}})" swaggertype:"object"` // per-model 限流
	IsActive        bool           `json:"isActive" gorm:"default:true;comment:是否启用"`             // 是否启用(停用=max_budget=0)
	ExpiresAt       *time.Time     `json:"expiresAt" gorm:"comment:过期时间(nil=永不过期,下发LiteLLM expires_at)"` // 过期时间(LiteLLM 原生拒绝过期请求)
	LastUsedAt      *time.Time     `json:"lastUsedAt" gorm:"comment:最近使用时间(用量回流回填,取最近调用时间)"` // 最近使用(僵尸Key治理)
}

// 密钥类型
const (
	KeyTypePersonalMain  = "personal_main"  // 个人主Key(管理员创建,默认含公开模型)
	KeyTypePersonalScene = "personal_scene" // 个人场景Key(手动申请)
	KeyTypeDeptMain      = "dept_main"      // 部门主Key
	KeyTypeDeptScene     = "dept_scene"     // 部门场景Key
)

// MainKeyType 判定是否主 Key 类型(personal_main/dept_main)。
func MainKeyType(kt string) bool {
	return kt == KeyTypePersonalMain || kt == KeyTypeDeptMain
}

// 归属类型
const (
	OwnerTypeUser = "user" // 个人(user_id)
	OwnerTypeDept = "dept" // 部门(dept_id)
)

// 限流模式
const (
	RateLimitModeNone      = "none"      // 无限流
	RateLimitModeTotal     = "total"     // 全局限流(tpm/rpm)
	RateLimitModePerModel  = "per_model" // 按模型限流(metadata.model_tpm/rpm_limit)
)

// 预算周期
const (
	BudgetDuration1d  = "1d"
	BudgetDuration7d  = "7d"
	BudgetDuration30d = "30d"
)

func (AiKey) TableName() string {
	return "gateway_ai_key"
}
