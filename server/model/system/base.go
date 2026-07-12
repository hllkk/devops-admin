package system

import (
	"time"

	"gorm.io/gorm"
)

// AuditModel 对齐前端 Api.Common.CommonRecord 的审计字段，仅用于 system 模块
// 需贴合 RuoYi/前端契约的对外模型（SysUser / SysRole）。
// 其它模型继续使用 global.OPS_MODEL。
type AuditModel struct {
	CreateBy   string         `gorm:"column:create_by;size:64;default:''" json:"createBy"`
	CreateTime time.Time      `gorm:"column:create_time;autoCreateTime" json:"createTime"`
	UpdateBy   string         `gorm:"column:update_by;size:64;default:''" json:"updateBy"`
	UpdateTime time.Time      `gorm:"column:update_time;autoUpdateTime" json:"updateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// 状态码（char），与前端 EnableStatus 对齐：0 正常 / 1 停用
const (
	StatusEnable  = "0"
	StatusDisable = "1"
)
