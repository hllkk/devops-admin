package gateway

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// TestDashboardRouterRegistration 验证看板路由注册不 panic。
func TestDashboardRouterRegistration(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("看板路由注册 panic: %v", rec)
		}
	}()
	(&DashboardRouter{}).InitDashboardRouter(r.Group("/api"))
}
