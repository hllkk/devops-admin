package middleware

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/config"
	"github.com/hllkk/devops-admin/server/global"
)

// Cors 直接放行所有跨域请求并放行所有 OPTIONS 方法（allow-all 模式使用）
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token,Authorization,Token,X-Token,X-User-Id,x-request-id,apifoxtoken")
		c.Header("Access-Control-Allow-Methods", "POST,GET,OPTIONS,DELETE,PUT")
		c.Header("Access-Control-Expose-Headers", "Content-Length,Access-Control-Allow-Origin,Access-Control-Allow-Headers,Content-Type,New-Token,New-Expires-At,Download-Filename")
		c.Header("Access-Control-Allow-Credentials", "true")

		// 放行所有 OPTIONS 方法
		if method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}

// CorsByRules 按照配置处理跨域请求
func CorsByRules() gin.HandlerFunc {
	// 配置为 allow-all，或开发模式下 strict-whitelist 自动降级为 allow-all
	if global.OPS_CONFIG.Cors.Mode == "allow-all" ||
		(global.OPS_CONFIG.Cors.Mode == "strict-whitelist" && gin.Mode() == gin.DebugMode) {
		return Cors()
	}
	return func(c *gin.Context) {
		whitelist := checkCors(c.GetHeader("origin"))

		// 通过检查，添加请求头
		if whitelist != nil {
			c.Header("Access-Control-Allow-Origin", whitelist.AllowOrigin)
			c.Header("Access-Control-Allow-Headers", whitelist.AllowHeaders)
			c.Header("Access-Control-Allow-Methods", whitelist.AllowMethods)
			c.Header("Access-Control-Expose-Headers", whitelist.ExposeHeaders)
			if whitelist.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		}

		// CORS 规范：无 Origin 头的请求不是跨域请求，不应被 CORS 拦截。
		// 同源请求（Origin 与 Host 一致）也不需要 CORS 检查。
		// 仅当 Origin 存在、非同源、且不匹配白名单时，strict-whitelist 模式才拒绝（403）。
		origin := c.GetHeader("origin")
		isSameOrigin := isSameOriginRequest(origin, c.GetHeader("Host"))
		if whitelist == nil && origin != "" && !isSameOrigin && global.OPS_CONFIG.Cors.Mode == "strict-whitelist" {
			c.AbortWithStatus(http.StatusForbidden)
		} else {
			// 非严格白名单模式，无论是否通过检查均放行所有 OPTIONS 方法
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusNoContent)
			}
		}
		// 处理请求
		c.Next()
	}
}

// checkCors 遍历白名单匹配 Origin
func checkCors(currentOrigin string) *config.CORSWhitelist {
	for _, whitelist := range global.OPS_CONFIG.Cors.Whitelist {
		if currentOrigin == whitelist.AllowOrigin {
			return &whitelist
		}
	}
	return nil
}

// isSameOriginRequest 判断请求是否同源
func isSameOriginRequest(origin, host string) bool {
	if origin == "" || host == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return u.Host == host
}
