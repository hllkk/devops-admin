package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// PostSearch 岗位分页查询(对齐前端 Api.System.PostSearchParams,GET query 传输)
// postCode/postName 模糊匹配;status/deptId/belongDeptId 精确;
// belongDeptId 为左侧部门树点击下发的归属部门过滤(对齐该部门直接挂载的岗位)。
type PostSearch struct {
	commonReq.PageInfo
	PostCode     string `json:"postCode" form:"postCode"`                 // 岗位编码(模糊匹配)
	PostName     string `json:"postName" form:"postName"`                 // 岗位名称(模糊匹配)
	Status       string `json:"status" form:"status"`                     // 岗位状态(精确 '0'正常/'1'停用)
	DeptId       int64  `json:"deptId,string" form:"deptId"`              // 部门ID(精确,直挂该部门)
	BelongDeptId int64  `json:"belongDeptId,string" form:"belongDeptId"`  // 归属部门ID(左侧部门树点击过滤,精确匹配 dept_id)
}

// PostOperateParams 岗位新增/修改请求(对齐前端 Api.System.PostOperateParams)
// create 时 postId 为空(主键走 DB 自增);update 时必填 postId。deptId 必填(归属部门)。
type PostOperateParams struct {
	PostId       int64  `json:"postId,string" form:"postId"`         // 岗位ID(新增时为空)
	DeptId       int64  `json:"deptId,string" form:"deptId"`         // 部门ID(必填,归属部门)
	PostCode     string `json:"postCode" form:"postCode"`            // 岗位编码(唯一)
	PostCategory string `json:"postCategory" form:"postCategory"`    // 岗位类别编码
	PostName     string `json:"postName" form:"postName"`            // 岗位名称
	PostSort     int    `json:"postSort" form:"postSort"`            // 显示顺序
	Status       string `json:"status" form:"status"`                // 岗位状态('0'正常/'1'停用)
	Remark       string `json:"remark" form:"remark"`                // 备注
}
