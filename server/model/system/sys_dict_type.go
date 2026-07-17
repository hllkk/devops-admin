package system

import "github.com/hllkk/devops-admin/server/global"

// SysDictType 字典类型(对外业务实体,字段对齐前端 Api.System.DictType)
type SysDictType struct {
	global.OPS_AUDIT_MODEL
	DictId   int64  `json:"dictId,string" gorm:"primarykey;comment:字典主键"`   // 字典主键
	DictName string `json:"dictName" gorm:"index;comment:字典名称"`             // 字典名称
	DictType string `json:"dictType" gorm:"uniqueIndex;comment:字典类型"`        // 字典类型(唯一)
	Remark   string `json:"remark" gorm:"comment:备注"`                       // 备注
}

func (SysDictType) TableName() string {
	return "sys_dict_type"
}
