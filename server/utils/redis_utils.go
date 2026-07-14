package utils

import "github.com/hllkk/devops-admin/server/config"

// IsValidRedisConfig 校验 Redis 配置是否有效（至少 addr 非空）。
// 空/无效配置不应尝试连接，避免无意义的连接超时阻塞启动。
func IsValidRedisConfig(cfg config.Redis) bool {
	return cfg.Addr != ""
}
