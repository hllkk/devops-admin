package system

import (
	"context"

	sysModel "github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/service/system"
	"gorm.io/gorm"
)

// 通知公告无跨初始化器依赖,排到单链末尾(initOrderDict 之后),避免撞号。
const initOrderNotice = initOrderDict + 1

type initNotice struct{}

// auto run
func init() {
	system.RegisterInit(initOrderNotice, &initNotice{})
}

func (i *initNotice) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&sysModel.SysNotice{}, &sysModel.SysNoticeRecord{})
}

func (i *initNotice) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&sysModel.SysNotice{})
}

func (i *initNotice) InitializerName() string {
	return sysModel.SysNotice{}.TableName()
}

// InitializeData 通知公告无种子数据。
func (i *initNotice) InitializeData(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

// DataInserted 无种子数据,恒为 true(探针幂等跳过 InitializeData)。
func (i *initNotice) DataInserted(ctx context.Context) bool {
	return true
}
