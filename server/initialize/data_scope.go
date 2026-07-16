package initialize

import (
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/service"
	"github.com/hllkk/devops-admin/server/utils/datascope"
)

// RegisterDataScopeCallbacks 为主库及所有多库连接注册数据权限 GORM 回调,
// 并接上审计事件异步落表(sys_data_access_logs)。
// 需在 DB(含 OPS_DBList)初始化完成后调用。
func RegisterDataScopeCallbacks() {
	datascope.RegisterCallbacks(global.OPS_DB)
	for _, db := range global.OPS_DBList {
		datascope.RegisterCallbacks(db)
	}
	auditSvc := &service.ServiceGroupApp.SystemServiceGroup.DataAccessLogService
	datascope.SetAuditHook(auditSvc.Enqueue)
	auditSvc.StartWriter()
}
