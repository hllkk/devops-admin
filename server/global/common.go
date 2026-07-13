package global

import (
	"time"

	"gorm.io/gorm"
)

// OPS_MODEL 生命周期基座：所有表通用的时间戳 + 软删除。不含主键——
// 主键由业务模型自定义（对外实体用业务命名 userId/roleId，内部表用 id），
// 雪花回调 ops:snowflake_id 按 PrioritizedPrimaryField 自动填充整型主键。
type OPS_MODEL struct {
	CreateTime time.Time      `gorm:"column:create_time;autoCreateTime" json:"createTime"`
	UpdateTime time.Time      `gorm:"column:update_time;autoUpdateTime" json:"updateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// OPS_AUDIT_MODEL 审计基座：在 OPS_MODEL 上加「操作人」追溯（CreateBy/UpdateBy），
// 用于对齐 RuoYi/前端 CommonRecord 的对外业务实体（用户/角色/菜单...）。
// 仅需生命周期、不需要审计的内部表（日志/黑名单等 append-only 系统记录）直接用 OPS_MODEL。
type OPS_AUDIT_MODEL struct {
	OPS_MODEL
	CreateBy string `gorm:"column:create_by;size:64;default:''" json:"createBy"`
	UpdateBy string `gorm:"column:update_by;size:64;default:''" json:"updateBy"`
}
