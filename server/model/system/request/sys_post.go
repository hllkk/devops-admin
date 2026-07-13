package request

import (
	"github.com/hllkk/devops-admin/server/model/common/request"
)

// SysPostSearch 岗位列表查询，对齐前端 Api.System.PostSearchParams（分页）。
// belongDeptId 为左侧部门树点击下钻的归属部门过滤，deptId 为岗位自身的归属部门字段。
type SysPostSearch struct {
	PostCode      string `json:"postCode" form:"postCode"`
	PostName      string `json:"postName" form:"postName"`
	Status        string `json:"status" form:"status"`
	DeptId        string `json:"deptId" form:"deptId"`
	BelongDeptId  string `json:"belongDeptId" form:"belongDeptId"`
	OrderByColumn string `json:"orderByColumn" form:"orderByColumn"`
	IsAsc         string `json:"isAsc" form:"isAsc"`
	request.PageInfo
}

// SysPostReq 岗位新增/修改，对齐前端 Api.System.PostOperateParams。
// 主键以字符串传输（雪花 ID 精度），用指针区分「未指定(新增)」与「已指定(修改)」。
type SysPostReq struct {
	PostId       *string `json:"postId" form:"postId"`
	DeptId       *string `json:"deptId" form:"deptId"`
	PostCode     string  `json:"postCode" form:"postCode"`
	PostCategory string  `json:"postCategory" form:"postCategory"`
	PostName     string  `json:"postName" form:"postName"`
	PostSort     int     `json:"postSort" form:"postSort"`
	Status       string  `json:"status" form:"status"`
	Remark       string  `json:"remark" form:"remark"`
}
