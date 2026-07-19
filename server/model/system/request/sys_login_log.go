package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// LoginLogSearch 登录日志分页查询(对齐前端 Api.Log.LoginLogSearchParams,GET query 传输)
// userName/ipaddr 模糊匹配;status 精确匹配('0'成功 '1'失败)。
type LoginLogSearch struct {
	commonReq.PageInfo
	UserName string `json:"userName" form:"userName"` // 用户账号(模糊匹配)
	Ipaddr   string `json:"ipaddr" form:"ipaddr"`     // 登录IP(模糊匹配)
	Status   string `json:"status" form:"status"`     // 登录状态(精确 '0'成功/'1'失败)
}
