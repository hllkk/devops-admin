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
// 供尚未改造为"业务命名主键"的内部系统表(日志/黑名单/单行配置等)继续使用,
// 避免一次性全表重构。对外业务实体应改用 OPS_AUDIT_MODEL 并自定义主键。
type OPS_MODEL struct {
	OPS_BASE
	ID uint `gorm:"primarykey" json:"id,string"` // 主键(过渡保留,未改造系统表用)
}

// OPS_AUDIT_MODEL 审计基座:内嵌 OPS_BASE + CreateBy/UpdateBy,
// 用于对齐 RuoYi/前端 CommonRecord 的对外业务实体(SysUser/SysRole/菜单…)。
// 不含主键,主键由业务模型自定义。
// CreateBy/UpdateBy 的列(create_by/update_by)同时供数据权限引擎盖章消费。
type OPS_AUDIT_MODEL struct {
	OPS_BASE
	CreateBy uint `json:"createBy,string" gorm:"comment:创建者"`
	UpdateBy uint `json:"updateBy,string" gorm:"comment:更新者"`
}
