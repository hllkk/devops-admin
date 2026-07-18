package request

import (
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// CustomClaims structure
type CustomClaims struct {
	BaseClaims
	BufferTime    int64
	MustChangePwd bool `json:"mustChangePwd"`
	jwt.RegisteredClaims
}

type BaseClaims struct {
	UUID       uuid.UUID
	ID         int64
	Username   string
	NickName   string
	RoleId     int64
	SuperAdmin bool // 超管标志:CasbinHandler 中间件据此放行,绕过策略校验
}
