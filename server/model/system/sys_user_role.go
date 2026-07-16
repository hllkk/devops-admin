package system

// SysUserRole 是 sysUser 和 sysRole 的连接表
type SysUserRole struct {
	SysUserId uint `gorm:"column:sys_user_id"`
	SysRoleId uint `gorm:"column:sys_role_id"`
}

func (s *SysUserRole) TableName() string {
	return "sys_user_role"
}
