package gateway

import (
	"github.com/hllkk/devops-admin/server/global"
	"gorm.io/datatypes"
)

// Model 模型主体（管理元数据+发布控制，部署见 ModelDeployment）。
// model_key 是平台模型 ID = LiteLLM model group 名：同 model_key 的多个部署在
// LiteLLM 侧共享同名路由组，天然构成负载均衡池（LB 策略委托 LiteLLM Router）。
// is_published/visibility_type/requires_approval 只影响用户端可见性与后续 Key 授权，
// 不影响部署同步（同步只看 deployment.is_active + credential.is_active）。
type Model struct {
	global.OPS_AUDIT_MODEL
	ModelId          int64          `json:"modelId,string" gorm:"primarykey;comment:模型ID(雪花)"`             // 模型ID(雪花)
	ModelKey         string         `json:"modelKey" gorm:"size:128;index;comment:路由名(LiteLLM model_group,服务层查重,软删不建唯一索引)"` // 平台模型ID(用户请求名/路由组名)
	Name             string         `json:"name" gorm:"size:128;comment:展示名"`                       // 展示名
	Category         string         `json:"category" gorm:"size:50;default:chat;comment:类别(chat/embedding/rerank)"` // 类别(决定路由前缀/测试端点)
	Capabilities     datatypes.JSON `json:"capabilities" gorm:"type:jsonb;comment:能力标签数组" swaggertype:"object"` // 能力标签(JSON数组)
	Description      string         `json:"description" gorm:"type:text;comment:描述"`                  // 描述
	LogoProviderType string         `json:"logoProviderType" gorm:"size:50;comment:LOGO取用的供应商类型"`     // 前端图标归属供应商类型
	IsActive         bool           `json:"isActive" gorm:"default:true;comment:是否启用"`               // 是否启用(软删联动)
	IsPublished      bool           `json:"isPublished" gorm:"default:false;comment:是否发布到用户端"`        // 是否发布
	VisibilityType   string         `json:"visibilityType" gorm:"size:20;default:all;comment:可见范围(all/selected)"` // 可见范围
	RequiresApproval bool           `json:"requiresApproval" gorm:"default:false;comment:用户订阅是否需审批"`   // 订阅需审批
}

// 模型类别
const (
	ModelCategoryChat      = "chat"      // 对话(默认)
	ModelCategoryEmbedding = "embedding" // 向量
	ModelCategoryRerank    = "rerank"    // 重排
)

// 模型可见范围
const (
	VisibilityTypeAll      = "all"      // 全员可见
	VisibilityTypeSelected = "selected" // 指定部门可见(配 gateway_model_visibility)
)

// 路由池命名后缀(禁用=改名出池+active 双写，litellm_model_id 永不变保归因锚点)
const (
	ModelAnthropicSuffix = "(Anthropic)"  // anthropic 格式凭证的部署独立分组(协议隔离，不能与 openai 混组 LB)
	ModelDisabledSuffix  = "__disabled__" // 不可路由后缀(部署或凭证停用时摘出 LB 组)
)

func (Model) TableName() string {
	return "gateway_model"
}

// ModelDeployment 模型部署（一个 Model 多 Deployment → LiteLLM 同名路由组 LB 池）。
// litellm_params 存处理后的完整路由参数(人民币口径定价四键；绑定凭证时含 litellm_credential_name
// 引用、无 inline api_key)；litellm_model_id 是 LiteLLM 侧部署 ID，成本/日志归因锚点，永不重置。
// credential_id 为 0 表示内联参数部署(params 可含明文 api_key，出网掩码；推荐绑定凭证)。
type ModelDeployment struct {
	global.OPS_AUDIT_MODEL
	DeploymentId     int64          `json:"deploymentId,string" gorm:"primarykey;comment:部署ID(雪花)"`    // 部署ID(雪花)
	ModelId          int64          `json:"modelId,string" gorm:"index;comment:关联模型ID(纯逻辑关联)"`      // 关联模型
	CredentialId     int64          `json:"credentialId,string" gorm:"index;comment:关联凭证ID(0=内联参数)"`   // 关联凭证(0=内联)
	LitellmModelId   string         `json:"litellmModelId" gorm:"size:100;index;comment:LiteLLM部署ID(归因锚点,永不重置)"` // LiteLLM 部署 UUID
	LitellmParams    datatypes.JSON `json:"litellmParams" gorm:"type:jsonb;comment:路由参数(人民币口径,含凭证引用/前缀化model)" swaggertype:"object"` // 处理后路由参数
	ModelInfo        datatypes.JSON `json:"modelInfo" gorm:"type:jsonb;comment:模型元数据(内外定价镜像)" swaggertype:"object"` // 元数据(定价镜像等)
	DeployName       string         `json:"deployName" gorm:"size:128;comment:部署名"`                 // 部署别名
	BillingType      string         `json:"billingType" gorm:"size:20;default:token;comment:计费类型 token/per_call/monthly_quota"` // 计费类型
	CostPerCall      *float64       `json:"costPerCall" gorm:"type:numeric(8,4);comment:单次成本(¥,nil=不限)"` // 按次成本
	MonthlyCallQuota *int           `json:"monthlyCallQuota" gorm:"comment:月调用配额(nil=不限)"`        // 月配额
	MonthlyCallUsed  int            `json:"monthlyCallUsed" gorm:"default:0;comment:月已用次数"`          // 月已用
	IsActive         bool           `json:"isActive" gorm:"default:true;comment:是否启用"`              // 是否启用
}

func (ModelDeployment) TableName() string {
	return "gateway_model_deployment"
}

// ModelVisibility 模型部门可见性(发布投影表，visibility_type=selected 时使用)。
// 非业务实体：重建时一律物理删除(Unscoped)，软删行会占住 (model_id, department_id)
// 唯一索引挡住同组合重新发布。部门用户级展开与 Key 自动授权依赖 AiKey，P1 slice4 落地。
type ModelVisibility struct {
	global.OPS_MODEL
	ModelId      int64 `json:"modelId,string" gorm:"uniqueIndex:idx_gateway_model_visibility;comment:关联模型ID"`        // 关联模型
	DepartmentId int64 `json:"departmentId,string" gorm:"uniqueIndex:idx_gateway_model_visibility;comment:关联部门ID(sys_departments.dept_id)"` // 关联部门
}

func (ModelVisibility) TableName() string {
	return "gateway_model_visibility"
}
