package gateway

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// TestUsageRouterRegistration 验证用量统计路由注册不 panic。
func TestUsageRouterRegistration(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("用量路由注册 panic: %v", rec)
		}
	}()
	(&UsageRouter{}).InitUsageRouter(r.Group("/api"))
}
