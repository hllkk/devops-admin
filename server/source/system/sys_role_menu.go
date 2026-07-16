package system

import (
	"context"
	"fmt"

	"github.com/hllkk/devops-admin/server/model/system"
	sysSvc "github.com/hllkk/devops-admin/server/service/system"
)

const initOrderRoleMenu = sysSvc.InitOrderSystem + 6

type initRoleMenu struct{}

func init() { sysSvc.RegisterInit(initOrderRoleMenu, &initRoleMenu{}) }

func (i *initRoleMenu) MigrateTable(ctx context.Context) (context.Context, error) {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return ctx, err
	}
	return ctx, db.AutoMigrate(&system.SysRoleMenu{})
}

func (i *initRoleMenu) TableCreated(ctx context.Context) bool {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return false
	}
	return db.Migrator().HasTable(&system.SysRoleMenu{})
}

func (i *initRoleMenu) InitializerName() string { return system.SysRoleMenu{}.TableName() }

func (i *initRoleMenu) InitializeData(ctx context.Context) (context.Context, error) {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return ctx, err
	}
	// super(admin) 与 admin 挂全部菜单；user(3) 不挂（无系统管理权限）
	var allMenuIds []int64
	if err := db.Model(&system.SysMenu{}).Pluck("menu_id", &allMenuIds).Error; err != nil {
		return ctx, fmt.Errorf("%s 表查询菜单失败: %w", i.InitializerName(), err)
	}
	var entities []system.SysRoleMenu
	for _, rid := range []int64{1, 2} {
		for _, mid := range allMenuIds {
			entities = append(entities, system.SysRoleMenu{RoleId: rid, MenuId: mid})
		}
	}
	if err := db.Create(&entities).Error; err != nil {
		return ctx, fmt.Errorf("%s 表数据初始化失败: %w", i.InitializerName(), err)
	}
	return context.WithValue(ctx, i.InitializerName(), entities), nil
}

func (i *initRoleMenu) DataInserted(ctx context.Context) bool {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return false
	}
	var c int64
	db.Model(&system.SysRoleMenu{}).Where("role_id = ?", 2).Count(&c)
	return c > 0
}
