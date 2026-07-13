package system

import "github.com/hllkk/devops-admin/server/global"

// SysMenu 系统菜单，对齐前端 Api.System.Menu。树形结构（parentId，根=0）。
//
//	menuType: M=目录 C=菜单 F=按钮
//	isFrame : 0=外链 1=内部 2=iframe
//	isCache : 0=缓存 1=不缓存
//	visible : 0=显示 1=隐藏
//	status  : 0=正常 1=停用
type SysMenu struct {
	MenuId     int64  `gorm:"column:menu_id;primaryKey;autoIncrement:false" json:"menuId,string"`
	MenuName   string `gorm:"column:menu_name;size:64" json:"menuName"`
	ParentId   int64  `gorm:"column:parent_id;default:0" json:"parentId,string"`
	OrderNum   int    `gorm:"column:order_num;default:0" json:"orderNum"`
	Path       string `gorm:"column:path;size:200;default:''" json:"path"`
	Component  string `gorm:"column:component;size:255;default:''" json:"component"`
	QueryParam string `gorm:"column:query_param;size:255;default:''" json:"queryParam"`
	IsFrame    string `gorm:"column:is_frame;size:1;default:'1'" json:"isFrame"`  // 0外链 1内部 2iframe
	IsCache    string `gorm:"column:is_cache;size:1;default:'0'" json:"isCache"`  // 0缓存 1不缓存
	MenuType   string `gorm:"column:menu_type;size:1;default:''" json:"menuType"` // M目录 C菜单 F按钮
	Visible    string `gorm:"column:visible;size:1;default:'0'" json:"visible"`   // 0显示 1隐藏
	Status     string `gorm:"column:status;size:1;default:'0'" json:"status"`     // 0正常 1停用
	Perms      string `gorm:"column:perms;size:100;default:''" json:"perms"`
	Icon       string `gorm:"column:icon;size:100;default:''" json:"icon"`
	Remark     string `gorm:"column:remark;size:500;default:''" json:"remark"`
	ParentName string    `gorm:"-" json:"parentName"` // 瞬态：父菜单名称，不入库
	Children   []SysMenu `gorm:"-" json:"children"`   // 瞬态：子菜单，树构建/treeselect 使用，不入库
	global.OPS_AUDIT_MODEL
}

// TableName 自定义表名 sys_menu
func (SysMenu) TableName() string { return "sys_menu" }
