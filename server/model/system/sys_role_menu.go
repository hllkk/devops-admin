package system

// SysRoleMenu 是 sysRole 和 sysMenu 的连接表(角色菜单权限,前端 RoleOperateParams.menuIds)
// 复合主键(role_id + menu_id),连接表无需雪花 ID
type SysRoleMenu struct {
	SysRoleId int64 `gorm:"column:sys_role_id;primaryKey;index"`
	SysMenuId int64 `gorm:"column:sys_menu_id;primaryKey;index"`
}

func (s *SysRoleMenu) TableName() string {
	return "sys_role_menu"
}
