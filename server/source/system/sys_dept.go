package system

import (
	"context"
	"errors"
	"fmt"

	"github.com/hllkk/devops-admin/server/model/system"
	sysSvc "github.com/hllkk/devops-admin/server/service/system"
	"gorm.io/gorm"
)

const initOrderDept = sysSvc.InitOrderSystem + 1

type initDept struct{}

func init() { sysSvc.RegisterInit(initOrderDept, &initDept{}) }

func (i *initDept) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, sysSvc.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&system.SysDept{})
}

func (i *initDept) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&system.SysDept{})
}

func (i *initDept) InitializerName() string { return system.SysDept{}.TableName() }

func (i *initDept) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, sysSvc.ErrMissingDBContext
	}
	entities := []system.SysDept{
		{DeptId: 1, ParentId: 0, Ancestors: "0", DeptName: "XXX科技", OrderNum: 0, Status: "0"},
		{DeptId: 2, ParentId: 1, Ancestors: "0,1", DeptName: "北京总部", OrderNum: 1, Status: "0"},
		{DeptId: 3, ParentId: 1, Ancestors: "0,1", DeptName: "天津工厂", OrderNum: 2, Status: "0"},
	}
	if err := db.Create(&entities).Error; err != nil {
		return ctx, fmt.Errorf("%s 表数据初始化失败: %w", i.InitializerName(), err)
	}
	return context.WithValue(ctx, i.InitializerName(), entities), nil
}

func (i *initDept) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return !errors.Is(db.Where("dept_id = ?", 3).First(&system.SysDept{}).Error, gorm.ErrRecordNotFound)
}
