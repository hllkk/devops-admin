package request

import (
	"github.com/hllkk/devops-admin/server/model/common/request"
)

// SysUserSearch 用户列表查询，对齐前端 Api.System.UserSearchParams。
// 注意：嵌入的 request.PageInfo 的 json tag 是 "page"（非 pageNum），
// 但列表接口走 GET query，按 form:"pageNum" 绑定，不受影响。
type SysUserSearch struct {
	DeptId        *string `json:"deptId" form:"deptId"`
	UserName      string  `json:"userName" form:"userName"`
	NickName      string  `json:"nickName" form:"nickName"`
	Phonenumber   string  `json:"phonenumber" form:"phonenumber"`
	Status        string  `json:"status" form:"status"`
	RoleId        *string `json:"roleId" form:"roleId"`
	OrderByColumn string  `json:"orderByColumn" form:"orderByColumn"`
	IsAsc         string  `json:"isAsc" form:"isAsc"`
	request.PageInfo
}

// SysUserReq 用户新增/修改，对齐前端 Api.System.UserOperateParams（create 时 userId 为 nil）。
type SysUserReq struct {
	UserId      *string  `json:"userId" form:"userId"`
	DeptId      *string  `json:"deptId" form:"deptId"`
	UserName    string   `json:"userName" form:"userName"`
	NickName    string   `json:"nickName" form:"nickName"`
	Email       string   `json:"email" form:"email"`
	Phonenumber string   `json:"phonenumber" form:"phonenumber"`
	Sex         string   `json:"sex" form:"sex"`
	Password    string   `json:"password" form:"password"`
	Status      string   `json:"status" form:"status"`
	Remark      string   `json:"remark" form:"remark"`
	RoleIds     []string `json:"roleIds" form:"roleIds"`
	PostIds     []string `json:"postIds" form:"postIds"`
}

// SysResetPwdReq 重置密码（前端带 isEncrypt 头，加解密在 service/api 层处理）。
type SysResetPwdReq struct {
	UserId   string `json:"userId" form:"userId"`
	Password string `json:"password" form:"password"`
}

// SysUserStatusReq 修改帐号状态。
type SysUserStatusReq struct {
	UserId string `json:"userId" form:"userId"`
	Status string `json:"status" form:"status"`
}
