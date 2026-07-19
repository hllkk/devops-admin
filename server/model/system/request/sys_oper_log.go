package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// OperLogSearch 操作日志分页查询(对齐前端 Api.Log.OperLogSearchParams,GET query 传输)
// title/operName/operIp 模糊匹配;businessType/status 精确匹配;BeginTime/EndTime 为 operTime 时间范围。
//
// 时间范围说明:前端 qs.stringify 将 params:{beginTime,endTime} 序列化为 params[beginTime]/params[endTime](bracket),
// gin 的 struct binding 不支持 bracket 嵌套映射, 故 BeginTime/EndTime 标 form:"-",
// 由 api 层用 c.Query("params[beginTime]") / c.Query("params[endTime]") 显式取后赋值。
type OperLogSearch struct {
	commonReq.PageInfo
	Title        string `json:"title" form:"title"`               // 系统模块(模糊匹配)
	BusinessType string `json:"businessType" form:"businessType"` // 操作类型(精确 '0'~'9')
	OperName     string `json:"operName" form:"operName"`         // 操作人员(模糊匹配)
	OperIp       string `json:"operIp" form:"operIp"`             // 操作IP(模糊匹配)
	Status       string `json:"status" form:"status"`             // 操作状态(精确 '0'正常/'1'异常)
	BeginTime    string `json:"beginTime" form:"-"`               // 起始时间(api 层从 params[beginTime] 显式取)
	EndTime      string `json:"endTime" form:"-"`                 // 结束时间(api 层从 params[endTime] 显式取)
}
