package system

// SysRoleDepartment 角色与部门的连接表(数据权限第5档"自定义部门集"的角色配置)
type SysRoleDepartment struct {
	SysRoleId       int64 `gorm:"column:sys_role_id;index"`
	SysDepartmentId int64 `gorm:"column:sys_department_id;index"`
}

func (s *SysRoleDepartment) TableName() string {
	return "sys_role_departments"
}
