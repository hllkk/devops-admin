package system

import (
	"context"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/hllkk/devops-admin/server/service/system"
	"gorm.io/gorm"
)

const initOrderCasbin = system.InitOrderSystem + 1

type initCasbin struct{}

// auto run
func init() {
	system.RegisterInit(initOrderCasbin, &initCasbin{})
}

func (i *initCasbin) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&adapter.CasbinRule{})
}

func (i *initCasbin) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&adapter.CasbinRule{})
}

func (i *initCasbin) InitializerName() string {
	var entity adapter.CasbinRule
	return entity.TableName()
}

func (i *initCasbin) InitializeData(ctx context.Context) (context.Context, error) {
	// Casbin 不预置种子策略: 权限规则由运行时角色授权写入 casbin_rule;
	// 超管则经 SuperAdmin 标志在 CasbinHandler 中间件直接放行(见 source/system/sys_role_menu.go 注释)。
	// 仅写空切片到 ctx 以满足 initializer 契约,不执行任何 INSERT,避免 gorm 对空切片报 "empty slice found"。
	return context.WithValue(ctx, i.InitializerName(), []adapter.CasbinRule{}), nil
}

func (i *initCasbin) DataInserted(ctx context.Context) bool {
	// 无种子策略可插入: 建表完成即视为数据初始化完成(表存在由 TableCreated 保证),
	// 返回 true 使 InitData 跳过 InitializeData,避免对空表反复执行无意义操作。
	return true
}
