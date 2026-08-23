package initialize

import (
	"context"
	"encoding/json"

	"github.com/hllkk/devops-admin/server/global"
	gatewayService "github.com/hllkk/devops-admin/server/service/gateway"
	mediaService "github.com/hllkk/devops-admin/server/service/media"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/hllkk/devops-admin/server/task"
)

// Timer 注册可供定时任务面板选用的命名方法。
// 不再硬编码调度: 调度由 sys_timed_tasks 表驱动(种子见 source/system/timed_task.go),
// 启动加载见 LoadTimedTasks。ctx 由统一 Runner 注入(已带 datascope.WithSystem 与超时)。
// 二开注册自己的任务: 在此追加 task.Register, 然后在面板新建任务选择该方法即可。
func Timer() {
	task.Register("ClearDB", "清理数据库过期日志(操作记录/登录日志/JWT黑名单/定时任务执行日志/错误日志)", func(ctx context.Context, _ json.RawMessage) error {
		// 保留天数取自常规配置 sys_general_config(<=0 则该项不清理)。
		// 闭包在 initialize 包取值后传入 task,避免 task 反向 import service/system 成环
		cfg := (&system.GeneralConfigService{}).Current(ctx)
		return task.ClearTable(global.OPS_DB.WithContext(ctx), task.ClearOptions{
			OperationLogRetentionDays: cfg.OperationLogRetentionDays,
			LoginLogRetentionDays:     cfg.LoginLogRetentionDays,
		})
	})
	task.Register("CleanStaleUploads", "清理过期大文件上传会话", func(ctx context.Context, _ json.RawMessage) error {
		svc := mediaService.MediaUploadService{}
		return svc.CleanupStale(ctx, global.OPS_CONFIG.Media.SessionTTL)
	})
	task.Register("SyncLLMLogs", "同步LiteLLM用量日志(归因+成本重算)", func(ctx context.Context, _ json.RawMessage) error {
		_, err := (&gatewayService.UsageSyncService{}).SyncLLMLogs(ctx)
		return err
	})
	task.Register("ReconcileLLMLogs", "对账回灌LiteLLM用量漏单", func(ctx context.Context, _ json.RawMessage) error {
		_, err := (&gatewayService.UsageSyncService{}).ReconcileLLMLogs(ctx)
		return err
	})
}
