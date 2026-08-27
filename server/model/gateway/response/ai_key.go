package response

import (
	"time"

	"github.com/hllkk/devops-admin/server/model/gateway"
)

// AiKeyView 密钥出网视图(管理员视角)：不返回 KeyValue 明文，只回 KeyPrefix。
type AiKeyView struct {
	gateway.AiKey
	OwnerName     string         `json:"ownerName"`     // 归属名(user→昵称/dept→部门名,联表填充)
	OwnerUsername string         `json:"ownerUsername"` // 归属用户登录名(user→sys_users.user_name,dept留空)
	ScenarioName  string         `json:"scenarioName"`  // 场景名(逻辑关联填充,场景Key才有)
	Models        []string       `json:"models"`        // 授权模型(展开)
	ModelBudgets  map[string]any `json:"modelBudgets"`  // 按模型预算(展开)
	ModelLimits   map[string]any `json:"modelLimits"`   // per-model限流(展开)
}

// AiKeyRevealView 密钥完整明文出网视图(仅 value/:id 按需返回)：管理员把 Key 复制给用户用，
// 与列表/详情的"只回 KeyPrefix"默认安全边界区分开。
type AiKeyRevealView struct {
	KeyValue string `json:"keyValue"` // 密钥明文(解密 key_value)
}

// MyIdentityView 我的 AI 身份(home 切真实接口的契约，对齐前端 home mock 的 IdentityKey)：
// 主 Key 明文(仅 owner 本人可查) + 我的场景 Key 列表 + 可用模型列表。
// 管理员创建制：主 Key 由管理员后台创建，未创建时 opened=false 其余字段为空。
type MyIdentityView struct {
	Opened          bool                 `json:"opened"`          // 是否已开通(存在主 Key)
	KeyValue        string               `json:"keyValue"`        // 主Key明文(仅identity/my返回)
	IsActive        bool                 `json:"isActive"`        // 主Key是否启用
	ExpiresAt       *time.Time           `json:"expiresAt"`       // 过期时间(nil=永不过期)
	BudgetLimit     *float64             `json:"budgetLimit"`     // 预算上限
	BudgetHardLimit bool                 `json:"budgetHardLimit"` // 硬限
	BudgetDuration  string               `json:"budgetDuration"`  // 预算周期
	Models          []string             `json:"models"`          // 已授权模型
	ModelBudgets    map[string]any       `json:"modelBudgets"`    // 按模型预算
	RateLimitMode   string               `json:"rateLimitMode"`   // 限流模式
	TpmLimit        *int                 `json:"tpmLimit"`        // 全局TPM
	RpmLimit        *int                 `json:"rpmLimit"`        // 全局RPM
	SceneKeys       []AiKeyView          `json:"sceneKeys"`       // 我的场景Key列表
	AvailableModels []AvailableModelView `json:"availableModels"` // 可用模型(供申请新Key选)
}

// AvailableModelView 可授权模型(精简版，供 Key 授权选择)。
type AvailableModelView struct {
	ModelId                int64  `json:"modelId,string"`         // 模型ID
	ModelKey               string `json:"modelKey"`               // 路由名(openai组)
	ModelKeyAnthropic      string `json:"modelKeyAnthropic"`      // anthropic变体路由名(无则空)
	Name                   string `json:"name"`                   // 展示名
	Category               string `json:"category"`               // 类别
	RequiresApproval       bool   `json:"requiresApproval"`       // 订阅需审批
	HasAnthropicDeployment bool   `json:"hasAnthropicDeployment"` // 有anthropic活跃部署
}

// BatchCreateMainKeysResult 批量开通个人主 Key 结果(部分成功语义：走 OkWithDetailed 成功
// 响应+data 标记，前端按 failed 列表渲染，避免 axios 自动弹错误码造成双提示)。
type BatchCreateMainKeysResult struct {
	Total   int                          `json:"total"`   // 目标用户数
	Created int                          `json:"created"` // 新开通
	Skipped int                          `json:"skipped"` // 已有主 Key 跳过
	Failed  []BatchCreateMainKeysFailure `json:"failed"`  // 失败明细(空数组=全部成功)
}

// BatchCreateMainKeysFailure 单用户开通失败明细。
type BatchCreateMainKeysFailure struct {
	UserId int64  `json:"userId,string"` // 用户ID
	Name   string `json:"name"`          // 用户昵称
	Reason string `json:"reason"`        // 失败原因
}
