package system

import "github.com/hllkk/devops-admin/server/global"

// SysPost 岗位(平表,不进 Casbin,仅作组织身份/职务,与角色正交;字段对齐前端 Api.System.Post)
// 注:岗位通过 deptId 归属部门,用户多岗位走 sys_user_post 连接表。
type SysPost struct {
	global.OPS_AUDIT_MODEL
	PostId       int64  `json:"postId,string" gorm:"primarykey;comment:岗位ID"`       // 岗位ID
	DeptId       int64  `json:"deptId,string" gorm:"index;comment:部门ID"`            // 部门ID(岗位归属部门)
	PostCode     string `json:"postCode" gorm:"index;comment:岗位编码"`                 // 岗位编码
	PostCategory string `json:"postCategory" gorm:"comment:岗位类别编码"`                 // 岗位类别编码
	PostName     string `json:"postName" gorm:"index;comment:岗位名称"`                 // 岗位名称
	PostSort     int    `json:"postSort" gorm:"default:0;comment:显示顺序"`             // 显示顺序
	Status       string `json:"status" gorm:"default:0;size:1;comment:岗位状态 0正常1停用"` // 岗位状态(对齐前端 '0'/'1')
	Remark       string `json:"remark" gorm:"comment:备注"`                           // 备注
}

func (SysPost) TableName() string {
	return "sys_posts"
}

