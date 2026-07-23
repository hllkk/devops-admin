package request

import (
	"github.com/hllkk/devops-admin/server/model/common"
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// DeptSearch 部门列表查询(对齐前端 Api.System.DeptSearchParams,GET query 传输)。
// 部门为树形,列表不分页,全量返回平表由前端组装树;内嵌 PageInfo 仅为兼容前端可能下发的分页参数(忽略不用)。
type DeptSearch struct {
	commonReq.PageInfo
	DeptName string `json:"deptName" form:"deptName"` // 部门名称(模糊匹配)
	Status   string `json:"status" form:"status"`     // 部门状态(精确 '0'正常/'1'停用)
}

// DeptOperateParams 部门新增/修改请求(对齐前端 Api.System.DeptOperateParams)。
// create 时 deptId 为空;update 时必填 deptId。ancestors 由后端维护(不入参)。
// deptId/parentId 用 common.Int64String 兼容前端空串与 IdType 混合;leader 用 int64(前端 null→0)。
type DeptOperateParams struct {
	DeptId       common.Int64String `json:"deptId"`       // 部门ID(新增时为空)
	ParentId     common.Int64String `json:"parentId"`     // 父部门ID(空/0=顶层)
	DeptName     string      `json:"deptName"`     // 部门名称
	DeptCategory string      `json:"deptCategory"` // 部门类别编码
	OrderNum     int         `json:"orderNum"`     // 显示顺序
	Leader       int64       `json:"leader"`       // 负责人用户ID(可空)
	Phone        string      `json:"phone"`        // 联系电话
	Email        string      `json:"email"`        // 邮箱
	Status       string      `json:"status"`       // 部门状态('0'正常/'1'停用)
}
