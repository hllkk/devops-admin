package global

import (
	"time"

	"gorm.io/gorm"
)

// OPS_BASE 生命周期基座:所有表通用,不含主键。
// 主键由各模型自定义(对外业务实体用业务命名 userId/roleId…,统一雪花 int64;
// 内部表用 id),回调 ops:snowflake_id 按 PrioritizedPrimaryField 自动填充(待落地)。
// json 对齐前端 CommonRecord 的 createTime/updateTime。
type OPS_BASE struct {
	CreateTime time.Time      `json:"createTime" gorm:"comment:创建时间"`
	UpdateTime time.Time      `json:"updateTime" gorm:"comment:更新时间"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// OPS_MODEL 过渡基座:OPS_BASE + ID 主键。
type OPS_MODEL struct {
	OPS_BASE
	ID uint `gorm:"primarykey" json:"id,string"` // 主键(过渡保留,未改造系统表用)
}

// OPS_AUDIT_MODEL 审计基座:内嵌 OPS_BASE + CreateBy/UpdateBy,
// CreateBy/UpdateBy 从 uint 升级为 int64(对齐雪花 ID 主键类型,json string 传输对齐前端)
type OPS_AUDIT_MODEL struct {
	OPS_BASE
	CreateBy int64 `json:"createBy,string" gorm:"comment:创建者"`
	UpdateBy int64 `json:"updateBy,string" gorm:"comment:更新者"`
}
