package utils

import (
	"os"
	"testing"
	"time"

	"github.com/hllkk/devops-admin/server/config"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/songzhibin97/gkit/cache/local_cache"
	"go.uber.org/zap"
)

// TestMain 为 utils 包测试初始化全局依赖（NewJWT 读 OPS_CONFIG，黑名单读 BlackCache，GetClaims 读 OPS_LOG）。
func TestMain(m *testing.M) {
	global.OPS_CONFIG = config.Server{
		JWT: config.JWT{
			SigningKey:    "test-signing-key",
			ExpiresTime:   "1h",
			RefreshExTime: "168h",
			BufferTime:    "1d",
			Issuer:        "test",
		},
	}
	global.BlackCache = local_cache.NewCache(local_cache.SetDefaultExpire(time.Hour))
	global.OPS_LOG = zap.NewNop()
	os.Exit(m.Run())
}
