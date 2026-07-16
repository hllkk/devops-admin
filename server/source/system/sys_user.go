package system

import (
	"context"
	"errors"
	"fmt"

	"github.com/hllkk/devops-admin/server/model/system"
	sysSvc "github.com/hllkk/devops-admin/server/service/system"
	"github.com/hllkk/devops-admin/server/utils"
	"gorm.io/gorm"
)

const initOrderUser = sysSvc.InitOrderSystem + 4

type initUser struct{}

func init() { sysSvc.RegisterInit(initOrderUser, &initUser{}) }

func (i *initUser) MigrateTable(ctx context.Context) (context.Context, error) {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return ctx, err
	}
	return ctx, db.AutoMigrate(&system.SysUser{})
}

func (i *initUser) TableCreated(ctx context.Context) bool {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return false
	}
	return db.Migrator().HasTable(&system.SysUser{})
}

func (i *initUser) InitializerName() string { return system.SysUser{}.TableName() }

func (i *initUser) InitializeData(ctx context.Context) (context.Context, error) {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return ctx, err
	}
	pw := "admin123"
	if v, _ := ctx.Value("adminPassword").(string); v != "" {
		pw = v
	} else {
		fmt.Println("[init] adminPassword 未提供，seed 用户使用默认密码 admin123，生产部署请经 /init/initdb 传入")
	}
	hashed := utils.BcryptHash(pw)
	entities := []system.SysUser{
		{UserId: 101, DeptId: 2, DeptName: "北京总部", UserName: "super", NickName: "超级管理员", Password: hashed, Status: "0"},
		{UserId: 102, DeptId: 2, DeptName: "北京总部", UserName: "admin", NickName: "管理员", Password: hashed, Status: "0"},
		{UserId: 103, DeptId: 3, DeptName: "天津工厂", UserName: "test1", NickName: "测试用户", Password: hashed, Status: "0"},
	}
	if err := db.Create(&entities).Error; err != nil {
		return ctx, fmt.Errorf("%s 表数据初始化失败: %w", i.InitializerName(), err)
	}
	return context.WithValue(ctx, i.InitializerName(), entities), nil
}

func (i *initUser) DataInserted(ctx context.Context) bool {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return false
	}
	return !errors.Is(db.Where("user_id = ?", 103).First(&system.SysUser{}).Error, gorm.ErrRecordNotFound)
}
