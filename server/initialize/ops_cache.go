package initialize

import (
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
	"github.com/hllkk/devops-admin/server/utils/ops_cache"
)

// InitOpsCache 初始化通用缓存句柄 global.OPS_CACHE。
// 必须在 Redis 初始化之后调用：有 Redis 用 Redis 后端，否则用内存后端。
func InitOpsCache() {
	if global.OPS_REDIS != nil {
		global.OPS_CACHE = ops_cache.NewRedisCache(global.OPS_REDIS)
		logger.Bg().Mod("system").Info("OPS_CACHE 使用 Redis 后端")
		return
	}
	dr, err := utils.ParseDuration(global.OPS_CONFIG.JWT.ExpiresTime)
	if err != nil {
		// JWT 过期配置非法应在启动期暴露
		panic(err)
	}
	global.OPS_CACHE = ops_cache.NewMemoryCache(dr)
	logger.Bg().Mod("system").Field("defaultExpire", dr).Info("OPS_CACHE 使用内存后端")
}
