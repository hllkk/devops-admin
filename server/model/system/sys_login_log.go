package system

import (
	"time"

	"github.com/hllkk/devops-admin/server/global"
)

// SysLoginLog 登录日志(对外业务实体,字段对齐前端 Api.Log.LoginLog)
//
// 设计要点:
//   - 嵌入 OPS_AUDIT_MODEL 获取 createBy/createTime/updateBy/updateTime(对齐前端 CommonRecord)
//   - 主键 InfoId 走雪花 int64(json infoId,string); 表名 sys_login_log
//   - status 对齐前端 EnableStatus '0'/'1'(0成功 1失败)
//   - deviceType 对齐前端 System.DeviceType pc/android/ios/xcx
//   - loginTime 为访问时间(业务字段,区别于记录写入时间 createTime)
type SysLoginLog struct {
	global.OPS_AUDIT_MODEL
	InfoId        int64     `json:"infoId,string" gorm:"primarykey;comment:访问ID"`             // 访问ID
	UserName      string    `json:"userName" gorm:"index;comment:用户账号"`                       // 用户账号
	ClientKey     string    `json:"clientKey" gorm:"comment:客户端"`                             // 客户端
	DeviceType    string    `json:"deviceType" gorm:"size:8;comment:设备类型 pc/android/ios/xcx"` // 设备类型(对齐前端 System.DeviceType)
	Ipaddr        string    `json:"ipaddr" gorm:"comment:登录IP地址"`                             // 登录IP地址
	LoginLocation string    `json:"loginLocation" gorm:"comment:登录地点"`                        // 登录地点
	Browser       string    `json:"browser" gorm:"comment:浏览器类型"`                             // 浏览器类型
	Os            string    `json:"os" gorm:"comment:操作系统"`                                   // 操作系统
	Status        string    `json:"status" gorm:"size:1;comment:登录状态 0成功1失败"`                 // 登录状态(对齐前端 EnableStatus '0'/'1')
	Msg           string    `json:"msg" gorm:"comment:提示消息"`                                  // 提示消息
	LoginTime     time.Time `json:"loginTime" gorm:"comment:访问时间"`                            // 访问时间(业务时间,区别于 createTime)
}

func (SysLoginLog) TableName() string {
	return "sys_login_log"
}
