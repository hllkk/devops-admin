package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
)

// mustChangePwdAllowList 强制改密状态下允许访问的接口后缀(路径以这些后缀结尾才放行,其余 403)。
// /auth/getUserInfo:前端读登录态; /auth/logout:允许登出避免死锁; /system/user/profile/updatePwd:唯一改密入口。
var mustChangePwdAllowList = []string{
	"/auth/getUserInfo",
	"/auth/logout",
	"/system/user/profile/updatePwd",
}

// MustChangePwdGuard 当 jwt 携带 MustChangePwd=true 时 仅放行改密/用户信息/登出 其余 403
func MustChangePwdGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, exists := c.Get("claims")
		if !exists {
			c.Next()
			return
		}
		claims, ok := raw.(*systemReq.CustomClaims)
		if !ok || !claims.MustChangePwd {
			c.Next()
			return
		}
		path := c.Request.URL.Path
		for _, allow := range mustChangePwdAllowList {
			if strings.HasSuffix(path, allow) {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{
			"code": 7,
			"data": gin.H{"needChangePassword": true},
			"msg":  "密码已过期，请先修改密码",
		})
		c.Abort()
	}
}
