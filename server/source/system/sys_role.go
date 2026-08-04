package system

import (
	"context"

	sysModel "github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const initOrderRole = initOrderCasbin + 1

type initRole struct{}

// auto run
func init() {
	system.RegisterInit(initOrderRole, &initRole{})
}

func (i *initRole) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&sysModel.SysRole{})
}

func (i *initRole) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&sysModel.SysRole{})
}

func (i *initRole) InitializerName() string {
	return sysModel.SysRole{}.TableName()
}

func (i *initRole) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}

	// DefaultRouter 统一为 home:登录后所有角色落地 AI 个人中心首页。route.ts redirectFromLogin
	// 用 userInfo.defaultRouter 跳转,sys_route.go routeHomeDefault 亦对齐 home。统一 home 与全员授权的
	// home 菜单一致,落地不再 404。
	entities := []sysModel.SysRole{
		{RoleName: "超级管理员", RoleKey: "super", RoleSort: 1, SuperAdmin: true, DataScope: 1, DefaultRouter: "home", Remark: "系统最高权限,可操作所有功能"},
		{RoleName: "系统管理员", RoleKey: "admin", RoleSort: 2, SuperAdmin: false, DataScope: 1, DefaultRouter: "home", Remark: "系统普通管理员,可操作大部分功能"},
		{RoleName: "普通用户", RoleKey: "user", RoleSort: 3, SuperAdmin: false, DataScope: 3, DefaultRouter: "home", Remark: "系统普通用户,仅可操作个人相关功能"},
	}

	if err := db.Create(&entities).Error; err != nil {
		return ctx, errors.Wrapf(err, "%s表数据初始化失败!", sysModel.SysRole{}.TableName())
	}

	next := context.WithValue(ctx, i.InitializerName(), entities)
	return next, nil
}

func (i *initRole) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	if errors.Is(db.Where("role_key = ?", "super").
		First(&sysModel.SysRole{}).Error, gorm.ErrRecordNotFound) { // 判断是否存在数据
		return false
	}
	return true
}
