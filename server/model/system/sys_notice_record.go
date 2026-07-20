package system

import (
	"time"

	"github.com/hllkk/devops-admin/server/global"
)

// SysNoticeRecord 通知接收记录:定向通知的已读跟踪。
// 创建定向通知时按目标用户预生成一行(read_at 为空=未读),用户阅读后置 read_at。
// 公告(广播全员)不入此表——公告历史回看走 sys_notice 主表的管理页。
// (notice_id, user_id) 复合唯一,防同一用户对同一通知重复落记录。
type SysNoticeRecord struct {
	global.OPS_MODEL
	NoticeId int64      `json:"noticeId,string" gorm:"uniqueIndex:idx_notice_user;comment:通知ID"`
	UserId   int64      `json:"userId,string" gorm:"uniqueIndex:idx_notice_user;comment:接收用户ID"`
	ReadAt   *time.Time `json:"readAt,omitempty" gorm:"comment:阅读时间(空=未读)"`
}

func (SysNoticeRecord) TableName() string {
	return "sys_notice_record"
}
