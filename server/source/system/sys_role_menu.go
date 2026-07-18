package system

import (
	"context"

	sysModel "github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 依赖 sys_menu 与 sys_role,须排在两者之后;用 +1 接入单链,避免相加写法制造撞号。
const initOrderRoleMenu = initOrderUser + 1

type initRoleMenu struct{}

// auto run
func init() {
	system.RegisterInit(initOrderRoleMenu, &initRoleMenu{})
}

func (i *initRoleMenu) InitializerName() string {
	return (&sysModel.SysRoleMenu{}).TableName()
}

func (i *initRoleMenu) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&sysModel.SysRoleMenu{})
}

func (i *initRoleMenu) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&sysModel.SysRoleMenu{})
}

func (i *initRoleMenu) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	// 探针收紧到 super 角色的菜单关联:全局 Count>0 会被任意残留记录误判为已完成,
	// super 角色必然初始化且关联全部菜单,以其是否已授权作为完成标志。
	var superRole sysModel.SysRole
	if errors.Is(db.Where("role_key = ?", "super").First(&superRole).Error, gorm.ErrRecordNotFound) {
		return false
	}
	var count int64
	db.Model(&sysModel.SysRoleMenu{}).Where("sys_role_id = ?", superRole.RoleId).Count(&count)
	return count > 0
}

func (i *initRoleMenu) InitializeData(ctx context.Context) (next context.Context, err error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}

	// 角色先于角色菜单初始化(initOrderRole < initOrderRoleMenu),
	// 且 sys_role.InitializeData 已将带雪花 ID 的角色切片回填到 ctx。
	roleEntities, ok := ctx.Value(sysModel.SysRole{}.TableName()).([]sysModel.SysRole)
	if !ok {
		return ctx, system.ErrMissingDependentContext
	}

	// 菜单同样先于角色菜单初始化(initOrderMenu < initOrderRoleMenu),
	// 且 sys_menu.InitializeData 已将带雪花 ID 的菜单切片回填到 ctx。
	menuEntities, ok := ctx.Value(sysModel.SysMenu{}.TableName()).([]sysModel.SysMenu)
	if !ok {
		return ctx, system.ErrMissingDependentContext
	}

	// 查找 super 和 admin 角色的雪花 ID
	var superRoleID int64
	var adminRoleID int64
	var userRoleID int64
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
	if superRoleID == 0 {
		return ctx, errors.New("未找到 super 角色,请确认 sys_role 初始化已先执行")
	}

	// ── super/admin: 分配所有菜单权限(super 虽有 SuperAdmin 标志可绕过权限检查,
	//    但写入 role_menu 保持数据一致性; admin DataScope=1 全部,也需要显式菜单权限) ──
	superLinks := make([]sysModel.SysRoleMenu, 0, len(menuEntities))
	adminLinks := make([]sysModel.SysRoleMenu, 0, len(menuEntities))
	for _, m := range menuEntities {
		superLinks = append(superLinks, sysModel.SysRoleMenu{SysRoleId: superRoleID, SysMenuId: m.MenuId})
		adminLinks = append(adminLinks, sysModel.SysRoleMenu{SysRoleId: adminRoleID, SysMenuId: m.MenuId})
	}

	// ── user: 仅分配首页(dashboard)菜单权限 ──
	var adminMenuID int64 // route.admin 的 MenuId
	for _, m := range menuEntities {
		if m.MenuName == "route.admin" {
			adminMenuID = m.MenuId
			break
		}
	}
	userLinks := []sysModel.SysRoleMenu{
		{SysRoleId: userRoleID, SysMenuId: adminMenuID},
	}

	// 合并所有角色菜单关联并批量写入
	allLinks := make([]sysModel.SysRoleMenu, 0, len(superLinks)+len(adminLinks)+len(userLinks))
	allLinks = append(allLinks, superLinks...)
	allLinks = append(allLinks, adminLinks...)
	allLinks = append(allLinks, userLinks...)

	// 联合主键 (sys_role_id, sys_menu_id) 上 OnConflict DoNothing,使重跑幂等、不产生重复关联。
	if err = db.Clauses(clause.OnConflict{DoNothing: true}).Create(&allLinks).Error; err != nil {
		return ctx, errors.Wrapf(err, "%s表数据初始化失败!", (&sysModel.SysRoleMenu{}).TableName())
	}

	next = context.WithValue(ctx, i.InitializerName(), allLinks)
	return next, nil
}
