package system

// SysRoleMenu 是 sysRole 和 sysMenu 的连接表(角色菜单权限,前端 RoleOperateParams.menuIds)
type SysRoleMenu struct {
	SysRoleId int64 `gorm:"column:sys_role_id;index"`
	SysMenuId int64 `gorm:"column:sys_menu_id;index"`
}

func (s *SysRoleMenu) TableName() string {
	return "sys_role_menu"
}
