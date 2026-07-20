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
// targetType 控制定向投递:公告(type=2)强制 all(广播,不入 record);通知(type=1)按 users/depts 定向。
type NoticeOperateParams struct {
	NoticeId      int64   `json:"noticeId,string" form:"noticeId"`    // 公告ID(新增时为空)
	NoticeTitle   string  `json:"noticeTitle" form:"noticeTitle"`     // 公告标题
	NoticeType    string  `json:"noticeType" form:"noticeType"`       // 公告类型('1'通知 '2'公告)
	NoticeContent string  `json:"noticeContent" form:"noticeContent"` // 公告内容
	Status        string  `json:"status" form:"status"`              // 公告状态('0'正常 '1'停用)
	TargetType    string  `json:"targetType" form:"targetType"`      // 投递范围:all/users/depts(公告强制 all)
	TargetUserIds []int64 `json:"targetUserIds" form:"targetUserIds"` // 指定用户(targetType=users)
	TargetDeptIds []int64 `json:"targetDeptIds" form:"targetDeptIds"` // 指定部门含子部门(targetType=depts)
}

// NoticeUnreadSearch 当前用户通知列表查询(对齐前端未读/历史拉取,GET query)。
type NoticeUnreadSearch struct {
	commonReq.PageInfo
	OnlyUnread bool `json:"onlyUnread" form:"onlyUnread"` // 仅未读(read_at IS NULL)
}

// NoticeReadParams 标记已读(NoticeIds 为空表示当前用户全部已读)。
type NoticeReadParams struct {
	NoticeIds []int64 `json:"noticeIds"` // 通知ID列表(空=当前用户全部已读)
}
