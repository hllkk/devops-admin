package system

// SysRoleMenu 角色-菜单关联表（复合主键）。同 SysUserRole，插入须显式指定两个 ID。
type SysRoleMenu struct {
	RoleId int64 `gorm:"column:role_id;primaryKey;autoIncrement:false" json:"roleId,string"`
	MenuId int64 `gorm:"column:menu_id;primaryKey;autoIncrement:false" json:"menuId,string"`
}

// TableName 自定义表名 sys_role_menu
func (SysRoleMenu) TableName() string { return "sys_role_menu" }
