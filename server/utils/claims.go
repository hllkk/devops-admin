package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// RequestIsSecure 判定当前请求是否走 HTTPS。
// 优先 c.Request.TLS(直连 HTTPS);其次 X-Forwarded-Proto: https(反代场景,#2 配置 TrustedProxies 后才可信)。
func RequestIsSecure(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	return c.GetHeader("X-Forwarded-Proto") == "https"
}

// ClearToken 清除 x-token cookie，domain 留空走 host-only（见 SetToken 说明）。
func ClearToken(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode) // 防 CSRF;Strict 禁止任何跨站携带,管理后台无外链带登录态需求
	c.SetCookie("x-token", "", -1, "/", "", RequestIsSecure(c), true)
}

// SetToken 登录成功后下发 x-token，走 httpOnly cookie。
// domain 传空串即为 host-only cookie，由浏览器绑定到实际访问的 host；
// 不再按 c.Request.Host 推断 domain：经反向代理或 vite proxy(changeOrigin) 转发时 Host 会被改写，
// 误设 domain 会导致浏览器丢弃该 cookie（表现为登录成功但后续接口 401 未登录）。
// secure 按 RequestIsSecure 动态设置(HTTPS 下带 Secure,防中间人窃听);SameSite=Strict 防 CSRF
// (社交登录绑定走前端同站 POST 提交 code,不依赖跨站导航携带 cookie,Strict 不影响其流程)。
func SetToken(c *gin.Context, token string, maxAge int) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("x-token", token, maxAge, "/", "", RequestIsSecure(c), true)
}

func GetToken(c *gin.Context) string {
	token := c.Request.Header.Get("x-token")
	if token == "" {
		j := NewJWT()
		token, _ = c.Cookie("x-token")
		claims, err := j.ParseToken(token)
		if err != nil {
			logger.WithCtx(c.Request.Context()).Mod("system").Error("重新写入cookie token失败,未能成功解析token,请检查请求头是否存在x-token且claims是否为规定结构")
			return token
		}
		SetToken(c, token, int(claims.ExpiresAt.Unix()-time.Now().Unix()))
	}
	return token
}

func GetClaims(c *gin.Context) (*systemReq.CustomClaims, error) {
	token := GetToken(c)
	j := NewJWT()
	claims, err := j.ParseToken(token)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("system").Error("从Gin的Context中获取从jwt解析信息失败, 请检查请求头是否存在x-token且claims是否为规定结构")
	}
	return claims, err
}

// GetUserID 从Gin的Context中获取从jwt解析出来的用户ID
func GetUserID(c *gin.Context) int64 {
	if claims, exists := c.Get("claims"); !exists {
		if cl, err := GetClaims(c); err != nil {
			return 0
		} else {
			return cl.BaseClaims.ID
		}
	} else {
		waitUse := claims.(*systemReq.CustomClaims)
		return waitUse.BaseClaims.ID
	}
}

// GetUserUuid 从Gin的Context中获取从jwt解析出来的用户UUID
func GetUserUuid(c *gin.Context) uuid.UUID {
	if claims, exists := c.Get("claims"); !exists {
		if cl, err := GetClaims(c); err != nil {
			return uuid.UUID{}
		} else {
			return cl.UUID
		}
	} else {
		waitUse := claims.(*systemReq.CustomClaims)
		return waitUse.UUID
	}
}

// GetUserRoleId 从Gin的Context中获取从jwt解析出来的用户角色id
func GetUserRoleId(c *gin.Context) int64 {
	if claims, exists := c.Get("claims"); !exists {
		if cl, err := GetClaims(c); err != nil {
			return 0
		} else {
			return cl.RoleId
		}
	} else {
		waitUse := claims.(*systemReq.CustomClaims)
		return waitUse.RoleId
	}
}

// GetUserInfo 从Gin的Context中获取从jwt解析出来的用户角色id
func GetUserInfo(c *gin.Context) *systemReq.CustomClaims {
	if claims, exists := c.Get("claims"); !exists {
		if cl, err := GetClaims(c); err != nil {
			return nil
		} else {
			return cl
		}
	} else {
		waitUse := claims.(*systemReq.CustomClaims)
		return waitUse
	}
}

// GetUserName 从Gin的Context中获取从jwt解析出来的用户名
func GetUserName(c *gin.Context) string {
	if claims, exists := c.Get("claims"); !exists {
		if cl, err := GetClaims(c); err != nil {
			return ""
		} else {
			return cl.Username
		}
	} else {
		waitUse := claims.(*systemReq.CustomClaims)
		return waitUse.Username
	}
}

func LoginToken(user system.Login) (token string, claims systemReq.CustomClaims, err error) {
	j := NewJWT()
	claims = j.CreateClaims(systemReq.BaseClaims{
		UUID:       user.GetUUID(),
		ID:         user.GetUserId(),
		NickName:   user.GetNickname(),
		Username:   user.GetUsername(),
		RoleId:     user.GetRoleId(),
		SuperAdmin: user.GetSuperAdmin(),
	})
	token, err = j.CreateToken(claims)
	return
}

// LoginTokenWithExpire 签发登录 token 可携带 MustChangePwd 强制改密标记
func LoginTokenWithExpire(user system.Login, mustChangePwd bool) (token string, claims systemReq.CustomClaims, err error) {
	j := NewJWT()
	claims = j.CreateClaims(systemReq.BaseClaims{
		UUID:       user.GetUUID(),
		ID:         user.GetUserId(),
		NickName:   user.GetNickname(),
		Username:   user.GetUsername(),
		RoleId:     user.GetRoleId(),
		SuperAdmin: user.GetSuperAdmin(),
	})
	claims.MustChangePwd = mustChangePwd
	token, err = j.CreateToken(claims)
	return
}
