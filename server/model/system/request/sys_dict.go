package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// DictTypeSearch 字典类型分页查询(对齐前端 Api.System.DictTypeSearchParams,GET query 传输)
type DictTypeSearch struct {
	commonReq.PageInfo
	DictName string `json:"dictName" form:"dictName"` // 字典名称(模糊匹配)
	DictType string `json:"dictType" form:"dictType"` // 字典类型(模糊匹配)
}

// DictTypeOperateParams 字典类型新增/修改请求(对齐前端 Api.System.DictTypeOperateParams)
// create 时 dictId 为空(主键走 DB 自增);update 时必填 dictId。
type DictTypeOperateParams struct {
	DictId   int64  `json:"dictId,string" form:"dictId"` // 字典主键(新增时为空)
	DictName string `json:"dictName" form:"dictName"`    // 字典名称
	DictType string `json:"dictType" form:"dictType"`    // 字典类型(唯一)
	Remark   string `json:"remark" form:"remark"`        // 备注
}

// DictDataSearch 字典数据分页查询(对齐前端 Api.System.DictDataSearchParams,GET query 传输)
type DictDataSearch struct {
	commonReq.PageInfo
	DictLabel string `json:"dictLabel" form:"dictLabel"` // 字典标签(模糊匹配)
	DictType  string `json:"dictType" form:"dictType"`   // 字典类型(精确匹配,页面右侧按选定 type 过滤)
}

// DictDataOperateParams 字典数据新增/修改请求(对齐前端 Api.System.DictDataOperateParams)
// create 时 dictCode 为空(主键走 DB 自增);update 时必填 dictCode。前端 operate 不含 isI18n/i18nKey。
type DictDataOperateParams struct {
	DictCode  int64  `json:"dictCode,string" form:"dictCode"` // 字典编码(新增时为空)
	DictSort  int    `json:"dictSort" form:"dictSort"`         // 字典排序
	DictLabel string `json:"dictLabel" form:"dictLabel"`       // 字典标签
	DictValue string `json:"dictValue" form:"dictValue"`       // 字典键值
	DictType  string `json:"dictType" form:"dictType"`         // 字典类型(关联 sys_dict_type.dict_type)
	CssClass  string `json:"cssClass" form:"cssClass"`         // 样式属性
	ListClass string `json:"listClass" form:"listClass"`       // 表格回显样式(NaiveUI ThemeColor)
	IsDefault string `json:"isDefault" form:"isDefault"`       // 是否默认(Y/N)
	Remark    string `json:"remark" form:"remark"`             // 备注
}
