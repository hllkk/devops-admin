package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// NoticeSearch 通知公告分页查询(对齐前端 Api.System.NoticeSearchParams,GET query 传输)
// noticeTitle 模糊匹配;noticeType 精确匹配('1'通知 '2'公告)。
type NoticeSearch struct {
	commonReq.PageInfo
	NoticeTitle string `json:"noticeTitle" form:"noticeTitle"` // 公告标题(模糊匹配)
	NoticeType  string `json:"noticeType" form:"noticeType"`   // 公告类型(精确 '1'通知/'2'公告)
}

// NoticeOperateParams 通知公告新增/修改请求(对齐前端 Api.System.NoticeOperateParams)
// create 时 noticeId 为空(主键走 DB 自增);update 时必填 noticeId。
type NoticeOperateParams struct {
	NoticeId      int64  `json:"noticeId,string" form:"noticeId"`    // 公告ID(新增时为空)
	NoticeTitle   string `json:"noticeTitle" form:"noticeTitle"`     // 公告标题
	NoticeType    string `json:"noticeType" form:"noticeType"`       // 公告类型('1'通知 '2'公告)
	NoticeContent string `json:"noticeContent" form:"noticeContent"` // 公告内容
	Status        string `json:"status" form:"status"`               // 公告状态('0'正常 '1'停用)
}
