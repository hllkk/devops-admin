package gateway

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// TestMCPRouterRegistration 验证 MCP 路由注册不 panic。
// 重点:GET "list"/"active"/"connect-config/:id"(静态段)与 GET ":id"/DELETE ":ids"(参数段)
// 在同层共存,确认 gin 允许 static+param 同层(static 优先匹配)。
func TestMCPRouterRegistration(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("MCP 路由注册 panic: %v", rec)
		}
	}()
	(&MCPRouter{}).InitMCPRouter(r.Group("/api"))
}
