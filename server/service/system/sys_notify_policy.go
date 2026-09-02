package system

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// NotifyPolicyService 通知策略(scene_key 一行)读写：定时内容型通知(如 TokenPlan 晨报)
// 的目标与启停配置，设置页聚合保存 upsert、定时任务读。
type NotifyPolicyService struct{}

// 默认晨报发送时间(与 timed_task 种子 33 8 * * 1-5 对齐)
const defaultMorningSendTime = "08:33"

// Get 按场景读策略；不存在时返回默认骨架(Enabled=false, TargetType=users)，由调用方按未启用处理。
// 存量行 SendTime 为空(AutoMigrate 加列前的老数据)时兜底默认值。
func (s *NotifyPolicyService) Get(ctx context.Context, sceneKey string) (system.SysNotifyPolicy, error) {
	var p system.SysNotifyPolicy
	if sceneKey == "" {
		return p, errors.New("场景标识不能为空")
	}
	err := global.OPS_DB.WithContext(ctx).Where("scene_key = ?", sceneKey).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return system.SysNotifyPolicy{
			SceneKey: sceneKey, Enabled: false,
			TargetType: system.NotifyPolicyTargetUsers, SendTime: defaultMorningSendTime,
		}, nil
	}
	if p.SendTime == "" {
		p.SendTime = defaultMorningSendTime
	}
	return p, err
}

// Upsert 按场景保存策略(scene_key 唯一，存在则更新)；晨报场景同步把发送时间写进
// 定时任务 sys_timed_tasks 的 spec 并热重载(行不存在则自动创建,已有库配置即生效)。
func (s *NotifyPolicyService) Upsert(ctx context.Context, p system.SysNotifyPolicy) error {
	if p.SceneKey == "" {
		return errors.New("场景标识不能为空")
	}
	if p.TargetType == "" {
		p.TargetType = system.NotifyPolicyTargetUsers
	}
	if p.TargetType != system.NotifyPolicyTargetAll &&
		p.TargetType != system.NotifyPolicyTargetDepts && p.TargetType != system.NotifyPolicyTargetUsers {
		return errors.New("目标类型非法(须 all/depts/users)")
	}
	db := global.OPS_DB.WithContext(ctx)
	var exist system.SysNotifyPolicy
	if err := db.Where("scene_key = ?", p.SceneKey).First(&exist).Error; err == nil {
		if err := db.Model(&exist).Updates(map[string]any{
			"enabled":     p.Enabled,
			"target_type": p.TargetType,
			"target_ids":  p.TargetIds,
			"send_time":   p.SendTime,
			"params":      p.Params,
		}).Error; err != nil {
			return err
		}
	} else if err := db.Create(&p).Error; err != nil {
		return err
	}
	if p.SceneKey == system.NotifySceneTokenPlanMorning {
		if err := syncMorningSchedule(ctx, p.SendTime); err != nil {
			// 策略已保存、调度同步失败不回滚(定时任务面板可兜底修),记日志即可
			logger.WithCtx(ctx).Mod("system").Err(err).Error("晨报发送时间同步定时任务失败")
			return fmt.Errorf("策略已保存,但同步发送时间失败: %w", err)
		}
	}
	return nil
}

// syncMorningSchedule 把晨报发送时间同步到定时任务 TokenPlanMorningReport:
// spec = "m h * * 1-5"(工作日);行不存在则创建并调度(已有库未种子时配置即生效),
// 存在且 spec 变化则更新并热重载(复用 TimedTaskService 保证调度器一致)。
func syncMorningSchedule(ctx context.Context, sendTime string) error {
	hour, minute, err := parseMorningSendTime(sendTime)
	if err != nil {
		return err
	}
	spec := fmt.Sprintf("%d %d * * 1-5", minute, hour)
	db := global.OPS_DB.WithContext(ctx)
	var row system.SysTimedTask
	err = db.Where("method_name = ?", "TokenPlanMorningReport").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = system.SysTimedTask{
			Name:         "TokenPlanMorningReport",
			Description:  "TokenPlan晨报推送(工作日,发送时间由通知设置维护)",
			Spec:         spec,
			ExecutorType: system.TimedTaskExecutorMethod,
			MethodName:   "TokenPlanMorningReport",
			Enabled:      true,
		}
		return (&TimedTaskService{}).CreateTimedTask(ctx, &row)
	}
	if err != nil {
		return err
	}
	if row.Spec == spec {
		return nil // 时间未变,不动调度
	}
	row.Spec = spec
	return (&TimedTaskService{}).UpdateTimedTask(ctx, &row)
}

// parseMorningSendTime 解析 "HH:mm"(24h)。空值回退默认时间。
func parseMorningSendTime(sendTime string) (hour, minute int, err error) {
	if sendTime == "" {
		sendTime = defaultMorningSendTime
	}
	t, err := time.Parse("15:04", sendTime)
	if err != nil {
		return 0, 0, errors.New("发送时间格式非法(须 HH:mm,如 08:33)")
	}
	return t.Hour(), t.Minute(), nil
}
