package system

import "github.com/hllkk/devops-admin/server/global"

// SysDictData 字典数据，对齐前端 Api.System.DictData。归属于某个字典类型（dict_type）。
//
//	isDefault: Y=是 N=否（对齐前端 Common.YesOrNoStatus）
//	listClass: 表格回显样式，取值 default/primary/info/success/warning/error（对齐 NaiveUI.ThemeColor）
//	isI18n/i18nKey: 多语言扩展，前端 DictDataOperateParams 未含，当前为预留只读字段
type SysDictData struct {
	DictCode  int64  `gorm:"column:dict_code;primaryKey;autoIncrement:false" json:"dictCode,string"`
	DictSort  int    `gorm:"column:dict_sort;default:0" json:"dictSort"`
	DictLabel string `gorm:"column:dict_label;size:100" json:"dictLabel"`
	DictValue string `gorm:"column:dict_value;size:100" json:"dictValue"`
	DictType  string `gorm:"column:dict_type;size:100;index:idx_dict_data_type" json:"dictType"`
	CssClass  string `gorm:"column:css_class;size:100;default:''" json:"cssClass"`
	ListClass string `gorm:"column:list_class;size:100;default:'default'" json:"listClass"`
	IsDefault string `gorm:"column:is_default;size:1;default:'N'" json:"isDefault"` // Y是 N否
	IsI18n    bool   `gorm:"column:is_i18n;default:false" json:"isI18n"`             // 是否多语言（预留）
	I18nKey   string `gorm:"column:i18n_key;size:255;default:''" json:"i18nKey"`    // 多语言标识（预留）
	Remark    string `gorm:"column:remark;size:500;default:''" json:"remark"`
	global.OPS_AUDIT_MODEL
}

// TableName 自定义表名 sys_dict_data
func (SysDictData) TableName() string { return "sys_dict_data" }
