package system

import "github.com/hllkk/devops-admin/server/global"

// SysDictType 字典类型，对齐前端 Api.System.DictType。
// 字典数据（SysDictData）通过 dict_type 关联到本表；dict_type 为业务唯一键，
// 也是接口 /system/dict/data/type/{dictType} 的查询依据。
//
// 注：前端 CommonRecord 残留 createDept（部门数据权限），后端基座未建，与 SysUser/SysRole 一致。
type SysDictType struct {
	DictId   int64  `gorm:"column:dict_id;primaryKey;autoIncrement:false" json:"dictId,string"`
	DictName string `gorm:"column:dict_name;size:100" json:"dictName"`
	DictType string `gorm:"column:dict_type;size:100;uniqueIndex:uk_dict_type" json:"dictType"`
	Remark   string `gorm:"column:remark;size:500;default:''" json:"remark"`
	global.OPS_AUDIT_MODEL
}

// TableName 自定义表名 sys_dict_type
func (SysDictType) TableName() string { return "sys_dict_type" }
