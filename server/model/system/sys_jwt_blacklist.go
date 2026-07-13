package system

import (
	"github.com/hllkk/devops-admin/server/global"
)

type JwtBlacklist struct {
	ID  int64 `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	global.OPS_MODEL
	Jwt string `gorm:"type:text;comment:jwt"`
}
