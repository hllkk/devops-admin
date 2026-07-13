package system

import (
	"context"
	"errors"
	"fmt"

	"github.com/hllkk/devops-admin/server/model/system"
	sysSvc "github.com/hllkk/devops-admin/server/service/system"
	"gorm.io/gorm"
)

const initOrderUserRole = sysSvc.InitOrderSystem + 5

type initUserRole struct{}

func init() { sysSvc.RegisterInit(initOrderUserRole, &initUserRole{}) }

func (i *initUserRole) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, sysSvc.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&system.SysUserRole{})
}

func (i *initUserRole) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&system.SysUserRole{})
}

func (i *initUserRole) InitializerName() string { return system.SysUserRole{}.TableName() }

func (i *initUserRole) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, sysSvc.ErrMissingDBContext
	}
	entities := []system.SysUserRole{
		{UserId: 101, RoleId: 1}, // super → superadmin
		{UserId: 102, RoleId: 2}, // admin → admin
		{UserId: 103, RoleId: 3}, // test1 → user
	}
	if err := db.Create(&entities).Error; err != nil {
		return ctx, fmt.Errorf("%s 表数据初始化失败: %w", i.InitializerName(), err)
	}
	return context.WithValue(ctx, i.InitializerName(), entities), nil
}

func (i *initUserRole) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return !errors.Is(db.Where("user_id = ? AND role_id = ?", 103, 3).First(&system.SysUserRole{}).Error, gorm.ErrRecordNotFound)
}
