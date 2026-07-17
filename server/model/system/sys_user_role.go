package system

// SysUserRole 是 sysUser 和 sysRole 的连接表(用户多角色,前端 UserOperateParams.roleIds)
type SysUserRole struct {
	SysUserId int64 `gorm:"column:sys_user_id;index"`
	SysRoleId int64 `gorm:"column:sys_role_id;index"`
}

func (s *SysUserRole) TableName() string {
	return "sys_user_role"
}
