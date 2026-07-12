package request

import (
	"github.com/hllkk/devops-admin/server/model/common/request"
)

// SysRoleSearch 角色列表查询，对齐前端 Api.System.RoleSearchParams。
type SysRoleSearch struct {
	RoleName      string `json:"roleName" form:"roleName"`
	RoleKey       string `json:"roleKey" form:"roleKey"`
	Status        string `json:"status" form:"status"`
	OrderByColumn string `json:"orderByColumn" form:"orderByColumn"`
	IsAsc         string `json:"isAsc" form:"isAsc"`
	request.PageInfo
}

// SysRoleReq 角色新增/修改，对齐前端 Api.System.RoleOperateParams（create 时 roleId 为 nil）。
type SysRoleReq struct {
	RoleId            *string  `json:"roleId" form:"roleId"`
	RoleName          string   `json:"roleName" form:"roleName"`
	RoleKey           string   `json:"roleKey" form:"roleKey"`
	RoleSort          int      `json:"roleSort" form:"roleSort"`
	MenuCheckStrictly bool     `json:"menuCheckStrictly" form:"menuCheckStrictly"`
	Status            string   `json:"status" form:"status"`
	Remark            string   `json:"remark" form:"remark"`
	MenuIds           []string `json:"menuIds" form:"menuIds"`
}

// SysRoleAuthUserReq 角色分配/取消用户（userIds 逗号分隔，由 API 层切分）。
type SysRoleAuthUserReq struct {
	RoleId  string `json:"roleId" form:"roleId"`
	UserIds string `json:"userIds" form:"userIds"`
}
