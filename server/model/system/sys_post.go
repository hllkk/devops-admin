package system

import "github.com/hllkk/devops-admin/server/global"

// SysPost 系统岗位，对齐前端 Api.System.Post。归属于部门（deptId），与用户为多对多（关联表 sys_user_post 待建）。
//
//	status: 0=正常 1=停用
//	注：前端 Post 类型残留 tenantId，多租户已清理，后端不建模，与 SysUser/SysRole 一致。
type SysPost struct {
	PostId       int64  `gorm:"column:post_id;primaryKey;autoIncrement:false" json:"postId,string"`
	DeptId       int64  `gorm:"column:dept_id;default:0" json:"deptId,string"`
	PostCode     string `gorm:"column:post_code;size:64" json:"postCode"`
	PostCategory string `gorm:"column:post_category;size:64;default:''" json:"postCategory"`
	PostName     string `gorm:"column:post_name;size:64" json:"postName"`
	PostSort     int    `gorm:"column:post_sort;default:0" json:"postSort"`
	Status       string `gorm:"column:status;size:1;default:'0'" json:"status"` // 0正常 1停用
	Remark       string `gorm:"column:remark;size:500;default:''" json:"remark"`
	global.OPS_AUDIT_MODEL
}

// TableName 自定义表名 sys_post
func (SysPost) TableName() string { return "sys_post" }
