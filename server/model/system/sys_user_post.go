package system

// SysUserPost 是 sysUser 和 sysPost 的连接表(用户多岗位,前端 UserOperateParams.postIds)
// 由 GORM 依据 SysUser.Posts 的 many2many tag 自动建表,此显式 struct 仅供 service 层直接操作 join 表
type SysUserPost struct {
	SysUserId int64 `gorm:"column:sys_user_id;index"`
	SysPostId int64 `gorm:"column:sys_post_id;index"`
}

func (s *SysUserPost) TableName() string {
	return "sys_user_post"
}
