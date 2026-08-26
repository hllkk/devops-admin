package gateway

import (
	"github.com/hllkk/devops-admin/server/global"
	"gorm.io/datatypes"
)

// 路由策略类型(对齐 LiteLLM routing_strategy 取值)
const (
	RoutingStrategySimpleShuffle = "simple-shuffle"         // 轮询(默认)
	RoutingStrategyLatencyBased  = "latency-based-routing"  // 最低延迟
	RoutingStrategyCostBased     = "cost-based-routing"     // 最低成本
	RoutingStrategyLeastBusy     = "least-busy"             // 最少使用
	RoutingStrategyUsageBased    = "usage-based-routing"    // 按用量均衡
)

// FallbackItem 降级链项(前端对象格式：源模型 → 降级到的模型列表)。
// DB 存此格式；同步 LiteLLM 时投影为 [[model, [fallbacks]]] 蛇形数组。
type FallbackItem struct {
	Model     string   `json:"model"`     // 源模型 ID(model_key)
	Fallbacks []string `json:"fallbacks"` // 降级到的模型 ID 列表
}

// RouterSettings 全局路由策略(单行表,id=1,FirstOrCreate)。
// 平台 DB 是唯一事实源，LiteLLM /router/settings 是投影：更新时同步推送，热更新即时生效。
// 控制负载均衡策略、故障摘除(allowed_fails)、冷却(cooldown_time)、重试(num_retries)、
// 超时(timeout)、降级链(fallbacks)。路由池本身靠 model_key 同名聚合 + 后缀摘除，不在此表。
type RouterSettings struct {
	global.OPS_MODEL
	RoutingStrategy string         `json:"routingStrategy" gorm:"size:40;default:simple-shuffle;comment:路由策略"`
	Fallbacks       datatypes.JSON `json:"fallbacks" gorm:"type:jsonb;comment:降级链([{model,fallbacks}])" swaggertype:"object"`
	AllowedFails    int            `json:"allowedFails" gorm:"default:3;comment:允许连续失败次数(健康摘除阈值)"`
	CooldownTime    int            `json:"cooldownTime" gorm:"default:60;comment:冷却时间(秒)"`
	NumRetries      int            `json:"numRetries" gorm:"default:2;comment:全局重试次数"`
	Timeout         int            `json:"timeout" gorm:"default:30;comment:全局超时(秒)"`
	Config          datatypes.JSON `json:"config" gorm:"type:jsonb;comment:扩展配置(预留)" swaggertype:"object"`
}

func (RouterSettings) TableName() string {
	return "gateway_router_settings"
}

// DefaultRouterSettings 默认路由策略(首次取时 FirstOrCreate 用)。
func DefaultRouterSettings() RouterSettings {
	return RouterSettings{
		RoutingStrategy: RoutingStrategySimpleShuffle,
		Fallbacks:       datatypes.JSON([]byte("[]")),
		AllowedFails:    3,
		CooldownTime:    60,
		NumRetries:      2,
		Timeout:         30,
		Config:          datatypes.JSON([]byte("{}")),
	}
}
