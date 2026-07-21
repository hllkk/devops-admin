package system

import (
	"context"
	"time"

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
	// 显式迁移 SysUser 的全部 many2many 连接表: GORM 对主表 AutoMigrate 不会可靠迁移 join 表,
	// 历史上 sys_user_role 因未纳入迁移而 schema 漂移(旧列 user_id/role_id 残留); 另两个连接表列名虽未变更,
	// 同样存在隐式建表不可靠的隐患, 一并纳入显式迁移堵死同类问题。
	return ctx, db.AutoMigrate(
		&sysModel.SysUser{}, &sysModel.SysLoginLog{}, &sysModel.SysOperLog{},
		&sysModel.SysUserRole{}, &sysModel.SysUserDepartment{}, &sysModel.SysUserPost{},
	)
}

func (i *initUser) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	// 主表与全部 many2many 连接表都已存在才视为已建; 任一缺失或 sys_user_role 关键列漂移时返回 false,
	// 触发 MigrateTable 用 AutoMigrate 校正(幂等, 重复无害)。
	if !db.Migrator().HasTable(&sysModel.SysUser{}) ||
		!db.Migrator().HasTable(&sysModel.SysUserRole{}) ||
		!db.Migrator().HasTable(&sysModel.SysUserDepartment{}) ||
		!db.Migrator().HasTable(&sysModel.SysUserPost{}) {
		return false
	}
	// sys_user_role 列名历史变更过(fd99b5d: user_id/role_id → 13816c0: sys_user_id/sys_role_id),
	// 旧库可能表在但列漂移, 故额外校验关键列; 另两个连接表列名从未变更, 表在即视为 OK。
	return db.Migrator().HasColumn(&sysModel.SysUserRole{}, "sys_user_id")
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
	// 注:原实现每段 if 后 break,循环只命中第一个角色(roleEntities[0]=super),
	// adminRoleID/userRoleID 恒为 0,Create 时被 GORM default:888 覆盖。改 switch 遍历全部角色。
	for _, r := range roleEntities {
		switch r.RoleKey {
		case "super":
			superRoleID = r.RoleId
		case "admin":
			adminRoleID = r.RoleId
		case "user":
			userRoleID = r.RoleId
		}
	}
	if superRoleID == 0 || adminRoleID == 0 || userRoleID == 0 {
		return ctx, errors.New("未找到 super/admin/user 角色,请确认 sys_role 初始化已先执行")
	}

	ap := ctx.Value("adminPassword")
	apStr, ok := ap.(string)
	if !ok || apStr == "" {
		// adminPassword 已由初始化向导 binding:"required" 强制传入(sys_init.go);
		// 走到这里说明 init 流程被绕过,拒绝兜底 123456 弱口令,直接报错。
		return ctx, errors.New("未传入 adminPassword,拒绝使用默认弱口令初始化;请走初始化向导")
	}
	adminPassword := utils.BcryptHash(apStr)
	// seed 用户也设置 PasswordUpdatedAt,否则密码过期策略(IsPasswordExpired 对 nil 视为不过期)对它们失效。
	pwdUpdatedAt := time.Now()

	entities := []sysModel.SysUser{
		{
			UUID:              uuid.New(),
			UserName:          "super",
			Password:          adminPassword,
			PasswordUpdatedAt: &pwdUpdatedAt,
			NickName:          "超级管理员",
			Phonenumber:       "13666666666",
			Email:             "super@example.com",
			Avatar:            "https://vue3.baiwumm.com/static/image/2024-07/cc9e77ee-cf84-48e8-a9d0-dc3e9d21224c.jpeg",
			RoleId:            superRoleID, // 主角色(登录链路 claims 用)
		},
		{
			UUID:              uuid.New(),
			UserName:          "admin",
			Password:          adminPassword,
			PasswordUpdatedAt: &pwdUpdatedAt,
			NickName:          "系统管理员",
			Phonenumber:       "13777777777",
			Email:             "admin@example.com",
			Avatar:            "https://wpimg.wallstcn.com/f778738c-e4f8-4870-b634-56703b4acafe.gif?imageView2/1/w/80/h/80",
			RoleId:            adminRoleID, // 主角色(登录链路 claims 用)
		},
		{
			UUID:              uuid.New(),
			UserName:          "user",
			Password:          adminPassword,
			PasswordUpdatedAt: &pwdUpdatedAt,
			NickName:          "普通用户",
			Phonenumber:       "13888888888",
			Email:             "user@example.com",
			Avatar:            "https://img2.baidu.com/it/u=1978192862,2048448374&fm=253&fmt=auto&app=138&f=JPEG?w=504&h=500",
			RoleId:            userRoleID, // 主角色(登录链路 claims 用)
		},
	}
	if err = db.Create(&entities).Error; err != nil {
		return ctx, errors.Wrapf(err, "%s表数据初始化失败!", sysModel.SysUser{}.TableName())
	}
	// Create 后 entities[0].UserId 由雪花回调回填

	// 写显式连接表 sys_user_role,使"主角色(RoleId)"与"多角色(Roles many2many)"一致;
	// 与 SysUser.Roles 的 many2many tag 同表,后续 service 层可经 Association 或本 struct 直接操作。
	// 三个用户都写主角色关联(原实现仅写 entities[0]=super,admin/user 的 sys_user_role 缺失)
	userRoleLinks := []sysModel.SysUserRole{
		{SysUserId: entities[0].UserId, SysRoleId: superRoleID},
		{SysUserId: entities[1].UserId, SysRoleId: adminRoleID},
		{SysUserId: entities[2].UserId, SysRoleId: userRoleID},
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
