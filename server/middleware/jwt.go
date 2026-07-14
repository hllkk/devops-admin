package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/utils"
)

// expiredTokenCode 命中前端 VITE_SERVICE_EXPIRED_TOKEN_CODES，触发刷新；刷新失败由 refresh 端点返回 8888 登出。
const expiredTokenCode = "9999"

// JWTAuth 校验 access token：从 Authorization 头或 token cookie 取值，强制 audience=access，校验黑名单。
// 失败统一 HTTP 200 + code "9999"（前端据此刷新或登出）。不再下发 new-token 头（改由 refresh 端点轮换）。
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := utils.GetToken(c)
		if err != nil || token == "" {
			response.NoAuthWithCode(expiredTokenCode, "未登录或令牌失效，请登录", c)
			c.Abort()
			return
		}
		if utils.IsBlacklisted(token) {
			response.NoAuthWithCode(expiredTokenCode, "令牌已失效，请重新登录", c)
			c.Abort()
			return
		}
		j := utils.NewJWT()
		claims, err := j.ParseAccessToken(token)
		if err != nil {
			response.NoAuthWithCode(expiredTokenCode, "登录已过期，请重新登录", c)
			c.Abort()
			return
		}
		c.Set("claims", claims)
		c.Next()
	}
}
