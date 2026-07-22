package task

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hllkk/devops-admin/server/model/common"

	"gorm.io/gorm"
)

//@author: [songzhibin97](https://github.com/songzhibin97)
//@function: ClearTable
//@description: 清理数据库表数据
//@param: db(数据库对象) *gorm.DB, tableName(表名) string, compareField(比较字段) string, interval(间隔) string
//@return: error

// ClearOptions 日志清理保留天数(单位:天),<=0 表示该项不清理(永久保留)。
// 由调用方(initialize/timer.go 的注册闭包)从 GeneralConfigService.Current() 注入,
// 避免 task 包反向 import service/system 形成循环依赖(service/system 已 import task)。
type ClearOptions struct {
	OperationLogRetentionDays int // 操作记录(sys_oper_log)保留天数
	LoginLogRetentionDays     int // 登录日志(sys_login_log)保留天数
}

// ClearTable 清理数据库过期数据。
//   - sys_oper_log / sys_login_log:保留天数取自 ClearOptions(天),<=0 跳过
//   - jwt_blacklists(7天) / sys_timed_task_logs(30天):与常规配置无关,保持硬编码
//
// 所有表均嵌入 OPS_BASE,列名锁定为 create_time(见 global/model.go)。
// GVA 遗留的 created_at 列名在本项目不存在,旧硬编码已修正。
func ClearTable(db *gorm.DB, opts ClearOptions) error {
	if db == nil {
		return errors.New("db Cannot be empty")
	}

	var details []common.ClearDB
	if opts.OperationLogRetentionDays > 0 {
		details = append(details, common.ClearDB{
			TableName:    "sys_oper_log",
			CompareField: "create_time",
			Interval:     fmt.Sprintf("%dh", opts.OperationLogRetentionDays*24),
		})
	}
	details = append(details, common.ClearDB{
		TableName:    "jwt_blacklists",
		CompareField: "create_time",
		Interval:     "168h",
	})
	details = append(details, common.ClearDB{
		TableName:    "sys_timed_task_logs",
		CompareField: "create_time",
		Interval:     "720h", // 执行日志保留 30 天
	})
	if opts.LoginLogRetentionDays > 0 {
		details = append(details, common.ClearDB{
			TableName:    "sys_login_log",
			CompareField: "create_time", // OPS_AUDIT_MODEL 锁定列名
			Interval:     fmt.Sprintf("%dh", opts.LoginLogRetentionDays*24),
		})
	}

	var firstErr error
	for _, detail := range details {
		duration, err := time.ParseDuration(detail.Interval)
		if err != nil {
			return err
		}
		if duration < 0 {
			return errors.New("parse duration < 0")
		}
		if err = db.Debug().Exec(fmt.Sprintf("DELETE FROM %s WHERE %s < ?", detail.TableName, detail.CompareField), time.Now().Add(-duration)).Error; err != nil {
			// PostgreSQL "relation does not exist"(42P01) 表示表尚未建/被删,跳过继续清理其他表
			if isTableNotExistErr(err) {
				continue
			}
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// isTableNotExistErr 判断是否为"表不存在"错误(允许跳过不存在的表继续清理其他表)。
// PostgreSQL SQLSTATE 42P01 "relation does not exist"; MySQL errno 1146 "table doesn't exist"。
func isTableNotExistErr(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "42P01") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "doesn't exist")
}
