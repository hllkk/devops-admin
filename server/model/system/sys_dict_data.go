package system

import "github.com/hllkk/devops-admin/server/global"

// SysDictData 字典数据(对外业务实体,字段对齐前端 Api.System.DictData)
type SysDictData struct {
	global.OPS_AUDIT_MODEL
	DictCode  int64  `json:"dictCode,string" gorm:"primarykey;comment:字典编码"`           // 字典编码
	DictSort  int    `json:"dictSort" gorm:"default:0;comment:字典排序"`                // 字典排序
	DictLabel string `json:"dictLabel" gorm:"comment:字典标签"`                       // 字典标签
	DictValue string `json:"dictValue" gorm:"comment:字典键值"`                       // 字典键值
	DictType  string `json:"dictType" gorm:"index;comment:字典类型"`                   // 字典类型(关联 sys_dict_type.dict_type)
	CssClass  string `json:"cssClass" gorm:"comment:样式属性"`                        // 样式属性(其他样式扩展)
	ListClass string `json:"listClass" gorm:"comment:表格回显样式"`                     // 表格回显样式(NaiveUI ThemeColor)
	IsDefault string `json:"isDefault" gorm:"default:N;size:1;comment:是否默认 Y是N否"` // 是否默认(对齐前端 YesOrNoStatus)
	Remark    string `json:"remark" gorm:"comment:备注"`                            // 备注
	IsI18n    *bool  `json:"isI18n" gorm:"comment:是否多语言"`                       // 是否多语言(可空)
	I18nKey   string `json:"i18nKey" gorm:"comment:多语言标识"`                       // 多语言标识
}

func (SysDictData) TableName() string {
	return "sys_dict_data"
}
