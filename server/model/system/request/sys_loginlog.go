package request

import "github.com/hllkk/devops-admin/server/model/common/request"

// LoginLogSearch 登录日志分页查询，对齐前端 Api.Log.LoginLogSearchParams（userName/ipaddr/status + 分页）。
type LoginLogSearch struct {
	UserName string `json:"userName" form:"userName"`
	Ipaddr   string `json:"ipaddr" form:"ipaddr"`
	Status   string `json:"status" form:"status"` // "0" 成功 / "1" 失败
	request.PageInfo
}
