package system

import (
	"time"

	"github.com/hllkk/devops-admin/server/global"
)

// SysLoginLog 登录日志（append-only 系统记录），对齐前端 Api.Log.LoginLog / RuoYi sys_loginlog。
// 主键 InfoId 由雪花回调 ops:snowflake_id 自动填充；生命周期走 global.OPS_MODEL
// （append-only，不含 CreateBy/UpdateBy 审计字段）。
type SysLoginLog struct {
	InfoId int64 `gorm:"column:info_id;primaryKey;autoIncrement:false" json:"infoId,string"`
	global.OPS_MODEL
	TenantId      int64      `gorm:"column:tenant_id;default:0" json:"tenantId,string"`                             // 租户编号
	UserName      string     `gorm:"column:user_name;size:64;index;comment:用户账号" json:"userName"`                   // 用户账号
	ClientKey     string     `gorm:"column:client_key;size:64;comment:客户端" json:"clientKey"`                        // 客户端
	DeviceType    string     `gorm:"column:device_type;size:16;comment:设备类型(pc/android/ios/xcx)" json:"deviceType"` // 设备类型
	Ipaddr        string     `gorm:"column:ipaddr;size:128;index;comment:登录IP地址" json:"ipaddr"`                     // 登录IP地址
	LoginLocation string     `gorm:"column:login_location;size:255;comment:登录地点" json:"loginLocation"`              // 登录地点
	Browser       string     `gorm:"column:browser;size:255;comment:浏览器类型" json:"browser"`                          // 浏览器类型
	Os            string     `gorm:"column:os;size:255;comment:操作系统" json:"os"`                                     // 操作系统
	Status        string     `gorm:"column:status;size:1;default:'0';index;comment:登录状态(0成功 1失败)" json:"status"`    // 登录状态
	Msg           string     `gorm:"column:msg;size:255;comment:提示消息" json:"msg"`                                   // 提示消息
	LoginTime     *time.Time `gorm:"column:login_time;index;comment:访问时间" json:"loginTime"`                         // 访问时间
}

// TableName 自定义表名 sys_loginlog
func (SysLoginLog) TableName() string { return "sys_loginlog" }
