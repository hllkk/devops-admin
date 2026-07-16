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
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return ctx, err
	}
	return ctx, db.AutoMigrate(&system.SysMenu{})
}

func (i *initMenu) TableCreated(ctx context.Context) bool {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
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
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return ctx, err
	}
	entities := []system.SysMenu{
		// ===== 顶级：仪表盘 admin（单页，无子菜单，前端 /admin）=====
		{MenuId: 1, ParentId: 0, MenuName: "仪表盘", MenuType: "C", Path: "/admin", Component: "_admin/admin/index", Icon: "mdi:monitor-dashboard", OrderNum: 1, Visible: "0", Status: "0"},

		// ===== 顶级：系统管理 system（icon/order 对齐前端 routes.ts）=====
		{MenuId: 100, ParentId: 0, MenuName: "系统管理", MenuType: "M", Path: "/system", Icon: "carbon:cloud-service-management", OrderNum: 2, Visible: "0", Status: "0"},
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
		// C 字典管理 + 5 F（后端 API 尚未实现，apis 按 RuoYi 规范预填，落地后 casbin 自动生效）
		{MenuId: 1700, ParentId: 100, MenuName: "字典管理", MenuType: "C", Path: "dict", Component: "_admin/system/dict/index", Perms: "system:dict:list", Icon: "menu-dict", OrderNum: 6, Visible: "0", Status: "0",
			Apis: apis([2]string{"/system/dict/type/list", "GET"}, [2]string{"/system/dict/type", "POST"}, [2]string{"/system/dict/type", "PUT"}, [2]string{"/system/dict/type/:ids", "DELETE"})},
		{MenuId: 1701, ParentId: 1700, MenuName: "新增", MenuType: "F", Perms: "system:dict:add", OrderNum: 1, Status: "0"},
		{MenuId: 1702, ParentId: 1700, MenuName: "修改", MenuType: "F", Perms: "system:dict:edit", OrderNum: 2, Status: "0"},
		{MenuId: 1703, ParentId: 1700, MenuName: "删除", MenuType: "F", Perms: "system:dict:remove", OrderNum: 3, Status: "0"},
		{MenuId: 1704, ParentId: 1700, MenuName: "导出", MenuType: "F", Perms: "system:dict:export", OrderNum: 4, Status: "0"},
		{MenuId: 1705, ParentId: 1700, MenuName: "刷新缓存", MenuType: "F", Perms: "system:dict:refreshCache", OrderNum: 5, Status: "0"},
		// C 通知公告 + 3 F（后端 API 尚未实现，apis 按 RuoYi 规范预填）
		{MenuId: 1800, ParentId: 100, MenuName: "通知公告", MenuType: "C", Path: "notice", Component: "_admin/system/notice/index", Perms: "system:notice:list", Icon: "carbon:notification", OrderNum: 7, Visible: "0", Status: "0",
			Apis: apis([2]string{"/system/notice/list", "GET"}, [2]string{"/system/notice", "POST"}, [2]string{"/system/notice", "PUT"}, [2]string{"/system/notice/:ids", "DELETE"})},
		{MenuId: 1801, ParentId: 1800, MenuName: "新增", MenuType: "F", Perms: "system:notice:add", OrderNum: 1, Status: "0"},
		{MenuId: 1802, ParentId: 1800, MenuName: "修改", MenuType: "F", Perms: "system:notice:edit", OrderNum: 2, Status: "0"},
		{MenuId: 1803, ParentId: 1800, MenuName: "删除", MenuType: "F", Perms: "system:notice:remove", OrderNum: 3, Status: "0"},
		// C 系统设置 + 1 F
		{MenuId: 1600, ParentId: 100, MenuName: "系统设置", MenuType: "C", Path: "setting", Component: "_admin/system/setting/index", Perms: "system:setting:list", OrderNum: 8, Visible: "0", Status: "0",
			Apis: apis([2]string{"/system/setting", "GET"}, [2]string{"/system/setting/public", "GET"}, [2]string{"/system/setting", "PUT"})},
		{MenuId: 1601, ParentId: 1600, MenuName: "保存", MenuType: "F", Perms: "system:setting:save", OrderNum: 1, Status: "0"},

		// ===== 顶级：日志管理 log（前端 /log）=====
		{MenuId: 200, ParentId: 0, MenuName: "日志管理", MenuType: "M", Path: "/log", Icon: "menu-log", OrderNum: 3, Visible: "0", Status: "0"},
		// C 登录日志 + 3 F（后端 API 已实现，apis 为真实路径）
		{MenuId: 2100, ParentId: 200, MenuName: "登录日志", MenuType: "C", Path: "loginlog", Component: "_admin/log/loginlog/index", Perms: "log:loginlog:list", Icon: "menu-login_log", OrderNum: 1, Visible: "0", Status: "0",
			Apis: apis([2]string{"/log/loginlog/list", "GET"}, [2]string{"/log/loginlog/:action", "DELETE"}, [2]string{"/log/loginlog/unlock/:username", "GET"})},
		{MenuId: 2101, ParentId: 2100, MenuName: "删除", MenuType: "F", Perms: "log:loginlog:remove", OrderNum: 1, Status: "0"},
		{MenuId: 2102, ParentId: 2100, MenuName: "清空", MenuType: "F", Perms: "log:loginlog:clean", OrderNum: 2, Status: "0"},
		{MenuId: 2103, ParentId: 2100, MenuName: "解锁", MenuType: "F", Perms: "log:loginlog:unlock", OrderNum: 3, Status: "0"},
		// C 操作日志 + 3 F（后端 API 尚未实现，apis 按 RuoYi 规范预填）
		{MenuId: 2200, ParentId: 200, MenuName: "操作日志", MenuType: "C", Path: "operlog", Component: "_admin/log/operlog/index", Perms: "log:operlog:list", Icon: "menu-operate_log", OrderNum: 2, Visible: "0", Status: "0",
			Apis: apis([2]string{"/log/operlog/list", "GET"}, [2]string{"/log/operlog/:action", "DELETE"})},
		{MenuId: 2201, ParentId: 2200, MenuName: "删除", MenuType: "F", Perms: "log:operlog:remove", OrderNum: 1, Status: "0"},
		{MenuId: 2202, ParentId: 2200, MenuName: "清空", MenuType: "F", Perms: "log:operlog:clean", OrderNum: 2, Status: "0"},
		{MenuId: 2203, ParentId: 2200, MenuName: "导出", MenuType: "F", Perms: "log:operlog:export", OrderNum: 3, Status: "0"},
	}
	if err := db.Create(&entities).Error; err != nil {
		return ctx, fmt.Errorf("%s 表数据初始化失败: %w", i.InitializerName(), err)
	}
	return context.WithValue(ctx, i.InitializerName(), entities), nil
}

func (i *initMenu) DataInserted(ctx context.Context) bool {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return false
	}
	return !errors.Is(db.Where("menu_id = ?", 1500).First(&system.SysMenu{}).Error, gorm.ErrRecordNotFound)
}
