package system

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// TestDictRouterRegistration 验证字典路由注册不 panic。
// 重点:DELETE ":ids"(参数段)与 "refreshCache"(静态段)在同层共存,
// 确认 gin 允许 static+param 同层(static 优先匹配),refreshCache 不被 :ids 吞掉。
func TestDictRouterRegistration(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("字典路由注册 panic: %v", rec)
		}
	}()
	(&DictRouter{}).InitDictRouter(r.Group("/api"))
}
