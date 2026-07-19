package system

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRoleRouterRegistration 验证角色路由注册不 panic。
// 重点:PUT ""(root handler)与 "changeStatus"/"authUser/*"(static children)同节点共存,
// 以及 DELETE ":ids" 与各 GET/PUT static 在同 group 共存。
func TestRoleRouterRegistration(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("角色路由注册 panic: %v", rec)
		}
	}()
	(&RoleRouter{}).InitRoleRouter(r.Group("/api"))
}
