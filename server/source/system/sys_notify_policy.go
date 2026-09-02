package system

import (
	"context"

	sysModel "github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/service/system"
	"gorm.io/gorm"
)

// 通知策略排在通知公告之后(链尾自注册,避免撞号)。
const initOrderNotifyPolicy = initOrderNotice + 1

type initNotifyPolicy struct{}

// auto run
func init() {
	system.RegisterInit(initOrderNotifyPolicy, &initNotifyPolicy{})
}

func (i *initNotifyPolicy) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&sysModel.SysNotifyPolicy{}, &sysModel.SysWecomBotGroup{})
}

func (i *initNotifyPolicy) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&sysModel.SysNotifyPolicy{}) && db.Migrator().HasTable(&sysModel.SysWecomBotGroup{})
}

func (i *initNotifyPolicy) InitializerName() string {
	return sysModel.SysNotifyPolicy{}.TableName()
}

// InitializeData 通知策略无种子数据(场景行由设置页保存时 upsert)。
func (i *initNotifyPolicy) InitializeData(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

// DataInserted 无种子数据,恒为 true(探针幂等跳过 InitializeData)。
func (i *initNotifyPolicy) DataInserted(ctx context.Context) bool {
	return true
}
