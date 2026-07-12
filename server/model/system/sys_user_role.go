package system

// SysUserRole 用户-角色关联表（复合主键）。
// 插入时必须显式指定 UserId/RoleId；雪花回调仅在主键为 0 时填充，不覆盖显式值。
type SysUserRole struct {
	UserId int64 `gorm:"column:user_id;primaryKey;autoIncrement:false" json:"userId,string"`
	RoleId int64 `gorm:"column:role_id;primaryKey;autoIncrement:false" json:"roleId,string"`
}

// TableName 自定义表名 sys_user_role
func (SysUserRole) TableName() string { return "sys_user_role" }
