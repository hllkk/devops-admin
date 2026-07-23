package request

import "github.com/hllkk/devops-admin/server/model/common/request"

// OnlineSearch 在线设备列表查询(对齐前端 OnlineUserSearchParams)。
// 个人中心视角:仅返回当前登录用户自己的设备,userName 搜索参数忽略;
// ipaddr 模糊过滤 + 分页(走 PageInfo.LimitOffset,MaxPageSize=100 截断)。
type OnlineSearch struct {
	request.PageInfo
	Ipaddr string `json:"ipaddr" form:"ipaddr"` // 登录IP(模糊匹配)
}
