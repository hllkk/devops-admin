package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/global"
)

// SetLoginCookies 写入 access/refresh 两个 httpOnly cookie。
// 属性：HttpOnly=true、SameSite=Strict、Secure=动态(RequestIsSecure)、Path=/、Domain=""。
func SetLoginCookies(c *gin.Context, accessToken, refreshToken string) {
	secure := RequestIsSecure(c)
	c.SetSameSite(http.SameSiteStrictMode)

	c.SetCookie("token", accessToken, int(accessExpirySeconds().Seconds()), "/", "", secure, true)
	c.SetCookie("refresh-token", refreshToken, int(refreshExpirySeconds().Seconds()), "/", "", secure, true)
}

// ClearLoginCookies 清除登录 cookie（与 Set 时同 SameSite/Secure，确保能被覆盖清除）。
func ClearLoginCookies(c *gin.Context) {
	secure := RequestIsSecure(c)
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("token", "", -1, "/", "", secure, true)
	c.SetCookie("refresh-token", "", -1, "/", "", secure, true)
}

func accessExpirySeconds() time.Duration {
	dr, err := ParseDuration(global.OPS_CONFIG.JWT.ExpiresTime)
	if err != nil || dr <= 0 {
		dr = 7 * 24 * time.Hour
	}
	return dr
}

func refreshExpirySeconds() time.Duration {
	dr, err := ParseDuration(global.OPS_CONFIG.JWT.RefreshExTime)
	if err != nil || dr <= 0 {
		dr = 168 * time.Hour
	}
	return dr
}
