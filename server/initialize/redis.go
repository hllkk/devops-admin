package initialize

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/hllkk/devops-admin/server/config"
	"github.com/hllkk/devops-admin/server/global"
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
	pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}
	global.OPS_LOG.Info("redis connect ping response:", zap.String("name", redisCfg.Name), zap.String("pong", pong))
	return client, nil
}

func Redis() {
	redisClient, err := DialRedis(global.OPS_CONFIG.Redis)
	if err != nil {
		panic(err)
	}
	global.OPS_REDIS = redisClient
}

func RedisList() {
	redisMap := make(map[string]redis.UniversalClient)
	for _, redisCfg := range global.OPS_CONFIG.RedisList {
		client, err := DialRedis(redisCfg)
		if err != nil {
			panic(err)
		}
		redisMap[redisCfg.Name] = client
	}
	global.OPS_REDISList = redisMap
}
