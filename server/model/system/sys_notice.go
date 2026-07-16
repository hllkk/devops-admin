package system

import "github.com/hllkk/devops-admin/server/global"

// SysNotice 系统通知公告，对齐前端 Api.System.Notice。
//
//	noticeType: 1=通知 2=公告（对齐前端 System.NoticeType = '1' | '2'）
//	noticeContent: 公告内容，富文本可能较大，用 text 存储
//	status: 0=正常 1=停用（对齐前端 Common.EnableStatus）
//	注：前端 Notice.createByName 为只读展示字段，由审计基座 createBy 承载，不单独建列；
//	    前端 CommonRecord 残留 createDept（部门数据权限），后端基座未建，与 SysUser/SysRole/SysPost/SysDict 一致。
type SysNotice struct {
	NoticeId      int64  `gorm:"column:notice_id;primaryKey;autoIncrement:false" json:"noticeId,string"`
	NoticeTitle   string `gorm:"column:notice_title;size:100" json:"noticeTitle"`
	NoticeType    string `gorm:"column:notice_type;size:1;default:'1'" json:"noticeType"` // 1通知 2公告
	NoticeContent string `gorm:"column:notice_content;type:text" json:"noticeContent"`
	Status        string `gorm:"column:status;size:1;default:'0'" json:"status"` // 0正常 1停用
	Remark        string `gorm:"column:remark;size:500;default:''" json:"remark"`
	global.OPS_AUDIT_MODEL
}

// TableName 自定义表名 sys_notice
func (SysNotice) TableName() string { return "sys_notice" }
