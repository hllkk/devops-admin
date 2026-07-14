package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hllkk/devops-admin/server/global"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
)

// GetToken 取 access token：Authorization: Bearer 头优先 → token httpOnly cookie。
// 禁止从 URL 查询参数取 token，避免 JWT 进入 URL（Nginx 日志/浏览器历史/Referer 泄漏）。
func GetToken(c *gin.Context) (string, error) {
	if authorization := c.Request.Header.Get("Authorization"); authorization != "" {
		if !strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
			return "", fmt.Errorf("invalid authorization header format, expected 'Bearer <token>', got: %q", authorization)
		}
		return authorization[7:], nil
	}
	if token, err := c.Cookie("token"); err == nil && token != "" {
		return token, nil
	}
	return "", errors.New("token not found in header or cookie")
}

// RequestIsSecure 判断请求是否经 HTTPS 传输，用于动态决定 cookie 的 Secure 标志。
// 优先 X-Forwarded-Proto（多层反代反映浏览器真实协议），回退 c.Request.TLS。
func RequestIsSecure(c *gin.Context) bool {
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		return strings.EqualFold(proto, "https")
	}
	return c.Request.TLS != nil
}

// GetClaims 取 access token 并解析为 claims（业务接口强制 access，拒绝 refresh token）。
func GetClaims(c *gin.Context) (*systemReq.CustomClaims, error) {
	token, err := GetToken(c)
	if err != nil {
		return nil, err
	}
	j := NewJWT()
	claims, err := j.ParseAccessToken(token) // 业务接口强制 access token，拒绝 refresh token
	if err != nil {
		global.OPS_LOG.Error("从请求中解析 access token 失败，请检查 Authorization 头或 token cookie")
		return nil, err
	}
	return claims, nil
}

// GetUserID 从 Context 中获取 jwt 用户ID。
func GetUserID(c *gin.Context) uint {
	if claims, exists := c.Get("claims"); exists {
		return claims.(*systemReq.CustomClaims).BaseClaims.ID
	}
	if cl, err := GetClaims(c); err == nil {
		return cl.BaseClaims.ID
	}
	return 0
}

// GetUserUuid 从 Context 中获取 jwt 用户UUID。
func GetUserUuid(c *gin.Context) uuid.UUID {
	if claims, exists := c.Get("claims"); exists {
		return claims.(*systemReq.CustomClaims).UUID
	}
	if cl, err := GetClaims(c); err == nil {
		return cl.UUID
	}
	return uuid.UUID{}
}

// GetUserAuthorityId 从 Context 中获取 jwt 角色ID。
func GetUserAuthorityId(c *gin.Context) uint {
	if claims, exists := c.Get("claims"); exists {
		return claims.(*systemReq.CustomClaims).RoleId
	}
	if cl, err := GetClaims(c); err == nil {
		return cl.RoleId
	}
	return 0
}

// GetUserInfo 从 Context 中获取完整 claims。
func GetUserInfo(c *gin.Context) *systemReq.CustomClaims {
	if claims, exists := c.Get("claims"); exists {
		return claims.(*systemReq.CustomClaims)
	}
	if cl, err := GetClaims(c); err == nil {
		return cl
	}
	return nil
}

// GetUserName 从 Context 中获取用户名。
func GetUserName(c *gin.Context) string {
	if claims, exists := c.Get("claims"); exists {
		return claims.(*systemReq.CustomClaims).Username
	}
	if cl, err := GetClaims(c); err == nil {
		return cl.Username
	}
	return ""
}

