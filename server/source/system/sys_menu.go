package system

import (
	"context"
	"errors"
	"fmt"

	"github.com/hllkk/devops-admin/server/model/common"
	"github.com/hllkk/devops-admin/server/model/system"
	sysSvc "github.com/hllkk/devops-admin/server/service/system"
	"gorm.io/gorm"
)

const initOrderMenu = sysSvc.InitOrderSystem + 3

type initMenu struct{}

func init() { sysSvc.RegisterInit(initOrderMenu, &initMenu{}) }

func (i *initMenu) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, sysSvc.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&system.SysMenu{})
}

func (i *initMenu) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&system.SysMenu{})
}

func (i *initMenu) InitializerName() string { return system.SysMenu{}.TableName() }

// apis 构造 C 菜单挂载的 API 资源（方式 B：menu.apis 内嵌），path 去掉 RouterPrefix，
// 与 middleware/casbin_rbac.go 计算的 obj 一致。
func apis(pairs ...[2]string) common.JSONSlice[system.MenuApi] {
	out := make(common.JSONSlice[system.MenuApi], 0, len(pairs))
	for _, p := range pairs {
		out = append(out, system.MenuApi{Path: p[0], Method: p[1]})
	}
	return out
}

func (i *initMenu) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, sysSvc.ErrMissingDBContext
	}
	entities := []system.SysMenu{
		// M 目录
		{MenuId: 100, ParentId: 0, MenuName: "系统管理", MenuType: "M", Path: "/system", Icon: "ion:settings-outline", OrderNum: 1, Visible: "0", Status: "0"},
		// C 用户管理 + 6 F
		{MenuId: 1100, ParentId: 100, MenuName: "用户管理", MenuType: "C", Path: "user", Component: "_admin/system/user/index", Perms: "system:user:list", OrderNum: 1, Visible: "0", Status: "0",
			Apis: apis([2]string{"/system/user/list", "GET"}, [2]string{"/system/user", "POST"}, [2]string{"/system/user", "PUT"}, [2]string{"/system/user/:id", "DELETE"})},
		{MenuId: 1101, ParentId: 1100, MenuName: "新增", MenuType: "F", Perms: "system:user:add", OrderNum: 1, Status: "0"},
		{MenuId: 1102, ParentId: 1100, MenuName: "修改", MenuType: "F", Perms: "system:user:edit", OrderNum: 2, Status: "0"},
		{MenuId: 1103, ParentId: 1100, MenuName: "删除", MenuType: "F", Perms: "system:user:remove", OrderNum: 3, Status: "0"},
		{MenuId: 1104, ParentId: 1100, MenuName: "导出", MenuType: "F", Perms: "system:user:export", OrderNum: 4, Status: "0"},
		{MenuId: 1105, ParentId: 1100, MenuName: "导入", MenuType: "F", Perms: "system:user:import", OrderNum: 5, Status: "0"},
		{MenuId: 1106, ParentId: 1100, MenuName: "重置密码", MenuType: "F", Perms: "system:user:resetPwd", OrderNum: 6, Status: "0"},
		// C 角色管理 + 4 F
		{MenuId: 1200, ParentId: 100, MenuName: "角色管理", MenuType: "C", Path: "role", Component: "_admin/system/role/index", Perms: "system:role:list", OrderNum: 2, Visible: "0", Status: "0",
			Apis: apis([2]string{"/system/role/list", "GET"}, [2]string{"/system/role", "POST"}, [2]string{"/system/role", "PUT"}, [2]string{"/system/role/:id", "DELETE"})},
		{MenuId: 1201, ParentId: 1200, MenuName: "新增", MenuType: "F", Perms: "system:role:add", OrderNum: 1, Status: "0"},
		{MenuId: 1202, ParentId: 1200, MenuName: "修改", MenuType: "F", Perms: "system:role:edit", OrderNum: 2, Status: "0"},
		{MenuId: 1203, ParentId: 1200, MenuName: "删除", MenuType: "F", Perms: "system:role:remove", OrderNum: 3, Status: "0"},
		{MenuId: 1204, ParentId: 1200, MenuName: "导出", MenuType: "F", Perms: "system:role:export", OrderNum: 4, Status: "0"},
		// C 菜单管理 + 3 F
		{MenuId: 1300, ParentId: 100, MenuName: "菜单管理", MenuType: "C", Path: "menu", Component: "_admin/system/menu/index", Perms: "system:menu:list", OrderNum: 3, Visible: "0", Status: "0",
			Apis: apis([2]string{"/system/menu/list", "GET"}, [2]string{"/system/menu", "POST"}, [2]string{"/system/menu", "PUT"}, [2]string{"/system/menu/:id", "DELETE"})},
		{MenuId: 1301, ParentId: 1300, MenuName: "新增", MenuType: "F", Perms: "system:menu:add", OrderNum: 1, Status: "0"},
		{MenuId: 1302, ParentId: 1300, MenuName: "修改", MenuType: "F", Perms: "system:menu:edit", OrderNum: 2, Status: "0"},
		{MenuId: 1303, ParentId: 1300, MenuName: "删除", MenuType: "F", Perms: "system:menu:remove", OrderNum: 3, Status: "0"},
		// C 部门管理 + 3 F
		{MenuId: 1400, ParentId: 100, MenuName: "部门管理", MenuType: "C", Path: "dept", Component: "_admin/system/dept/index", Perms: "system:dept:list", OrderNum: 4, Visible: "0", Status: "0",
			Apis: apis([2]string{"/system/dept/list", "GET"}, [2]string{"/system/dept", "POST"}, [2]string{"/system/dept", "PUT"}, [2]string{"/system/dept/:id", "DELETE"})},
		{MenuId: 1401, ParentId: 1400, MenuName: "新增", MenuType: "F", Perms: "system:dept:add", OrderNum: 1, Status: "0"},
		{MenuId: 1402, ParentId: 1400, MenuName: "修改", MenuType: "F", Perms: "system:dept:edit", OrderNum: 2, Status: "0"},
		{MenuId: 1403, ParentId: 1400, MenuName: "删除", MenuType: "F", Perms: "system:dept:remove", OrderNum: 3, Status: "0"},
		// C 岗位管理 + 4 F
		{MenuId: 1500, ParentId: 100, MenuName: "岗位管理", MenuType: "C", Path: "post", Component: "_admin/system/post/index", Perms: "system:post:list", OrderNum: 5, Visible: "0", Status: "0",
			Apis: apis([2]string{"/system/post/list", "GET"}, [2]string{"/system/post", "POST"}, [2]string{"/system/post", "PUT"}, [2]string{"/system/post/:id", "DELETE"})},
		{MenuId: 1501, ParentId: 1500, MenuName: "新增", MenuType: "F", Perms: "system:post:add", OrderNum: 1, Status: "0"},
		{MenuId: 1502, ParentId: 1500, MenuName: "修改", MenuType: "F", Perms: "system:post:edit", OrderNum: 2, Status: "0"},
		{MenuId: 1503, ParentId: 1500, MenuName: "删除", MenuType: "F", Perms: "system:post:remove", OrderNum: 3, Status: "0"},
		{MenuId: 1504, ParentId: 1500, MenuName: "导出", MenuType: "F", Perms: "system:post:export", OrderNum: 4, Status: "0"},
		// C 系统设置 + 1 F
		{MenuId: 1600, ParentId: 100, MenuName: "系统设置", MenuType: "C", Path: "setting", Component: "_admin/system/setting/index", Perms: "system:setting:list", OrderNum: 6, Visible: "0", Status: "0",
			Apis: apis([2]string{"/system/setting", "GET"}, [2]string{"/system/setting/public", "GET"}, [2]string{"/system/setting", "PUT"})},
		{MenuId: 1601, ParentId: 1600, MenuName: "保存", MenuType: "F", Perms: "system:setting:save", OrderNum: 1, Status: "0"},
	}
	if err := db.Create(&entities).Error; err != nil {
		return ctx, fmt.Errorf("%s 表数据初始化失败: %w", i.InitializerName(), err)
	}
	return context.WithValue(ctx, i.InitializerName(), entities), nil
}

func (i *initMenu) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return !errors.Is(db.Where("menu_id = ?", 1500).First(&system.SysMenu{}).Error, gorm.ErrRecordNotFound)
}
