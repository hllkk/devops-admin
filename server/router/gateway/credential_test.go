package gateway

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// TestCredentialRouterRegistration 验证凭证路由注册不 panic。
// 重点:GET "provider-fields"/POST "resync"(静态段)与 GET ":id"/DELETE ":ids"(参数段)在同层共存,
// 确认 gin 允许 static+param 同层(static 优先匹配)。
func TestCredentialRouterRegistration(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("凭证路由注册 panic: %v", rec)
		}
	}()
	(&CredentialRouter{}).InitCredentialRouter(r.Group("/api"))
}
