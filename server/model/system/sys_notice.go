package system

import "github.com/hllkk/devops-admin/server/global"

// SysNotice 通知公告(对外业务实体,字段对齐前端 Api.System.Notice)
type SysNotice struct {
	global.OPS_AUDIT_MODEL
	NoticeId      int64  `json:"noticeId,string" gorm:"primarykey;comment:公告ID"`         // 公告ID
	NoticeTitle   string `json:"noticeTitle" gorm:"index;comment:公告标题"`                  // 公告标题
	NoticeType    string `json:"noticeType" gorm:"default:1;size:1;comment:公告类型 1通知2公告"` // 公告类型(对齐前端 NoticeType '1'|'2')
	NoticeContent string `json:"noticeContent" gorm:"type:text;comment:公告内容"`            // 公告内容
	Status        string `json:"status" gorm:"default:0;size:1;comment:公告状态 0正常1停用"`     // 公告状态(对齐前端 '0'/'1')
	Remark        string `json:"remark" gorm:"comment:备注"`                               // 备注
	CreateByName  string `json:"createByName" gorm:"-"`                                  // 创建者名称(内存组装,join sys_users 带出)
}

// 通知公告类型(对齐字典 sys_notice_type)
const (
	NoticeTypeNotice       = "1" // 通知(定向投递)
	NoticeTypeAnnouncement = "2" // 公告(全员广播)
)

func (SysNotice) TableName() string {
	return "sys_notice"
}
