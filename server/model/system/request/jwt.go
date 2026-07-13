package request

import (
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// CustomClaims structure
type CustomClaims struct {
	BaseClaims
	BufferTime int64
	jwt.RegisteredClaims
}

type BaseClaims struct {
	UUID       uuid.UUID
	ID         uint
	Username   string
	NickName   string
	RoleId     uint
	SuperAdmin bool // 超管标记：登录时由 sys_role.super_admin 写入，casbin 中间件据此豁免
}
