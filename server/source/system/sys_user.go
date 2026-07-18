package system

import (
	"context"

	"github.com/google/uuid"
	sysModel "github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const initOrderUser = initOrderMenu + 1

type initUser struct{}

// auto run
func init() {
	system.RegisterInit(initOrderUser, &initUser{})
}

func (i *initUser) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&sysModel.SysUser{}, &sysModel.SysLoginLog{}, &sysModel.SysOperLog{})
}

func (i *initUser) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&sysModel.SysUser{})
}

func (i *initUser) InitializerName() string {
	return sysModel.SysUser{}.TableName()
}

func (i *initUser) InitializeData(ctx context.Context) (next context.Context, err error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}

	// 角色先于用户初始化(initOrderRole < initOrderUser),且 sys_role.InitializeData
	// 已将带雪花 ID 的角色切片回填到 ctx(键为 sys_roles 表名)。
	// 雪花 ID 在 db.Create 时才生成,因此这里只能从 ctx 取,不能硬编码。
	roleEntities, ok := ctx.Value(sysModel.SysRole{}.TableName()).([]sysModel.SysRole)
	if !ok {
		return ctx, system.ErrMissingDependentContext
	}
	var superRoleID int64
	var adminRoleID int64
	var userRoleID int64
	for _, r := range roleEntities {
		if r.RoleKey == "super" {
			superRoleID = r.RoleId
			break
		}
		if r.RoleKey == "admin" {
			adminRoleID = r.RoleId
			break
		}
		if r.RoleKey == "user" {
			userRoleID = r.RoleId
			break
		}
	}
	if superRoleID == 0 {
		return ctx, errors.New("未找到 super 角色,请确认 sys_role 初始化已先执行")
	}

	ap := ctx.Value("adminPassword")
	apStr, ok := ap.(string)
	if !ok {
		apStr = "123456"
	}
	adminPassword := utils.BcryptHash(apStr)

	entities := []sysModel.SysUser{
		{
			UUID:        uuid.New(),
			UserName:    "super",
			Password:    adminPassword,
			NickName:    "超级管理员",
			Phonenumber: "13666666666",
			Email:       "super@example.com",
			Avatar:      "https://minio.xlsea.cn/ruoyi/2026/01/30/3021f4d908ff44d6b42f8e8616126734.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Date=20260717T074418Z&X-Amz-SignedHeaders=host&X-Amz-Credential=fCUgpKZtsmLNuaHERD4v%2F20260717%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Expires=120&X-Amz-Signature=d1eacaaa8a06b81f60e8445ee5c224dd75d29297326ce3fc41aca630623973a3",
			RoleId:      superRoleID, // 主角色(登录链路 claims 用)
		},
		{
			UUID:        uuid.New(),
			UserName:    "admin",
			Password:    adminPassword,
			NickName:    "系统管理员",
			Phonenumber: "13777777777",
			Email:       "admin@example.com",
			Avatar:      "https://vue3.baiwumm.com/static/image/2024-07/cc9e77ee-cf84-48e8-a9d0-dc3e9d21224c.jpeg",
			RoleId:      adminRoleID, // 主角色(登录链路 claims 用)
		},
		{
			UUID:        uuid.New(),
			UserName:    "user",
			Password:    adminPassword,
			NickName:    "普通用户",
			Phonenumber: "13888888888",
			Email:       "user@example.com",
			Avatar:      "https://vue3.baiwumm.com/static/image/2024-07/cc9e77ee-cf84-48e8-a9d0-dc3e9d21224c.jpeg",
			RoleId:      userRoleID, // 主角色(登录链路 claims 用)
		},
	}
	if err = db.Create(&entities).Error; err != nil {
		return ctx, errors.Wrapf(err, "%s表数据初始化失败!", sysModel.SysUser{}.TableName())
	}
	// Create 后 entities[0].UserId 由雪花回调回填

	// 写显式连接表 sys_user_role,使"主角色(RoleId)"与"多角色(Roles many2many)"一致;
	// 与 SysUser.Roles 的 many2many tag 同表,后续 service 层可经 Association 或本 struct 直接操作。
	userRoleLinks := []sysModel.SysUserRole{
		{SysUserId: entities[0].UserId, SysRoleId: superRoleID},
	}
	if err = db.Create(&userRoleLinks).Error; err != nil {
		return ctx, errors.Wrapf(err, "%s连接表数据初始化失败!", (&sysModel.SysUserRole{}).TableName())
	}

	next = context.WithValue(ctx, i.InitializerName(), entities)
	return next, nil
}

func (i *initUser) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	if errors.Is(db.Where("user_name = ?", "super").
		First(&sysModel.SysUser{}).Error, gorm.ErrRecordNotFound) {
		return false
	}
	return true
}
