package initialize

import (
	"context"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/hllkk/devops-admin/server/utils/datascope"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// LoadTimedTasks 从 sys_timed_tasks 恢复调度(仅 enabled; 幂等, 可在启动/重载/initdb 后重复调用)。
// 必须在 RegisterTables(建表)之后调用。
func LoadTimedTasks() {
	if global.OPS_DB == nil {
		return
	}
	ctx := datascope.WithSystem(context.Background())
	if err := system.TimedTaskServiceApp.LoadAll(ctx); err != nil {
		logger.Bg().Mod("timedTask").Err(err).Error("定时任务启动加载失败")
	}
}
