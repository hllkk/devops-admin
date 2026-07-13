package request

// SysMenuSearch 菜单列表查询，对齐前端 Api.System.MenuSearchParams。
// 菜单为树形结构，列表接口返回扁平数组（前端用 handleTree 构树），故不内嵌 PageInfo。
type SysMenuSearch struct {
	MenuName      string `json:"menuName" form:"menuName"`
	Status        string `json:"status" form:"status"`
	MenuType      string `json:"menuType" form:"menuType"`
	ParentId      string `json:"parentId" form:"parentId"`
	OrderByColumn string `json:"orderByColumn" form:"orderByColumn"`
	IsAsc         string `json:"isAsc" form:"isAsc"`
}

// SysMenuReq 菜单新增/修改，对齐前端 Api.System.MenuOperateParams。
// 主键以字符串传输（雪花 ID 精度），用指针区分「未指定(新增)」与「已指定(修改)」。
type SysMenuReq struct {
	MenuId     *string `json:"menuId" form:"menuId"`
	ParentId   *string `json:"parentId" form:"parentId"`
	MenuName   string  `json:"menuName" form:"menuName"`
	OrderNum   int     `json:"orderNum" form:"orderNum"`
	Path       string  `json:"path" form:"path"`
	Component  string  `json:"component" form:"component"`
	QueryParam string  `json:"queryParam" form:"queryParam"`
	IsFrame    string  `json:"isFrame" form:"isFrame"`
	IsCache    string  `json:"isCache" form:"isCache"`
	MenuType   string  `json:"menuType" form:"menuType"`
	Visible    string  `json:"visible" form:"visible"`
	Status     string  `json:"status" form:"status"`
	Perms      string  `json:"perms" form:"perms"`
	Icon       string  `json:"icon" form:"icon"`
	Remark     string  `json:"remark" form:"remark"`
}
