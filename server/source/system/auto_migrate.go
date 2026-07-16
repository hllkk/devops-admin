package system

import (
	"context"

	"github.com/hllkk/devops-admin/server/model/system"
	sysSvc "github.com/hllkk/devops-admin/server/service/system"
)

const initOrderAutoMigrate = sysSvc.InitOrderInternal // 排在所有有数据的 seed initializer 之后

// initAutoMigrate 收纳「无初始数据」的系统表：这些表只需建结构、无需灌 seed。
//
// 它与各 seed initializer（有数据的表，各自 MigrateTable 建表 + InitializeData 灌数据）
// 共同构成建表清单的唯一真相源——RegisterTables（正常启动）与 createTables（首次
// InitDB）都遍历 initializer 体系建表，不再依赖任何独立维护的全量清单。
//
// 维护约定：
//   - 新增「无数据表」→ 把模型追加到下方 models()，两条建表路径自动覆盖；
//   - 新增「有数据表」→ 另写一个 seed initializer（参考 sys_menu.go），不要加到这里。
type initAutoMigrate struct{}

func init() { sysSvc.RegisterInit(initOrderAutoMigrate, &initAutoMigrate{}) }

// models 返回所有无初始数据的系统表模型。MigrateTable 与 TableCreated 共用此清单，
// 避免出现两份不一致的表列表。
func (i *initAutoMigrate) models() []interface{} {
	return []interface{}{
		&system.JwtBlacklist{},
		&system.SysError{},
		&system.SysPost{},
		&system.SysLoginLog{},
		&system.SysOperLog{},
		&system.SysNotice{},
	}
}

func (i *initAutoMigrate) MigrateTable(ctx context.Context) (context.Context, error) {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return ctx, err
	}
	return ctx, db.AutoMigrate(i.models()...)
}

func (i *initAutoMigrate) TableCreated(ctx context.Context) bool {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return false
	}
	for _, m := range i.models() {
		if !db.Migrator().HasTable(m) {
			return false
		}
	}
	return true
}

func (i *initAutoMigrate) InitializerName() string { return "auto_migrate_tables" }

// InitializeData 这些表无初始数据，直接返回。
func (i *initAutoMigrate) InitializeData(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

// DataInserted 无数据需要插入，恒为 true，InitData 遍历时直接跳过。
func (i *initAutoMigrate) DataInserted(ctx context.Context) bool { return true }
