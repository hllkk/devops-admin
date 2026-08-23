package system

import (
	"context"

	sysModel "github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const initOrderTimedTask = initOrderCasbin + 1

type initTimedTask struct{}

// auto run
func init() {
	system.RegisterInit(initOrderTimedTask, &initTimedTask{})
}

func (i *initTimedTask) InitializerName() string {
	return sysModel.SysTimedTask{}.TableName()
}

func (i *initTimedTask) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&sysModel.SysTimedTask{}, &sysModel.SysTimedTaskLog{})
}

func (i *initTimedTask) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	m := db.Migrator()
	return m.HasTable(&sysModel.SysTimedTask{}) && m.HasTable(&sysModel.SysTimedTaskLog{})
}

// InitializeData 吃狗粮: 原 initialize/timer.go 两条硬编码任务迁为种子, 由面板接管
func (i *initTimedTask) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	entities := []sysModel.SysTimedTask{
		{Name: "ClearDB", Description: "定时清理数据库过期日志(操作记录/JWT黑名单/定时任务执行日志)", Spec: "@daily", ExecutorType: sysModel.TimedTaskExecutorMethod, MethodName: "ClearDB", Enabled: true},
		{Name: "CleanStaleUploads", Description: "定时清理过期大文件上传会话", Spec: "@hourly", ExecutorType: sysModel.TimedTaskExecutorMethod, MethodName: "CleanStaleUploads", Enabled: true},
		{Name: "SyncLLMLogs", Description: "同步LiteLLM用量日志(归因+成本重算,复合游标增量)", Spec: "*/5 * * * *", ExecutorType: sysModel.TimedTaskExecutorMethod, MethodName: "SyncLLMLogs", Enabled: true},
		{Name: "ReconcileLLMLogs", Description: "对账回灌LiteLLM用量漏单(近30天NOT EXISTS兜底)", Spec: "0 * * * *", ExecutorType: sysModel.TimedTaskExecutorMethod, MethodName: "ReconcileLLMLogs", Enabled: true},
		{Name: "AggregateUsage", Description: "聚合用量到日桶+重算预算+超限停用闭环", Spec: "*/5 * * * *", ExecutorType: sysModel.TimedTaskExecutorMethod, MethodName: "AggregateUsage", Enabled: true},
	}
	if err := db.Create(&entities).Error; err != nil {
		return ctx, errors.Wrap(err, sysModel.SysTimedTask{}.TableName()+"表数据初始化失败!")
	}
	next := context.WithValue(ctx, i.InitializerName(), entities)
	return next, nil
}

func (i *initTimedTask) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	if errors.Is(db.Where("name = ?", "ClearDB").First(&sysModel.SysTimedTask{}).Error, gorm.ErrRecordNotFound) {
		return false
	}
	return true
}
