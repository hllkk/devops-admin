package response

import (
	"github.com/hllkk/devops-admin/server/model/gateway"
)

// RouterSettingsView 全局路由策略出网视图(剥离审计基座,对齐前端 Api.Gateway.RouterSettings)。
type RouterSettingsView struct {
	RoutingStrategy string                   `json:"routingStrategy"` // 路由策略
	Fallbacks       []gateway.FallbackItem   `json:"fallbacks"`        // 降级链
	AllowedFails    int                      `json:"allowedFails"`    // 允许连续失败次数
	CooldownTime    int                      `json:"cooldownTime"`    // 冷却时间(秒)
	NumRetries      int                      `json:"numRetries"`      // 全局重试次数
	Timeout         int                      `json:"timeout"`         // 全局超时(秒)
	Config          map[string]any           `json:"config"`          // 扩展配置
}
