package gateway

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// TestSkillRouterRegistration 验证 Skill 路由注册不 panic。
// 重点:GET "list"/"active"/"available"/"usage/list"/"download/:id"/"install-info/:id"/"publish/:id"(静态段)
// 与 GET ":id"/DELETE ":ids"(参数段)在同层共存,确认 gin 允许 static+param 同层(static 优先匹配);
// agent/:id/zip 挂公开组(与私有组同树注册,验证跨组静态段共存不冲突)。
func TestSkillRouterRegistration(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("Skill 路由注册 panic: %v", rec)
		}
	}()
	(&SkillRouter{}).InitSkillRouter(r.Group("/api"), r.Group("/api"))
}
