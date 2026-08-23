package gateway

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// TestModelRouterRegistration 验证模型路由注册不 panic。
// 重点:GET "active"(静态段)、GET "publish/:id"(静态+参数段)与 GET ":id"(参数段)在同层共存,
// 确认 gin 允许 static+param 同层(static 优先匹配)。
func TestModelRouterRegistration(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("模型路由注册 panic: %v", rec)
		}
	}()
	(&ModelRouter{}).InitModelRouter(r.Group("/api"))
}

// TestDeploymentRouterRegistration 验证部署路由注册不 panic(list/test 静态段与 :ids 参数段同层)。
func TestDeploymentRouterRegistration(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("部署路由注册 panic: %v", rec)
		}
	}()
	(&DeploymentRouter{}).InitDeploymentRouter(r.Group("/api"))
}
