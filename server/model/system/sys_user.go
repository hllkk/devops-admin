package system

import (
	"github.com/google/uuid"
)

type Login interface {
	GetUsername() string
	GetNickname() string
	GetUUID() uuid.UUID
	GetUserId() uint
	GetAuthorityId() uint
	GetUserInfo() any
}

// var _ Login = new(SysUser)

// type SysUser struct {
// 	global.OPS_MODEL
// }
