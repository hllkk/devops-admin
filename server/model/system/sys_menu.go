package system

import (
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/common"
)

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
	MenuType   string    `json:"menuType" gorm:"default:C;size:1;comment:菜单类型 M目录C菜单F按钮"`  // 菜单类型
	MenuName   string    `json:"menuName" gorm:"index;comment:菜单名称"`                       // 菜单名称
	OrderNum   int       `json:"orderNum" gorm:"default:0;comment:显示顺序"`                   // 显示顺序
	Path       string    `json:"path" gorm:"comment:路由地址"`                                 // 路由地址
	Component  string    `json:"component" gorm:"comment:组件路径"`                            // 组件路径
	Module     string    `json:"module,omitempty" gorm:"column:module;comment:业务模块归属(admin/disk/server/gateway)"` // 业务模块归属(dynamic 路由 meta.module 来源;空=老库未回填)
	QueryParam string    `json:"queryParam" gorm:"comment:路由参数"`                           // 路由参数
	IsFrame    string    `json:"isFrame" gorm:"default:1;size:1;comment:是否外链 0是1否2iframe"` // 是否外链
	IsCache    string    `json:"isCache" gorm:"default:0;size:1;comment:是否缓存 0缓存1不缓存"`     // 是否缓存
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

// MenuTreeSelectNode 菜单树选择节点(角色/菜单选择树专用,对齐前端 NTree key-field=id/label-field=label 与 RuoYi MenuTreeSelect)。
// 仅含树选择渲染所需字段,精简于 SysMenu(去 component/path/queryParam/isFrame/isCache/perms/remark/orderNum 等冗余);
// 已按 parent_id 组装 children 树,前端 NTree 直接消费。
type MenuTreeSelectNode struct {
	Id       int64                `json:"id,string"`  // 菜单ID(对齐前端 IdType)
	Label    string               `json:"label"`      // 菜单名称(NTree label-field;可能为 i18n key 如 route.xxx)
	MenuType string               `json:"menuType"`       // 菜单类型 M目录C菜单F按钮(前端渲染区分)
	Path     string               `json:"path,omitempty"` // 路由地址(C 菜单默认路由小房子推导 routeKey 用)
	Module   string               `json:"module,omitempty"` // 业务模块归属(admin/disk/server/gateway;角色授权树前端按模块分组用)
	Icon     string               `json:"icon"`           // 菜单图标
	Visible  string               `json:"visible"`    // 显示状态 0显示1隐藏(前端隐藏标灰)
	Status   string               `json:"status"`     // 菜单状态 0正常1停用(前端禁用标红)
	Children []MenuTreeSelectNode `json:"children,omitempty"` // 子节点(内存组装;叶子节点省略)
}

// RoleMenuTreeSelect 角色菜单权限树响应(对齐前端 Api.System.RoleMenuTreeSelect)。
// menus=全部菜单的 MenuTreeSelectNode 树(精简字段,后端组装);checkedKeys=角色已分配菜单的叶子 ID(string[];NTree cascade 回显用,对齐 RuoYi 并与 menus.id 统一 string)。
type RoleMenuTreeSelect struct {
	CheckedKeys common.Int64StringSlice `json:"checkedKeys"` // 角色已分配菜单的叶子 ID(string[],雪花 id 统一 string)
	Menus       []MenuTreeSelectNode    `json:"menus"`       // 全部菜单树(精简 VO,后端组装)
}
