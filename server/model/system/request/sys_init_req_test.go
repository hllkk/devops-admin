package request

import "testing"

func TestToRedisConfig(t *testing.T) {
	i := InitDB{RedisAddr: "127.0.0.1:6379", RedisPassword: "pw", RedisDB: 2}
	c := i.ToRedisConfig()
	if c.Addr != "127.0.0.1:6379" || c.Password != "pw" || c.DB != 2 {
		t.Fatalf("redis config mismatch: %+v", c)
	}
	if c.UseCluster {
		t.Fatal("单实例模式 UseCluster 应为 false")
	}
}
