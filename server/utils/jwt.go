package utils

import (
	"context"
	"errors"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system/request"
)

const (
	AudienceAccess  = "access"
	AudienceRefresh = "refresh"
)

type JWT struct {
	SigningKey []byte
}

var (
	TokenValid            = errors.New("未知错误")
	TokenExpired          = errors.New("token已过期")
	TokenNotValidYet      = errors.New("token尚未激活")
	TokenMalformed        = errors.New("这不是一个token")
	TokenSignatureInvalid = errors.New("无效签名")
	TokenInvalid          = errors.New("无法处理此token")
	TokenAudienceMismatch = errors.New("token受众不匹配")
)

func NewJWT() *JWT {
	return &JWT{
		[]byte(global.OPS_CONFIG.JWT.SigningKey),
	}
}

// createClaims 构造指定 audience 与过期时长的 claims。
func (j *JWT) createClaims(bc request.BaseClaims, audience string, exp time.Duration) request.CustomClaims {
	return request.CustomClaims{
		BaseClaims: bc,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{audience},
			NotBefore: jwt.NewNumericDate(time.Now().Add(-1000 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
			Issuer:    global.OPS_CONFIG.JWT.Issuer,
		},
	}
}

// CreateAccessToken 签发 access token（audience=access，业务接口鉴权用）。
func (j *JWT) CreateAccessToken(bc request.BaseClaims) (string, error) {
	ep, err := ParseDuration(global.OPS_CONFIG.JWT.ExpiresTime)
	if err != nil || ep <= 0 {
		ep = 7 * 24 * time.Hour
	}
	return j.CreateToken(j.createClaims(bc, AudienceAccess, ep))
}

// CreateRefreshToken 签发 refresh token（audience=refresh，仅 /auth/refreshToken 使用）。
func (j *JWT) CreateRefreshToken(bc request.BaseClaims) (string, error) {
	rp, err := ParseDuration(global.OPS_CONFIG.JWT.RefreshExTime)
	if err != nil || rp <= 0 {
		rp = 168 * time.Hour
	}
	return j.CreateToken(j.createClaims(bc, AudienceRefresh, rp))
}

// CreateToken 用给定 claims 签名。
func (j *JWT) CreateToken(claims request.CustomClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.SigningKey)
}

// ParseToken 解析 token（不校验 audience）。
func (j *JWT) ParseToken(tokenString string) (*request.CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &request.CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.SigningKey, nil
	})
	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, TokenExpired
		case errors.Is(err, jwt.ErrTokenMalformed):
			return nil, TokenMalformed
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, TokenSignatureInvalid
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			return nil, TokenNotValidYet
		default:
			return nil, TokenInvalid
		}
	}
	if token != nil {
		if claims, ok := token.Claims.(*request.CustomClaims); ok && token.Valid {
			return claims, nil
		}
	}
	return nil, TokenValid
}

// ParseAccessToken 解析并强制 audience=access；业务接口用，拒绝 refresh token。
func (j *JWT) ParseAccessToken(tokenString string) (*request.CustomClaims, error) {
	claims, err := j.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}
	if !hasAudience(claims, AudienceAccess) {
		return nil, TokenAudienceMismatch
	}
	return claims, nil
}

// ParseRefreshToken 解析并强制 audience=refresh。
func (j *JWT) ParseRefreshToken(tokenString string) (*request.CustomClaims, error) {
	claims, err := j.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}
	if !hasAudience(claims, AudienceRefresh) {
		return nil, TokenAudienceMismatch
	}
	return claims, nil
}

func hasAudience(claims *request.CustomClaims, audience string) bool {
	for _, a := range claims.Audience {
		if a == audience {
			return true
		}
	}
	return false
}

// JoinBlacklist 将 token 加入黑名单（内存缓存，进程级；进程重启失效）。
func JoinBlacklist(token string) {
	if token == "" {
		return
	}
	global.BlackCache.SetDefault(token, struct{}{})
}

// IsBlacklisted 判断 token 是否在黑名单。
func IsBlacklisted(token string) bool {
	if token == "" {
		return false
	}
	_, ok := global.BlackCache.Get(token)
	return ok
}

// 以下旧方法供 Task 3 重写 middleware 前过渡编译，Task 4 末尾统一删除。

// CreateClaims 已废弃（原滑动续期），Task 4 删除。
func (j *JWT) CreateClaims(baseClaims request.BaseClaims) request.CustomClaims {
	bf, _ := ParseDuration(global.OPS_CONFIG.JWT.BufferTime)
	ep, _ := ParseDuration(global.OPS_CONFIG.JWT.ExpiresTime)
	claims := request.CustomClaims{
		BaseClaims: baseClaims,
		BufferTime: int64(bf / time.Second),
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{"GVA"},
			NotBefore: jwt.NewNumericDate(time.Now().Add(-1000)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ep)),
			Issuer:    global.OPS_CONFIG.JWT.Issuer,
		},
	}
	return claims
}

// CreateTokenByOldToken 已废弃（原滑动续期），Task 4 删除。
func (j *JWT) CreateTokenByOldToken(oldToken string, claims request.CustomClaims) (string, error) {
	v, err, _ := global.OPS_Concurrency_Control.Do("JWT:"+oldToken, func() (interface{}, error) {
		return j.CreateToken(claims)
	})
	return v.(string), err
}

// SetRedisJWT 已废弃（多点登录），Task 4 删除。
func SetRedisJWT(jwt string, userName string) (err error) {
	dr, err := ParseDuration(global.OPS_CONFIG.JWT.ExpiresTime)
	if err != nil {
		return err
	}
	timer := dr
	err = global.OPS_REDIS.Set(context.Background(), userName, jwt, timer).Err()
	return err
}
