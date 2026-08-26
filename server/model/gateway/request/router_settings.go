package request

import (
	"github.com/hllkk/devops-admin/server/model/gateway"
)

// RouterSettingsUpdate 全局路由策略更新(整体覆盖,对齐前端 PUT /gateway/router/settings)。
// fallbacks 为前端对象数组 [{model, fallbacks}]；service 同步 LiteLLM 时转 [[model,[fallbacks]]]。
type RouterSettingsUpdate struct {
	RoutingStrategy string                 `json:"routingStrategy"` // 路由策略(空=默认 simple-shuffle)
	Fallbacks       []gateway.FallbackItem `json:"fallbacks"`        // 降级链(可空)
	AllowedFails    int                    `json:"allowedFails"`    // 允许连续失败次数
	CooldownTime    int                    `json:"cooldownTime"`    // 冷却时间(秒)
	NumRetries      int                    `json:"numRetries"`      // 全局重试次数
	Timeout         int                    `json:"timeout"`         // 全局超时(秒)
	Config          map[string]any         `json:"config"`          // 扩展配置(可空)
}
