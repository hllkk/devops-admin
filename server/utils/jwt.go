package utils

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/model/system/request"
	"go.uber.org/zap"
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

// sha256Hex 返回字符串的 SHA256 十六进制摘要。
// 用于 JwtBlacklist.JwtHash，使 DB 回退查询走定长索引而非全表扫描。
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
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

// parseBlacklistClaims 解析黑名单 token 的 claims 与剩余有效期。
// token 无效或已过期时 ok=false，调用方应直接返回（无需拉黑）。
func (j *JWT) parseBlacklistClaims(token string) (claims *request.CustomClaims, expiration time.Duration, ok bool) {
	c, err := j.ParseToken(token)
	if err != nil {
		return nil, 0, false
	}
	if exp := time.Until(c.ExpiresAt.Time); exp > 0 {
		return c, exp, true
	}
	return nil, 0, false
}

// JoinBlacklist 将 token 加入黑名单（Redis + 数据库双写）
func (j *JWT) JoinBlacklist(token string) error {
	claims, expiration, ok := j.parseBlacklistClaims(token)
	if !ok {
		// token 无效或已过期，无需加入黑名单
		return nil
	}

	// Redis：将 token 加入黑名单，设置过期时间等于 token 剩余有效期
	if global.OPS_REDIS != nil {
		if err := global.OPS_REDIS.Set(context.Background(), "blacklist:"+token, "1", expiration).Err(); err != nil {
			return err
		}
	}

	// DB：同时保存作为审计记录（JwtHash 走索引，加速 Redis miss 时的回退查询）
	blacklist := system.JwtBlacklist{
		Jwt:       token,
		JwtHash:   sha256Hex(token),
		ExpiresAt: claims.ExpiresAt.Time,
	}
	if global.OPS_DB != nil {
		if err := global.OPS_DB.Create(&blacklist).Error; err != nil {
			// DB 写入失败不阻断（Redis 已写入生效），仅记录告警
			global.OPS_LOG.Warn("JWT 黑名单 DB 写入失败", zap.Error(err))
		}
	}

	// 本地缓存兜底（Redis 不可用时进程级缓存）
	global.BlackCache.SetDefault(token, struct{}{})

	return nil
}

// IsBlacklisted 判断 token 是否在黑名单中（Redis 优先 → 本地缓存 → DB 回退）
func (j *JWT) IsInBlacklist(token string) bool {
	if token == "" {
		return false
	}

	// 1. Redis 优先查询
	if global.OPS_REDIS != nil {
		val, err := global.OPS_REDIS.Get(context.Background(), "blacklist:"+token).Result()
		if err == nil && val == "1" {
			return true
		}
	}

	// 2. 本地缓存兜底
	_, ok := global.BlackCache.Get(token)
	if ok {
		return true
	}

	// 3. DB 回退查询（按 JwtHash 走等值索引，避免全表扫描）
	if global.OPS_DB != nil {
		var count int64
		if err := global.OPS_DB.Model(&system.JwtBlacklist{}).Where("jwt_hash = ?", sha256Hex(token)).Count(&count).Error; err == nil && count > 0 {
			// 同步回 Redis + 本地缓存，恢复缓存
			if global.OPS_REDIS != nil {
				if claims, err := j.ParseToken(token); err == nil {
					expiration := time.Until(claims.ExpiresAt.Time)
					if expiration > 0 {
						_ = global.OPS_REDIS.Set(context.Background(), "blacklist:"+token, "1", expiration).Err()
					}
				}
			}
			global.BlackCache.SetDefault(token, struct{}{})
			return true
		}
	}

	return false
}

// JoinBlacklistRedisOnly 将 token 加入黑名单（仅写入 Redis，用于高性能场景）
func (j *JWT) JoinBlacklistRedisOnly(token string) error {
	_, expiration, ok := j.parseBlacklistClaims(token)
	if !ok {
		return nil
	}
	if global.OPS_REDIS != nil {
		return global.OPS_REDIS.Set(context.Background(), "blacklist:"+token, "1", expiration).Err()
	}
	// Redis 不可用时回退到本地缓存
	global.BlackCache.SetDefault(token, struct{}{})
	return nil
}

// ---- 包级兼容函数（旧调用点 utils.IsBlacklisted / utils.JoinBlacklist 映射到新方法）----

// IsBlacklisted 判断 token 是否在黑名单（包级兼容，内部调用 JWT.IsInBlacklist）
func IsBlacklisted(token string) bool {
	return NewJWT().IsInBlacklist(token)
}

// JoinBlacklist 将 token 加入黑名单（包级兼容，内部调用 JWT.JoinBlacklist）
func JoinBlacklist(token string) {
	if token == "" {
		return
	}
	_ = NewJWT().JoinBlacklist(token)
}
