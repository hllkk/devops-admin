package system

import (
	"context"
	"errors"
	"fmt"

	"github.com/hllkk/devops-admin/server/model/system"
	sysSvc "github.com/hllkk/devops-admin/server/service/system"
	"gorm.io/gorm"
)

const initOrderRole = sysSvc.InitOrderSystem + 2

type initRole struct{}

func init() { sysSvc.RegisterInit(initOrderRole, &initRole{}) }

func (i *initRole) MigrateTable(ctx context.Context) (context.Context, error) {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return ctx, err
	}
	return ctx, db.AutoMigrate(&system.SysRole{})
}

func (i *initRole) TableCreated(ctx context.Context) bool {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return false
	}
	return db.Migrator().HasTable(&system.SysRole{})
}

func (i *initRole) InitializerName() string { return system.SysRole{}.TableName() }

func (i *initRole) InitializeData(ctx context.Context) (context.Context, error) {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return ctx, err
	}
	entities := []system.SysRole{
		{RoleId: 1, RoleName: "超级管理员", RoleKey: "superadmin", SuperAdmin: true, RoleSort: 0, Status: "0"},
		{RoleId: 2, RoleName: "管理员", RoleKey: "admin", SuperAdmin: false, RoleSort: 1, Status: "0"},
		{RoleId: 3, RoleName: "普通用户", RoleKey: "user", SuperAdmin: false, RoleSort: 2, Status: "0"},
	}
	if err := db.Create(&entities).Error; err != nil {
		return ctx, fmt.Errorf("%s 表数据初始化失败: %w", i.InitializerName(), err)
	}
	return context.WithValue(ctx, i.InitializerName(), entities), nil
}

func (i *initRole) DataInserted(ctx context.Context) bool {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return false
	}
	return !errors.Is(db.Where("role_id = ?", 3).First(&system.SysRole{}).Error, gorm.ErrRecordNotFound)
}
