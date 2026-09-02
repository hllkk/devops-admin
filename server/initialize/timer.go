package initialize

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/common"
	sysModel "github.com/hllkk/devops-admin/server/model/system"
	gatewayService "github.com/hllkk/devops-admin/server/service/gateway"
	mediaService "github.com/hllkk/devops-admin/server/service/media"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/hllkk/devops-admin/server/task"
	"github.com/hllkk/devops-admin/server/utils/logger"
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
	task.Register("SyncMcpLogs", "同步LiteLLM MCP调用日志(工具归因+per_call成本)", func(ctx context.Context, _ json.RawMessage) error {
		_, err := (&gatewayService.UsageSyncService{}).SyncMcpLogs(ctx)
		return err
	})
	task.Register("ReconcileMcpLogs", "对账回灌MCP调用漏单", func(ctx context.Context, _ json.RawMessage) error {
		_, err := (&gatewayService.UsageSyncService{}).ReconcileMcpLogs(ctx)
		return err
	})
	task.Register("CleanupUsageLogs", "清理过期用量日志(llm+mcp按保留天数物理删,log-retention-days<=0禁用)", func(ctx context.Context, _ json.RawMessage) error {
		_, err := (&gatewayService.UsageSyncService{}).CleanupUsageLogs(ctx)
		return err
	})
	task.Register("CheckBudgetAlerts", "预算预警检查(软限预警/硬限停用+站内与企微通知;此前仅手动触发才发通知,本任务补齐定时路径)", func(ctx context.Context, _ json.RawMessage) error {
		results, err := (&gatewayService.BudgetRuleService{}).CheckBudgetAlerts(ctx)
		if err != nil {
			return err
		}
		if len(results) == 0 {
			return nil
		}
		// 告警→草稿(gateway 纯函数)→三渠道发送(system 服务);闭包注入规避 import 环
		notifyCfg := (&system.NotifyConfigService{}).Current(ctx)
		sendSvc := &system.NotifySendService{}
		for _, d := range gatewayService.BudgetAlertNotices(results) {
			if err := sendSvc.Send(ctx, system.SendRequest{
				Title: d.Title, Content: d.Content,
				TargetType: sysModel.NotifyPolicyTargetUsers, UserIds: d.TargetUserIds,
				Channels: system.SendChannels{
					InApp:    true,
					WecomApp: notifyCfg.WecomPushEnabled && notifyCfg.PushBudgetAlertEnabled,
				},
			}); err != nil {
				logger.WithCtx(ctx).Mod("gateway").Err(err).Warn(fmt.Sprintf("预算规则 %d 告警通知发送失败", d.RuleId))
			}
		}
		return nil
	})
	task.Register("AggregateUsage", "聚合用量到日桶+重算预算+超限停用闭环", func(ctx context.Context, _ json.RawMessage) error {
		_, err := (&gatewayService.UsageAggregateService{}).AggregateUsage(ctx)
		return err
	})
	task.Register("SyncProviderBalances", "同步供应商套餐余量(百炼TokenPlan坐席+共享包,旁路只读)", func(ctx context.Context, _ json.RawMessage) error {
		_, err := (&gatewayService.ProviderBalanceService{}).SyncAllProviderBalances(ctx)
		return err
	})
	task.Register("ResyncAiKeys", "全量重推AI密钥投影到LiteLLM(改名级联/授权对齐同步失败的漂移兜底)", func(ctx context.Context, _ json.RawMessage) error {
		_, err := (&gatewayService.AiKeyService{}).ResyncAllKeys(ctx)
		return err
	})
	task.Register("HealthCheckMcps", "MCP服务器定时健康巡检(全量启用经LiteLLM探测,结果落库)", func(ctx context.Context, _ json.RawMessage) error {
		_, err := (&gatewayService.McpService{}).HealthCheckAllMcps(ctx)
		return err
	})
	task.Register("SyncWecomContact", "同步企业微信通讯录(部门/用户/岗位单向拉取,含复职恢复与离职停用;手动触发走部门管理页按钮)", func(ctx context.Context, _ json.RawMessage) error {
		_, err := (&system.WecomContactService{}).SyncStructure(ctx)
		return err
	})
	task.Register("TokenPlanMorningReport", "TokenPlan晨报推送(工作日,汇总坐席+共享包余量与重置日,按策略发目标部门/用户:站内+企微应用+群机器人;策略未启用时不发)", func(ctx context.Context, _ json.RawMessage) error {
		policy, err := (&system.NotifyPolicyService{}).Get(ctx, sysModel.NotifySceneTokenPlanMorning)
		if err != nil {
			return err
		}
		if !policy.Enabled {
			return nil // 策略未启用,静默跳过(任务开着,配置可控)
		}
		drafts, err := (&gatewayService.MorningReportService{}).BuildMorningReport(ctx)
		if err != nil {
			return err
		}
		if len(drafts) == 0 {
			return nil // 无 token_plan 余量快照(未配置同步或尚未跑),跳过
		}
		// 策略目标(depts/users/all)解析为发送参数。target_ids 由前端部门树/用户选择器
		// 写入,是字符串 id 数组(雪花 id 统一 string 序列化),须用 Int64StringSlice 兼容解析;
		// 裸 []int64 unmarshal 必失败,若吞错继续会目标为空、静默不发。
		var userIds, deptIds []int64
		switch policy.TargetType {
		case sysModel.NotifyPolicyTargetDepts:
			var ids common.Int64StringSlice
			if err := json.Unmarshal(policy.TargetIds, &ids); err != nil {
				return fmt.Errorf("解析晨报目标部门失败: %w", err)
			}
			deptIds = ids
		case sysModel.NotifyPolicyTargetUsers:
			var ids common.Int64StringSlice
			if err := json.Unmarshal(policy.TargetIds, &ids); err != nil {
				return fmt.Errorf("解析晨报目标用户失败: %w", err)
			}
			userIds = ids
		}
		notifyCfg := (&system.NotifyConfigService{}).Current(ctx)
		channels := system.SendChannels{
			InApp:    true,
			WecomApp: notifyCfg.WecomPushEnabled && notifyCfg.PushMorningReportEnabled,
			WecomBot: notifyCfg.WecomBotEnabled && notifyCfg.PushMorningReportEnabled,
		}
		sendSvc := &system.NotifySendService{}
		for _, d := range drafts {
			if err := sendSvc.Send(ctx, system.SendRequest{
				Title: d.Title, Content: d.Content, Markdown: d.Markdown,
				Url:        "/gateway",
				TargetType: policy.TargetType, UserIds: userIds, DeptIds: deptIds,
				Channels: channels,
			}); err != nil {
				logger.WithCtx(ctx).Mod("gateway").Err(err).Warn(fmt.Sprintf("供应商 %s 晨报发送失败", d.ProviderName))
			}
		}
		return nil
	})
}
