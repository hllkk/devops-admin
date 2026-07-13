package request

// SysDeptSearch 部门列表查询，对齐前端 Api.System.DeptSearchParams。
// 部门为树形结构，列表接口返回扁平数组（前端用 handleTree 构树），故不内嵌 PageInfo（同 SysMenuSearch）。
type SysDeptSearch struct {
	DeptName      string `json:"deptName" form:"deptName"`
	Status        string `json:"status" form:"status"`
	OrderByColumn string `json:"orderByColumn" form:"orderByColumn"`
	IsAsc         string `json:"isAsc" form:"isAsc"`
}

// SysDeptReq 部门新增/修改，对齐前端 Api.System.DeptOperateParams。
// 主键以字符串传输（雪花 ID 精度），用指针区分「未指定(新增)」与「已指定(修改)」。
// ancestors 由 Service 层依据 parentId 的祖级链推导写入，不暴露给前端。
type SysDeptReq struct {
	DeptId       *string `json:"deptId" form:"deptId"`
	ParentId     *string `json:"parentId" form:"parentId"`
	DeptName     string  `json:"deptName" form:"deptName"`
	DeptCategory string  `json:"deptCategory" form:"deptCategory"`
	OrderNum     int     `json:"orderNum" form:"orderNum"`
	Leader       *string `json:"leader" form:"leader"`
	Phone        string  `json:"phone" form:"phone"`
	Email        string  `json:"email" form:"email"`
	Status       string  `json:"status" form:"status"`
}
