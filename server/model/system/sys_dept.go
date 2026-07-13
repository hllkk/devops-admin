package system

import "github.com/hllkk/devops-admin/server/global"

// SysDept 系统部门，对齐前端 Api.System.Dept。
// 树形结构（parentId，根=0）；ancestors 记录祖级链（如 "0,100,101"），便于权限继承与子树查询快速定位。
//
//	status: 0=正常 1=停用
//	注：前端 Dept 类型残留 tenantId 字段，多租户已于 2026-07-12 清理（见 demand-index），
//	后端不建模 tenantId，与 SysUser/SysRole 保持一致。
//	负责人 leader 为用户 userId，按主键契约以字符串传输（json:",string"）。
type SysDept struct {
	DeptId       int64     `gorm:"column:dept_id;primaryKey;autoIncrement:false" json:"deptId,string"`
	ParentId     int64     `gorm:"column:parent_id;default:0" json:"parentId,string"`
	Ancestors    string    `gorm:"column:ancestors;size:500;default:''" json:"ancestors"`
	DeptName     string    `gorm:"column:dept_name;size:64" json:"deptName"`
	DeptCategory string    `gorm:"column:dept_category;size:64;default:''" json:"deptCategory"`
	OrderNum     int       `gorm:"column:order_num;default:0" json:"orderNum"`
	Leader       int64     `gorm:"column:leader;default:0" json:"leader,string"` // 负责人 userId（雪花 ID 字符串传输，0=未指定）
	Phone        string    `gorm:"column:phone;size:20;default:''" json:"phone"`
	Email        string    `gorm:"column:email;size:128;default:''" json:"email"`
	Status       string    `gorm:"column:status;size:1;default:'0'" json:"status"` // 0正常 1停用
	Children     []SysDept `gorm:"-" json:"children"`                              // 瞬态：子部门，树构建/treeselect 使用，不入库
	global.OPS_AUDIT_MODEL
}

// TableName 自定义表名 sys_dept
func (SysDept) TableName() string { return "sys_dept" }
