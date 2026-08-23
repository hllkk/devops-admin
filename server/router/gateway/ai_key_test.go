package gateway

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// TestAiKeyRouterRegistration 验证 AI 密钥路由注册不 panic。
// 重点:GET "identity/my"(多级静态段)、GET "list" 与 GET ":id"(参数段)在同层共存，
// 确认 gin 允许 static+param 同层(static 优先匹配)。
func TestAiKeyRouterRegistration(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("AI 密钥路由注册 panic: %v", rec)
		}
	}()
	(&AiKeyRouter{}).InitAiKeyRouter(r.Group("/api"))
}
