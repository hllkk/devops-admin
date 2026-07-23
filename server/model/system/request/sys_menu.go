package request

import (
	"github.com/hllkk/devops-admin/server/model/common"
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// MenuSearch 菜单列表查询(对齐前端 Api.System.MenuSearchParams,GET query 传输)。
// 菜单为树形,列表不分页,全量返回平表由前端组装树;menuName 模糊,status/menuType/parentId 精确。
type MenuSearch struct {
	commonReq.PageInfo
	MenuName string `json:"menuName" form:"menuName"`             // 菜单名称(模糊匹配)
	Status   string `json:"status" form:"status"`                 // 菜单状态(精确 '0'正常/'1'停用)
	MenuType string `json:"menuType" form:"menuType"`             // 菜单类型(精确 M目录C菜单F按钮)
	ParentId int64  `json:"parentId,string" form:"parentId"`      // 父菜单ID(精确,查按钮列表用)
}

// MenuOperateParams 菜单新增/修改请求(对齐前端 Api.System.MenuOperateParams)。
// create 时 menuId 为空;update 时必填 menuId。menuId/parentId 用 common.Int64String 兼容前端 IdType(string|number)。
type MenuOperateParams struct {
	MenuId     common.Int64String `json:"menuId"`     // 菜单ID(新增时为空)
	ParentId   common.Int64String `json:"parentId"`   // 父菜单ID(0=顶级)
	MenuType   string      `json:"menuType"`   // 菜单类型 M目录C菜单F按钮
	MenuName   string      `json:"menuName"`   // 菜单名称
	OrderNum   int         `json:"orderNum"`   // 显示顺序
	Path       string      `json:"path"`       // 路由地址
	Component  string      `json:"component"`  // 组件路径
	QueryParam string      `json:"queryParam"` // 路由参数
	IsFrame    string      `json:"isFrame"`    // 是否外链 0是1否2iframe
	IsCache    string      `json:"isCache"`    // 是否缓存 0缓存1不缓存
	Visible    string      `json:"visible"`    // 显示状态 0显示1隐藏
	Status     string      `json:"status"`     // 菜单状态 0正常1停用
	Perms      string      `json:"perms"`      // 权限标识
	Icon       string      `json:"icon"`       // 菜单图标
	Remark     string      `json:"remark"`     // 备注
}
