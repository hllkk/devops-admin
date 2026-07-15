package system

import (
	"time"

	"github.com/hllkk/devops-admin/server/global"
)

// SysOperLog 操作日志（append-only 系统记录），对齐前端 Api.Log.OperLog / RuoYi sys_oper_log。
// 主键 OperId 由雪花回调 ops:snowflake_id 自动填充；生命周期走 global.OPS_MODEL
// （append-only，不含 CreateBy/UpdateBy 审计字段；操作人由 operName、操作时间由 operTime 承载）。
type SysOperLog struct {
	OperId int64 `gorm:"column:oper_id;primaryKey;autoIncrement:false" json:"operId,string"`
	global.OPS_MODEL
	Title         string     `gorm:"column:title;size:50;index;comment:系统模块" json:"title"`                      // 系统模块
	BusinessType  string     `gorm:"column:business_type;size:1;default:'0';index;comment:操作类型" json:"businessType"` // 0其它 1新增 2修改 3删除...
	Method        string     `gorm:"column:method;size:200;comment:方法名称" json:"method"`                          // 方法名称
	RequestMethod string     `gorm:"column:request_method;size:10;comment:请求方式" json:"requestMethod"`             // 请求方式(GET/POST/...)
	OperatorType  string     `gorm:"column:operator_type;size:1;default:'0';comment:操作类别" json:"operatorType"`     // 0其它 1后台 2手机
	OperName      string     `gorm:"column:oper_name;size:50;index;comment:操作人员" json:"operName"`                 // 操作人员
	DeptName      string     `gorm:"column:dept_name;size:30;comment:部门名称" json:"deptName"`                       // 部门名称
	OperUrl       string     `gorm:"column:oper_url;size:255;comment:请求URL" json:"operUrl"`                        // 请求URL
	OperIp        string     `gorm:"column:oper_ip;size:128;index;comment:操作IP" json:"operIp"`                    // 操作IP
	OperLocation  string     `gorm:"column:oper_location;size:255;comment:操作地点" json:"operLocation"`              // 操作地点
	OperParam     string     `gorm:"column:oper_param;type:longtext;comment:请求参数" json:"operParam"`               // 请求参数(JSON)
	JsonResult    string     `gorm:"column:json_result;type:longtext;comment:返回参数" json:"jsonResult"`             // 返回参数(JSON)
	Status        string     `gorm:"column:status;size:1;default:'0';index;comment:操作状态" json:"status"`            // 0成功 1失败
	ErrorMsg      string     `gorm:"column:error_msg;size:2000;default:'';comment:错误消息" json:"errorMsg"`          // 错误消息
	CostTime      int        `gorm:"column:cost_time;default:0;comment:消耗时间(毫秒)" json:"costTime"`               // 消耗时间(毫秒)
	OperTime      *time.Time `gorm:"column:oper_time;index;comment:操作时间" json:"operTime"`                         // 操作时间
}

// TableName 自定义表名 sys_oper_log
func (SysOperLog) TableName() string { return "sys_oper_log" }
