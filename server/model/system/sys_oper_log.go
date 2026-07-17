package system

import (
	"time"

	"github.com/hllkk/devops-admin/server/global"
)

// SysOperLog 操作日志(对外业务实体,字段对齐前端 Api.Log.OperLog)
//
// 设计要点:
//   - 嵌入 OPS_AUDIT_MODEL 获取 createBy/createTime/updateBy/updateTime(对齐前端 CommonRecord)
//   - 主键 OperId 走雪花 int64(json operId,string); 表名 sys_oper_log
//   - businessType 对齐前端 BusinessType '0'~'9'(0其它 1新增 2修改 3删除 4授权 5导出 6导入 7强退 8生成代码 9清空)
//   - status 对齐前端 EnableStatus '0'/'1'(0正常 1异常)
//   - operTime 为操作发生时间(业务字段,区别于记录写入时间 createTime)
//   - operParam/jsonResult/errorMsg 为可能较大的请求/响应/错误正文,统一 type:text
type SysOperLog struct {
	global.OPS_AUDIT_MODEL
	OperId        int64     `json:"operId,string" gorm:"primarykey;comment:日志主键"` // 日志主键
	Title         string    `json:"title" gorm:"index;comment:系统模块"`              // 系统模块
	BusinessType  string    `json:"businessType" gorm:"size:1;comment:操作类型 0~9"` // 操作类型(对齐前端 BusinessType '0'~'9')
	Method        string    `json:"method" gorm:"comment:方法名称"`                 // 方法名称
	RequestMethod string    `json:"requestMethod" gorm:"comment:请求方式"`           // 请求方式(GET/POST...)
	OperatorType  string    `json:"operatorType" gorm:"size:1;comment:操作类别"`     // 操作类别(0其它 1后台用户 2手机端用户)
	OperName      string    `json:"operName" gorm:"index;comment:操作人员"`          // 操作人员
	DeptName      string    `json:"deptName" gorm:"comment:部门名称"`               // 部门名称
	OperUrl       string    `json:"operUrl" gorm:"comment:请求URL"`               // 请求URL
	OperIp        string    `json:"operIp" gorm:"comment:操作IP"`                  // 操作IP
	OperLocation  string    `json:"operLocation" gorm:"comment:操作地点"`            // 操作地点
	OperParam     string    `json:"operParam" gorm:"type:text;comment:请求参数"`     // 请求参数
	JsonResult    string    `json:"jsonResult" gorm:"type:text;comment:返回参数"`    // 返回参数
	Status        string    `json:"status" gorm:"size:1;comment:操作状态 0正常1异常"` // 操作状态(对齐前端 EnableStatus '0'/'1')
	ErrorMsg      string    `json:"errorMsg" gorm:"type:text;comment:错误消息"`      // 错误消息
	OperTime      time.Time `json:"operTime" gorm:"comment:操作时间"`               // 操作时间(业务时间,区别于 createTime)
	CostTime      int       `json:"costTime" gorm:"comment:消耗时间(毫秒)"`          // 消耗时间
}

func (SysOperLog) TableName() string {
	return "sys_oper_log"
}
