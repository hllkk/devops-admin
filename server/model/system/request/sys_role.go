package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// RoleSearch 角色分页查询(对齐前端 Api.System.RoleSearchParams,GET query 传输)。
type RoleSearch struct {
	commonReq.PageInfo
	RoleName string `json:"roleName" form:"roleName"` // 角色名称(模糊匹配)
	RoleKey  string `json:"roleKey" form:"roleKey"`   // 角色权限字符(模糊匹配)
	Status   string `json:"status" form:"status"`     // 角色状态(精确 '0'正常/'1'停用)
}

// RoleOperateParams 角色新增/修改请求(对齐前端 Api.System.RoleOperateParams)。
// 含 menuIds 用于分配菜单:create 时全量插入 sys_role_role_menu,update 时全量替换。
// menuIds 用 []Int64String 兼容前端 IdType[](string|number 混合,getCheckedMenuIds 返回 string[])。
type RoleOperateParams struct {
	RoleId            Int64String   `json:"roleId"`            // 角色ID(新增时为空)
	RoleName          string        `json:"roleName"`          // 角色名称
	RoleKey           string        `json:"roleKey"`           // 角色权限字符(唯一)
	RoleSort          int           `json:"roleSort"`          // 显示顺序
	MenuCheckStrictly bool          `json:"menuCheckStrictly"` // 菜单树选择项是否关联显示
	Status            string        `json:"status"`            // 角色状态('0'正常/'1'停用)
	Remark            string        `json:"remark"`            // 备注
	MenuIds           []Int64String `json:"menuIds"`           // 分配菜单ID(全量;叶子+半选父级)
	DataScope         int           `json:"dataScope,string"`  // 数据范围档位(1全部2本部门及子级3本部门4仅本人5自定义;仅 dataScope 接口消费)
	DeptIds           []Int64String `json:"deptIds"`            // 自定义部门集(仅档位5用;全量替换 sys_role_departments)
	DeptCheckStrictly bool          `json:"deptCheckStrictly"`  // 部门树选择项是否关联显示(仅 dataScope 接口消费)
}

// RoleUserSearch 角色已分配用户分页查询(对齐前端 GET /system/role/authUser/allocatedList,GET query 传输)。
type RoleUserSearch struct {
	commonReq.PageInfo
	RoleId      int64  `json:"roleId,string" form:"roleId"`   // 角色ID(必填)
	UserName    string `json:"userName" form:"userName"`      // 用户名(模糊匹配)
	Phonenumber string `json:"phonenumber" form:"phonenumber"` // 手机号(模糊匹配)
}

// RoleAuthUserParams 角色-用户授权操作请求(对齐前端 PUT authUser/selectAll|cancelAll,query 传参)。
// 前端以 params 传 { roleId, userIds: userIds.join(',') },userIds 为逗号分隔字符串。
type RoleAuthUserParams struct {
	RoleId  int64  `form:"roleId"`  // 角色ID
	UserIds string `form:"userIds"` // 用户ID列表(逗号分隔)
}
