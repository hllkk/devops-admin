package response

import (
	"github.com/hllkk/devops-admin/server/model/gateway"
)

// ModelView 模型列表行：本体+部署计数。
type ModelView struct {
	gateway.Model
	Capabilities          []string `json:"capabilities"`          // 能力标签(展开)
	DeploymentCount       int64    `json:"deploymentCount"`       // 部署总数(未删)
	ActiveDeploymentCount int64    `json:"activeDeploymentCount"` // 活跃部署数(is_active=true)
}

// ModelDetailView 模型详情：含部署列表(带路由名/掩码参数)。
type ModelDetailView struct {
	ModelView
	Deployments []DeploymentView `json:"deployments"` // 部署列表
}

// ActiveModelView 对外激活模型视图(home/AI 身份/模型市场用)：
// 仅 active+published 模型；hasAnthropicDeployment 时附 anthropic 变体路由名。
type ActiveModelView struct {
	ModelId                int64    `json:"modelId,string"`         // 模型ID
	ModelKey               string   `json:"modelKey"`               // 路由名(openai 格式组)
	ModelKeyAnthropic      string   `json:"modelKeyAnthropic"`      // anthropic 变体路由名(无 anthropic 部署为空)
	Name                   string   `json:"name"`                   // 展示名
	Category               string   `json:"category"`               // 类别
	Description            string   `json:"description"`            // 描述
	LogoProviderType       string   `json:"logoProviderType"`       // LOGO供应商类型
	Capabilities           []string `json:"capabilities"`           // 能力标签
	RequiresApproval       bool     `json:"requiresApproval"`       // 订阅需审批
	HasAnthropicDeployment bool     `json:"hasAnthropicDeployment"` // 是否存在 anthropic 格式活跃部署
}

// DeploymentView 部署出网视图：关联上下文(路由名/凭证名)+掩码后的路由参数。
type DeploymentView struct {
	gateway.ModelDeployment
	ModelKey       string         `json:"modelKey"`          // 关联模型路由名
	CredentialName string         `json:"credentialName"`    // 关联凭证名(无关联为空)
	Format         string         `json:"format"`            // 凭证协议格式(openai/anthropic)
	ProviderId     int64          `json:"providerId,string"` // 关联供应商ID(编辑回填供应商下拉用)
	ProviderType   string         `json:"providerType"`      // 关联供应商类型
	RouteName      string         `json:"routeName"`         // 当前路由名(三态命名,routable 版)
	LitellmParams  map[string]any `json:"litellmParams"`     // 路由参数(敏感值已掩码)
}

// ModelPublishView 模型发布设置视图。
type ModelPublishView struct {
	ModelId          int64   `json:"modelId,string"`   // 模型ID
	IsPublished      bool    `json:"isPublished"`      // 是否发布
	VisibilityType   string  `json:"visibilityType"`   // 可见范围(all/selected/user)
	RequiresApproval bool    `json:"requiresApproval"` // 订阅需审批
	DepartmentIds    []int64 `json:"departmentIds"`    // 可见部门(selected 模式)
	UserIds          []int64 `json:"userIds"`          // 可见用户(user 模式)
}

// DeploymentTestResult 部署连通性测试结果(管理员视角,经 LiteLLM 数据面)。
type DeploymentTestResult struct {
	Success         bool   `json:"success"`         // 是否连通
	LatencyMs       int64  `json:"latencyMs"`       // 耗时(毫秒)
	ErrorCategory   string `json:"errorCategory"`   // 错误类别(auth_error/model_not_found/rate_limited/bad_request/provider_error/network_error/unknown)
	Message         string `json:"message"`         // 用户可读错误信息
	TechnicalDetail string `json:"technicalDetail"` // 技术详情(已脱敏+截断)
}
