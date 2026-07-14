package system

import (
	"testing"

	"github.com/hllkk/devops-admin/server/global"
	sysReq "github.com/hllkk/devops-admin/server/model/system/request"
)

// TestApplyRedisConfig 验证编排器在 WriteConfig 前把 Redis 配置落到 OPS_CONFIG：
// UseRedis 置 true，Redis 段按 ToRedisConfig() 填充。
func TestApplyRedisConfig(t *testing.T) {
	global.OPS_CONFIG.System.UseRedis = false
	global.OPS_CONFIG.Redis.Addr = ""

	applyRedisConfig(sysReq.InitDB{RedisAddr: "127.0.0.1:6379", RedisPassword: "pw", RedisDB: 2})

	if !global.OPS_CONFIG.System.UseRedis {
		t.Fatal("UseRedis 应被置为 true")
	}
	if global.OPS_CONFIG.Redis.Addr != "127.0.0.1:6379" || global.OPS_CONFIG.Redis.DB != 2 {
		t.Fatalf("Redis 配置未落盘: %+v", global.OPS_CONFIG.Redis)
	}
}
