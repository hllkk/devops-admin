package system

import "github.com/hllkk/devops-admin/server/global"

// SysMenu 菜单(树形自关联,对外业务实体,字段对齐前端 Api.System.Menu / RuoYi 契约)
//
// 设计要点:
//   - 嵌入 OPS_AUDIT_MODEL 获取 createTime/updateTime/createBy/updateBy(对齐前端 CommonRecord)
//   - 主键 MenuId 走雪花 int64(json menuId,string 对齐前端 IdType)
//   - menuType: M目录 C菜单 F按钮; isFrame: 0是1否2iframe; isCache/visible/status 走 '0'/'1'
//   - Children 树形子节点内存组装,不建列; ParentName 内存组装
type SysMenu struct {
	global.OPS_AUDIT_MODEL
	MenuId     int64     `json:"menuId,string" gorm:"primarykey;comment:菜单ID"`             // 菜单ID
	ParentId   int64     `json:"parentId,string" gorm:"default:0;comment:父菜单ID"`           // 父菜单ID(0为顶级)
	MenuName   string    `json:"menuName" gorm:"index;comment:菜单名称"`                       // 菜单名称
	OrderNum   int       `json:"orderNum" gorm:"default:0;comment:显示顺序"`                   // 显示顺序
	Path       string    `json:"path" gorm:"comment:路由地址"`                                 // 路由地址
	Component  string    `json:"component" gorm:"comment:组件路径"`                            // 组件路径
	QueryParam string    `json:"queryParam" gorm:"comment:路由参数"`                           // 路由参数
	IsFrame    string    `json:"isFrame" gorm:"default:1;size:1;comment:是否外链 0是1否2iframe"` // 是否外链
	IsCache    string    `json:"isCache" gorm:"default:0;size:1;comment:是否缓存 0缓存1不缓存"`     // 是否缓存
	MenuType   string    `json:"menuType" gorm:"default:C;size:1;comment:菜单类型 M目录C菜单F按钮"`  // 菜单类型
	Visible    string    `json:"visible" gorm:"default:0;size:1;comment:显示状态 0显示1隐藏"`      // 显示状态
	Status     string    `json:"status" gorm:"default:0;size:1;comment:菜单状态 0正常1停用"`       // 菜单状态(对齐前端 '0'/'1')
	Perms      string    `json:"perms" gorm:"comment:权限标识"`                                // 权限标识
	Icon       string    `json:"icon" gorm:"comment:菜单图标"`                                 // 菜单图标
	Remark     string    `json:"remark" gorm:"comment:备注"`                                 // 备注
	ParentName string    `json:"parentName" gorm:"-"`                                      // 父菜单名称(内存组装,不建列)
	Children   []SysMenu `json:"children" gorm:"-"`                                        // 子菜单(内存组装,不建列)
}

func (SysMenu) TableName() string {
	return "sys_menu"
}
