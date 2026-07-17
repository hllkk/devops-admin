package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/service"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/datascope"
)

// DataScope 数据权限中间件: 在 JWTAuth 之后, 依据 claims 构建数据权限身份,
// 并注入 c.Request.Context(), 供 Service 层统一 WithContext(ctx) 透传到 GORM 回调消费。
// 这补上了历史缺口: 此前 jwt.go 只 c.Set("claims"), 身份没进 request.Context()。
// GetUserID/GetUserRoleId 已统一返回 int64(与雪花主键一致), 直接透传 BuildIdentity。
func DataScope() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := utils.GetUserID(c)
		if userID == 0 {
			c.Next()
			return
		}
		roleID := utils.GetUserRoleId(c)
		id, err := service.ServiceGroupApp.SystemServiceGroup.DataScopeService.
			BuildIdentity(c.Request.Context(), userID, roleID)
		if err == nil && id != nil {
			c.Request = c.Request.WithContext(datascope.WithIdentity(c.Request.Context(), id))
		}
		c.Next()
	}
}
