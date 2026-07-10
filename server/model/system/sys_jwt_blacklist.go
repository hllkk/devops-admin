package system

import (
	"github.com/hllkk/devops-admin/server/global"
)

type JwtBlacklist struct {
	global.OPS_MODEL
	Jwt string `gorm:"type:text;comment:jwt"`
}
