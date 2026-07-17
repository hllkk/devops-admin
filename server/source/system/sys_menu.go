package system

import (
	"context"

	sysModel "github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const initOrderMenu = initOrderRole + 1

type initMenu struct{}

// auto run
func init() {
	system.RegisterInit(initOrderMenu, &initMenu{})
}

func (i *initMenu) InitializerName() string {
	return sysModel.SysMenu{}.TableName()
}

func (i *initMenu) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&sysModel.SysMenu{})
}

func (i *initMenu) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	m := db.Migrator()
	return m.HasTable(&sysModel.SysMenu{})
}

func (i *initMenu) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	if errors.Is(db.Where("path = ?", "/admin").
		First(&sysModel.SysMenu{}).Error, gorm.ErrRecordNotFound) {
		return false
	}
	return true
}

func (i *initMenu) InitializeData(ctx context.Context) (next context.Context, err error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}

	// 定义所有菜单(顶级分组)
	rootMenus := []sysModel.SysMenu{
		{
			ParentId:  0,
			MenuName:  "route.admin",
			MenuType:  "C",
			Path:      "/admin",
			Component: "layout.base$view.admin",
			Icon:      "mdi:monitor-dashboard",
			Visible:   "0",
			OrderNum:  1,
		},
		{
			ParentId:  0,
			MenuName:  "route.system",
			MenuType:  "M",
			Path:      "/system",
			Component: "layout.base",
			Icon:      "carbon:cloud-service-management",
			Visible:   "0",
			OrderNum:  2,
		},
		{
			ParentId:  0,
			MenuName:  "route.log",
			MenuType:  "M",
			Path:      "/log",
			Component: "layout.base",
			Icon:      "local-icon-menu-log",
			Visible:   "0",
			OrderNum:  3,
		},
	}

	// 先创建父级菜单（ParentId = 0 的菜单）
	if err := db.Create(&rootMenus).Error; err != nil {
		return ctx, errors.Wrap(err, sysModel.SysMenu{}.TableName()+"父级菜单初始化失败!")
	}

	// 建立菜单映射 - 通过Name查找已创建的菜单及其ID
	menuNameMap := make(map[string]int64)
	for _, menu := range rootMenus {
		menuNameMap[menu.MenuName] = menu.MenuId
	}

	childMenus := []sysModel.SysMenu{
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_user",
			MenuType:  "M",
			Path:      "/system/user",
			Component: "view.system_user",
			Icon:      "carbon:user",
			Visible:   "0",
			OrderNum:  1,
		},
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_role",
			MenuType:  "M",
			Path:      "/system/role",
			Component: "view.system_role",
			Icon:      "carbon-user-role",
			Visible:   "0",
			OrderNum:  2,
		},
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_menu",
			MenuType:  "M",
			Path:      "/system/menu",
			Component: "view.system_menu",
			Icon:      "mingcute:list-ordered-line",
			Visible:   "0",
			OrderNum:  3,
		},
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_dept",
			MenuType:  "M",
			Path:      "/system/dept",
			Component: "view.system_dept",
			Icon:      "local-icon-menu-department",
			Visible:   "0",
			OrderNum:  4,
		},
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_post",
			MenuType:  "M",
			Path:      "/system/post",
			Component: "view.system_post",
			Icon:      "local-icon-menu-post",
			Visible:   "0",
			OrderNum:  5,
		},
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_dict",
			MenuType:  "M",
			Path:      "/system/dict",
			Component: "view.system_dict",
			Icon:      "local-icon-menu-dict",
			Visible:   "0",
			OrderNum:  6,
		},
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_notice",
			MenuType:  "M",
			Path:      "/system/notice",
			Component: "view.system_notice",
			Icon:      "carbon:notification",
			Visible:   "0",
			OrderNum:  7,
		},
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_setting",
			MenuType:  "M",
			Path:      "/system/setting",
			Component: "view.system_setting",
			Icon:      "carbon:settings",
			Visible:   "0",
			OrderNum:  8,
		},
		{
			ParentId:  menuNameMap["route.log"],
			MenuName:  "route.log_loginlog",
			MenuType:  "M",
			Path:      "/log/loginlog",
			Component: "view.log_loginlog",
			Icon:      "local-icon-menu-login_log",
			Visible:   "0",
			OrderNum:  1,
		},
		{
			ParentId:  menuNameMap["route.log"],
			MenuName:  "route.log_operlog",
			MenuType:  "M",
			Path:      "/log/operlog",
			Component: "view.log_operlog",
			Icon:      "local-icon-menu-operate_log",
			Visible:   "0",
			OrderNum:  2,
		},
	}
	// 创建子菜单
	if err = db.Create(&childMenus).Error; err != nil {
		return ctx, errors.Wrap(err, sysModel.SysMenu{}.TableName()+"子菜单初始化失败!")
	}

	// 将子菜单也加入映射，供按钮 ParentId 引用
	for _, menu := range childMenus {
		menuNameMap[menu.MenuName] = menu.MenuId
	}

	// 定义所有按钮权限（MenuType = F）
	buttonMenus := []sysModel.SysMenu{
		// ── 用户管理 ──
		{ParentId: menuNameMap["route.system_user"], MenuName: "用户查询", MenuType: "F", Perms: "system:user:query", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.system_user"], MenuName: "用户新增", MenuType: "F", Perms: "system:user:add", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.system_user"], MenuName: "用户修改", MenuType: "F", Perms: "system:user:edit", Visible: "0", OrderNum: 3},
		{ParentId: menuNameMap["route.system_user"], MenuName: "用户删除", MenuType: "F", Perms: "system:user:remove", Visible: "0", OrderNum: 4},
		{ParentId: menuNameMap["route.system_user"], MenuName: "用户导出", MenuType: "F", Perms: "system:user:export", Visible: "0", OrderNum: 5},
		{ParentId: menuNameMap["route.system_user"], MenuName: "导入用户", MenuType: "F", Perms: "system:user:import", Visible: "0", OrderNum: 6},
		{ParentId: menuNameMap["route.system_user"], MenuName: "重置密码", MenuType: "F", Perms: "system:user:resetPwd", Visible: "0", OrderNum: 7},
		// ── 角色管理 ──
		{ParentId: menuNameMap["route.system_role"], MenuName: "角色查询", MenuType: "F", Perms: "system:role:query", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.system_role"], MenuName: "角色新增", MenuType: "F", Perms: "system:role:add", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.system_role"], MenuName: "角色修改", MenuType: "F", Perms: "system:role:edit", Visible: "0", OrderNum: 3},
		{ParentId: menuNameMap["route.system_role"], MenuName: "角色删除", MenuType: "F", Perms: "system:role:remove", Visible: "0", OrderNum: 4},
		{ParentId: menuNameMap["route.system_role"], MenuName: "角色导出", MenuType: "F", Perms: "system:role:export", Visible: "0", OrderNum: 5},
		// ── 菜单管理 ──
		{ParentId: menuNameMap["route.system_menu"], MenuName: "菜单查询", MenuType: "F", Perms: "system:menu:query", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.system_menu"], MenuName: "菜单新增", MenuType: "F", Perms: "system:menu:add", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.system_menu"], MenuName: "菜单修改", MenuType: "F", Perms: "system:menu:edit", Visible: "0", OrderNum: 3},
		{ParentId: menuNameMap["route.system_menu"], MenuName: "菜单删除", MenuType: "F", Perms: "system:menu:remove", Visible: "0", OrderNum: 4},
		// ── 部门管理 ──
		{ParentId: menuNameMap["route.system_dept"], MenuName: "部门查询", MenuType: "F", Perms: "system:dept:query", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.system_dept"], MenuName: "部门新增", MenuType: "F", Perms: "system:dept:add", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.system_dept"], MenuName: "部门修改", MenuType: "F", Perms: "system:dept:edit", Visible: "0", OrderNum: 3},
		{ParentId: menuNameMap["route.system_dept"], MenuName: "部门删除", MenuType: "F", Perms: "system:dept:remove", Visible: "0", OrderNum: 4},
		// ── 岗位管理 ──
		{ParentId: menuNameMap["route.system_post"], MenuName: "岗位查询", MenuType: "F", Perms: "system:post:query", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.system_post"], MenuName: "岗位新增", MenuType: "F", Perms: "system:post:add", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.system_post"], MenuName: "岗位修改", MenuType: "F", Perms: "system:post:edit", Visible: "0", OrderNum: 3},
		{ParentId: menuNameMap["route.system_post"], MenuName: "岗位删除", MenuType: "F", Perms: "system:post:remove", Visible: "0", OrderNum: 4},
		{ParentId: menuNameMap["route.system_post"], MenuName: "岗位导出", MenuType: "F", Perms: "system:post:export", Visible: "0", OrderNum: 5},
		// ── 字典管理 ──
		{ParentId: menuNameMap["route.system_dict"], MenuName: "字典查询", MenuType: "F", Perms: "system:dict:query", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.system_dict"], MenuName: "字典新增", MenuType: "F", Perms: "system:dict:add", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.system_dict"], MenuName: "字典修改", MenuType: "F", Perms: "system:dict:edit", Visible: "0", OrderNum: 3},
		{ParentId: menuNameMap["route.system_dict"], MenuName: "字典删除", MenuType: "F", Perms: "system:dict:remove", Visible: "0", OrderNum: 4},
		{ParentId: menuNameMap["route.system_dict"], MenuName: "字典导出", MenuType: "F", Perms: "system:dict:export", Visible: "0", OrderNum: 5},
		// ── 通知公告 ──
		{ParentId: menuNameMap["route.system_notice"], MenuName: "公告查询", MenuType: "F", Perms: "system:notice:query", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.system_notice"], MenuName: "公告新增", MenuType: "F", Perms: "system:notice:add", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.system_notice"], MenuName: "公告修改", MenuType: "F", Perms: "system:notice:edit", Visible: "0", OrderNum: 3},
		{ParentId: menuNameMap["route.system_notice"], MenuName: "公告删除", MenuType: "F", Perms: "system:notice:remove", Visible: "0", OrderNum: 4},
		// ── 系统设置 ──
		{ParentId: menuNameMap["route.system_setting"], MenuName: "设置查询", MenuType: "F", Perms: "system:setting:query", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.system_setting"], MenuName: "设置保存", MenuType: "F", Perms: "system:setting:save", Visible: "0", OrderNum: 2},
		// ── 登录日志 ──
		{ParentId: menuNameMap["route.log_loginlog"], MenuName: "日志查询", MenuType: "F", Perms: "log:loginlog:query", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.log_loginlog"], MenuName: "日志删除", MenuType: "F", Perms: "log:loginlog:remove", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.log_loginlog"], MenuName: "日志导出", MenuType: "F", Perms: "log:loginlog:export", Visible: "0", OrderNum: 3},
		// ── 操作日志 ──
		{ParentId: menuNameMap["route.log_operlog"], MenuName: "日志查询", MenuType: "F", Perms: "monitor:operlog:query", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.log_operlog"], MenuName: "日志删除", MenuType: "F", Perms: "monitor:operlog:remove", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.log_operlog"], MenuName: "日志导出", MenuType: "F", Perms: "monitor:operlog:export", Visible: "0", OrderNum: 3},
	}

	// 创建按钮权限
	if err = db.Create(&buttonMenus).Error; err != nil {
		return ctx, errors.Wrap(err, sysModel.SysMenu{}.TableName()+"按钮权限初始化失败!")
	}

	// 组合所有菜单作为返回结果
	allEntities := append(append(rootMenus, childMenus...), buttonMenus...)

	next = context.WithValue(ctx, i.InitializerName(), allEntities)
	return next, nil
}
