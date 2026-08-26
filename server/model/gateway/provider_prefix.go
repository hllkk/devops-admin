package gateway

import "github.com/hllkk/devops-admin/server/global"

// ProviderPrefix 供应商路由前缀差异表（驱动 LiteLLM 部署 model 前缀化，不在代码里 switch）。
// 参照 AIHelms provider_prefix_map：按 (provider_type, format, category) 三键查 prefix 与 needs_v1，
// 本 slice 仅落表+种子，消费方是下一 slice 的 ModelDeployment 同步（_build_litellm_params_for_sync）。
type ProviderPrefix struct {
	global.OPS_MODEL
	ProviderType string `json:"providerType" gorm:"size:50;uniqueIndex:idx_gateway_provider_prefix;comment:供应商类型"` // 供应商类型(openai/anthropic/vllm...)
	Format       string `json:"format" gorm:"size:20;uniqueIndex:idx_gateway_provider_prefix;comment:协议格式"`       // 协议格式(openai/anthropic/lmstudio/ollama)
	Category     string `json:"category" gorm:"size:20;uniqueIndex:idx_gateway_provider_prefix;comment:模型类别"`      // 模型类别(chat/embedding/rerank...)
	Prefix       string `json:"prefix" gorm:"size:100;comment:LiteLLM路由前缀"`                      // LiteLLM 路由前缀(如 hosted_vllm/gemini)
	NeedsV1      bool   `json:"needsV1" gorm:"comment:api_base是否自动补/v1"`                    // api_base 是否自动补 /v1
}

func (ProviderPrefix) TableName() string {
	return "gateway_provider_prefix"
}
