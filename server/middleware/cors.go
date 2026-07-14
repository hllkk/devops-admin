package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Cors 放行跨域并允许携带 cookie（credentials）。
// Origin 动态回显请求来源，配合 httpOnly cookie + 前端 withCredentials。
// 生产环境建议在外层网关收敛白名单。
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		if origin := c.Request.Header.Get("Origin"); origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Token,X-User-Id,x-request-id,apifoxtoken")
			c.Header("Access-Control-Allow-Methods", "POST,GET,PUT,DELETE,OPTIONS")
			c.Header("Access-Control-Expose-Headers", "Content-Length,Content-Type,New-Token,New-Expires-At,Download-Filename")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
