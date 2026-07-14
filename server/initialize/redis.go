package initialize

import (
	"context"
	"time"

	"github.com/hllkk/devops-admin/server/config"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/utils"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// DialRedis 按 cfg 建客户端并 Ping；不写 global，失败返回 error（不 panic）。
// Redis()（启动）与 dbReadyCallback（首初始化后即时连接）共用此函数。
func DialRedis(redisCfg config.Redis) (redis.UniversalClient, error) {
	var client redis.UniversalClient
	if redisCfg.UseCluster {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    redisCfg.ClusterAddrs,
			Password: redisCfg.Password,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     redisCfg.Addr,
			Password: redisCfg.Password,
			DB:       redisCfg.DB,
		})
	}
	// Ping 使用 3 秒超时，避免 Redis 不可达时阻塞启动
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		global.OPS_LOG.Error("redis connect ping failed", zap.String("name", redisCfg.Name), zap.Error(err))
		return nil, err
	}
	global.OPS_LOG.Info("redis connect ping response:", zap.String("name", redisCfg.Name), zap.String("pong", pong))
	return client, nil
}

func Redis() {
	// 检查 Redis 配置是否有效
	if !utils.IsValidRedisConfig(global.OPS_CONFIG.Redis) {
		global.OPS_LOG.Warn("Redis 配置无效或未配置，跳过 Redis 初始化")
		return
	}

	redisClient, err := DialRedis(global.OPS_CONFIG.Redis)
	if err != nil {
		zap.L().Fatal("Redis 初始化失败，拒绝启动", zap.String("name", global.OPS_CONFIG.Redis.Name), zap.Error(err))
	}
	global.OPS_REDIS = redisClient
}

func RedisList() {
	redisMap := make(map[string]redis.UniversalClient)
	for _, redisCfg := range global.OPS_CONFIG.RedisList {
		if !utils.IsValidRedisConfig(redisCfg) {
			global.OPS_LOG.Warn("Redis 列表配置无效，跳过", zap.String("name", redisCfg.Name))
			continue
		}
		client, err := DialRedis(redisCfg)
		if err != nil {
			zap.L().Fatal("Redis 列表项初始化失败，拒绝启动", zap.String("name", redisCfg.Name), zap.Error(err))
		}
		redisMap[redisCfg.Name] = client
	}
	global.OPS_REDISList = redisMap
}
