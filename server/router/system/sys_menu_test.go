package system

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// TestMenuRouterRegistration 验证菜单路由注册不 panic。
// 重点:DELETE ":menuId"(参数段)与 "cascade/:menuIds"(静态段+参数段)在同层共存,
// 确认 gin 允许 static+param 同层(static 优先匹配)。
func TestMenuRouterRegistration(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("菜单路由注册 panic: %v", rec)
		}
	}()
	(&MenuRouter{}).InitMenuRouter(r.Group("/api"))
}
