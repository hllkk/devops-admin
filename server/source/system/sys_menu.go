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
	// 检查根级和子级两层哨兵记录,防止部分初始化误判为完整
	if errors.Is(db.Where("path = ?", "admin").
		First(&sysModel.SysMenu{}).Error, gorm.ErrRecordNotFound) {
		return false
	}
	if errors.Is(db.Where("path = ?", "system/user").
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

	// 根级菜单: M=目录(有子菜单) / C=菜单(实际页面)
	rootMenus := []sysModel.SysMenu{
		{
			ParentId:  0,
			MenuName:  "route.admin",
			MenuType:  "C",
			Path:      "admin",
			Component: "_admin/admin/index",
			Icon:      "mdi:monitor-dashboard",
			Visible:   "0",
			OrderNum:  1,
			Module:    "admin",
		},
		{
			ParentId:  0,
			MenuName:  "route.system",
			MenuType:  "M",
			Path:      "system",
			Component: "Layout",
			Icon:      "carbon:cloud-service-management",
			Visible:   "0",
			OrderNum:  2,
			Remark:    "系统管理目录",
			Module:    "admin",
		},
		{
			ParentId:  0,
			MenuName:  "route.timer",
			MenuType:  "C",
			Path:      "timer",
			Component: "Layout",
			Icon:      "fluent:clock-24-regular",
			Visible:   "0",
			OrderNum:  3,
			Module:    "admin",
		},
		{
			ParentId:  0,
			MenuName:  "route.log",
			MenuType:  "M",
			Path:      "log",
			Component: "Layout",
			Icon:      "local-icon-log",
			Visible:   "0",
			OrderNum:  4,
			Module:    "admin",
		},
		// 业务模块占位首页(顶层单级 C,对齐前端 views 的 _disk/_server/_gateway 占位页)
		{
			ParentId:  0,
			MenuName:  "route.disk",
			MenuType:  "C",
			Path:      "disk",
			Component: "layout.disk$view._disk/disk",
			Icon:      "mdi:harddisk",
			Visible:   "0",
			OrderNum:  5,
			Module:    "disk",
		},
		{
			ParentId:  0,
			MenuName:  "route.server",
			MenuType:  "C",
			Path:      "server",
			Component: "_server/server/index",
			Icon:      "mdi:server-network",
			Visible:   "0",
			OrderNum:  6,
			Module:    "server",
		},
		{
			ParentId:  0,
			MenuName:  "route.gateway",
			MenuType:  "C",
			Path:      "gateway",
			Component: "_gateway/gateway/index",
			Icon:      "mdi:robot-outline",
			Visible:   "0",
			OrderNum:  7,
			Module:    "gateway",
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

	// 子菜单(MenuType=C:有 Component 的实际页面,非目录节点)
	childMenus := []sysModel.SysMenu{
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_user",
			MenuType:  "C",
			Path:      "system/user",
			Component: "_admin/system/user/index",
			Icon:      "carbon:user",
			Visible:   "0",
			OrderNum:  1,
			Module:    "admin",
		},
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_role",
			MenuType:  "C",
			Path:      "system/role",
			Component: "_admin/system/role/index",
			Icon:      "carbon-user-role",
			Visible:   "0",
			OrderNum:  2,
			Module:    "admin",
		},
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_menu",
			MenuType:  "C",
			Path:      "system/menu",
			Component: "_admin/system/menu/index",
			Icon:      "mingcute:list-ordered-line",
			Visible:   "0",
			OrderNum:  3,
			Module:    "admin",
		},
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_dept",
			MenuType:  "C",
			Path:      "system/dept",
			Component: "_admin/system/dept/index",
			Icon:      "local-icon-department",
			Visible:   "0",
			OrderNum:  4,
			Module:    "admin",
		},
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_post",
			MenuType:  "C",
			Path:      "system/post",
			Component: "_admin/system/post/index",
			Icon:      "local-icon-post",
			Visible:   "0",
			OrderNum:  5,
			Module:    "admin",
		},
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_dict",
			MenuType:  "C",
			Path:      "system/dict",
			Component: "_admin/system/dict/index",
			Icon:      "streamline-ultimate:book-open-bookmark",
			Visible:   "0",
			OrderNum:  6,
			Module:    "admin",
		},
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_notice",
			MenuType:  "C",
			Path:      "system/notice",
			Component: "_admin/system/notice/index",
			Icon:      "carbon:notification",
			Visible:   "0",
			OrderNum:  7,
			Module:    "admin",
		},
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_setting",
			MenuType:  "C",
			Path:      "system/setting",
			Component: "_admin/system/setting/index",
			Icon:      "carbon:settings",
			Visible:   "0",
			OrderNum:  8,
			Module:    "admin",
		},
		{
			ParentId:  menuNameMap["route.log"],
			MenuName:  "route.log_loginlog",
			MenuType:  "C",
			Path:      "log/loginlog",
			Component: "_admin/log/loginlog/index",
			Icon:      "local-icon-login_log",
			Visible:   "0",
			OrderNum:  1,
			Module:    "admin",
		},
		{
			ParentId:  menuNameMap["route.log"],
			MenuName:  "route.log_operlog",
			MenuType:  "C",
			Path:      "log/operlog",
			Component: "_admin/log/operlog/index",
			Icon:      "local-icon-operate_log",
			Visible:   "0",
			OrderNum:  2,
			Module:    "admin",
		},
		{
			ParentId:  menuNameMap["route.log"],
			MenuName:  "route.log_errorlog",
			MenuType:  "C",
			Path:      "log/errorlog",
			Component: "_admin/log/errorlog/index",
			Icon:      "fluent:clipboard-error-24-regular",
			Visible:   "0",
			OrderNum:  3,
			Module:    "admin",
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
		{ParentId: menuNameMap["route.system_user"], MenuName: "用户查询", MenuType: "F", Perms: "system:user:query", Icon: "#", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.system_user"], MenuName: "用户新增", MenuType: "F", Perms: "system:user:add", Icon: "#", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.system_user"], MenuName: "用户修改", MenuType: "F", Perms: "system:user:edit", Icon: "#", Visible: "0", OrderNum: 3},
		{ParentId: menuNameMap["route.system_user"], MenuName: "用户删除", MenuType: "F", Perms: "system:user:remove", Icon: "#", Visible: "0", OrderNum: 4},
		{ParentId: menuNameMap["route.system_user"], MenuName: "用户导出", MenuType: "F", Perms: "system:user:export", Icon: "#", Visible: "0", OrderNum: 5},
		{ParentId: menuNameMap["route.system_user"], MenuName: "导入用户", MenuType: "F", Perms: "system:user:import", Icon: "#", Visible: "0", OrderNum: 6},
		{ParentId: menuNameMap["route.system_user"], MenuName: "重置密码", MenuType: "F", Perms: "system:user:resetPwd", Icon: "#", Visible: "0", OrderNum: 7},
		// ── 角色管理 ──
		{ParentId: menuNameMap["route.system_role"], MenuName: "角色查询", MenuType: "F", Perms: "system:role:query", Icon: "#", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.system_role"], MenuName: "角色新增", MenuType: "F", Perms: "system:role:add", Icon: "#", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.system_role"], MenuName: "角色修改", MenuType: "F", Perms: "system:role:edit", Icon: "#", Visible: "0", OrderNum: 3},
		{ParentId: menuNameMap["route.system_role"], MenuName: "角色删除", MenuType: "F", Perms: "system:role:remove", Icon: "#", Visible: "0", OrderNum: 4},
		{ParentId: menuNameMap["route.system_role"], MenuName: "角色导出", MenuType: "F", Perms: "system:role:export", Icon: "#", Visible: "0", OrderNum: 5},
		// ── 菜单管理 ──
		{ParentId: menuNameMap["route.system_menu"], MenuName: "菜单查询", MenuType: "F", Perms: "system:menu:query", Icon: "#", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.system_menu"], MenuName: "菜单新增", MenuType: "F", Perms: "system:menu:add", Icon: "#", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.system_menu"], MenuName: "菜单修改", MenuType: "F", Perms: "system:menu:edit", Icon: "#", Visible: "0", OrderNum: 3},
		{ParentId: menuNameMap["route.system_menu"], MenuName: "菜单删除", MenuType: "F", Perms: "system:menu:remove", Icon: "#", Visible: "0", OrderNum: 4},
		// ── 部门管理 ──
		{ParentId: menuNameMap["route.system_dept"], MenuName: "部门查询", MenuType: "F", Perms: "system:dept:query", Icon: "#", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.system_dept"], MenuName: "部门新增", MenuType: "F", Perms: "system:dept:add", Icon: "#", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.system_dept"], MenuName: "部门修改", MenuType: "F", Perms: "system:dept:edit", Icon: "#", Visible: "0", OrderNum: 3},
		{ParentId: menuNameMap["route.system_dept"], MenuName: "部门删除", MenuType: "F", Perms: "system:dept:remove", Icon: "#", Visible: "0", OrderNum: 4},
		// ── 岗位管理 ──
		{ParentId: menuNameMap["route.system_post"], MenuName: "岗位查询", MenuType: "F", Perms: "system:post:query", Icon: "#", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.system_post"], MenuName: "岗位新增", MenuType: "F", Perms: "system:post:add", Icon: "#", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.system_post"], MenuName: "岗位修改", MenuType: "F", Perms: "system:post:edit", Icon: "#", Visible: "0", OrderNum: 3},
		{ParentId: menuNameMap["route.system_post"], MenuName: "岗位删除", MenuType: "F", Perms: "system:post:remove", Icon: "#", Visible: "0", OrderNum: 4},
		{ParentId: menuNameMap["route.system_post"], MenuName: "岗位导出", MenuType: "F", Perms: "system:post:export", Icon: "#", Visible: "0", OrderNum: 5},
		// ── 字典管理 ──
		{ParentId: menuNameMap["route.system_dict"], MenuName: "字典查询", MenuType: "F", Perms: "system:dict:query", Icon: "#", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.system_dict"], MenuName: "字典新增", MenuType: "F", Perms: "system:dict:add", Icon: "#", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.system_dict"], MenuName: "字典修改", MenuType: "F", Perms: "system:dict:edit", Icon: "#", Visible: "0", OrderNum: 3},
		{ParentId: menuNameMap["route.system_dict"], MenuName: "字典删除", MenuType: "F", Perms: "system:dict:remove", Icon: "#", Visible: "0", OrderNum: 4},
		{ParentId: menuNameMap["route.system_dict"], MenuName: "字典导出", MenuType: "F", Perms: "system:dict:export", Icon: "#", Visible: "0", OrderNum: 5},
		// ── 通知公告 ──
		{ParentId: menuNameMap["route.system_notice"], MenuName: "公告查询", MenuType: "F", Perms: "system:notice:query", Icon: "#", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.system_notice"], MenuName: "公告新增", MenuType: "F", Perms: "system:notice:add", Icon: "#", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.system_notice"], MenuName: "公告修改", MenuType: "F", Perms: "system:notice:edit", Icon: "#", Visible: "0", OrderNum: 3},
		{ParentId: menuNameMap["route.system_notice"], MenuName: "公告删除", MenuType: "F", Perms: "system:notice:remove", Icon: "#", Visible: "0", OrderNum: 4},
		// ── 系统设置 ──
		{ParentId: menuNameMap["route.system_setting"], MenuName: "设置查询", MenuType: "F", Perms: "system:setting:query", Icon: "#", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.system_setting"], MenuName: "设置保存", MenuType: "F", Perms: "system:setting:save", Icon: "#", Visible: "0", OrderNum: 2},
		// ── 登录日志 ──
		{ParentId: menuNameMap["route.log_loginlog"], MenuName: "日志查询", MenuType: "F", Perms: "log:loginlog:query", Icon: "#", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.log_loginlog"], MenuName: "日志删除", MenuType: "F", Perms: "log:loginlog:remove", Icon: "#", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.log_loginlog"], MenuName: "日志导出", MenuType: "F", Perms: "log:loginlog:export", Icon: "#", Visible: "0", OrderNum: 3},
		// ── 操作日志 ──
		{ParentId: menuNameMap["route.log_operlog"], MenuName: "日志查询", MenuType: "F", Perms: "monitor:operlog:query", Icon: "#", Visible: "0", OrderNum: 1},
		{ParentId: menuNameMap["route.log_operlog"], MenuName: "日志删除", MenuType: "F", Perms: "monitor:operlog:remove", Icon: "#", Visible: "0", OrderNum: 2},
		{ParentId: menuNameMap["route.log_operlog"], MenuName: "日志导出", MenuType: "F", Perms: "monitor:operlog:export", Icon: "#", Visible: "0", OrderNum: 3},
	}

	// 创建按钮权限
	if err = db.Create(&buttonMenus).Error; err != nil {
		return ctx, errors.Wrap(err, sysModel.SysMenu{}.TableName()+"按钮权限初始化失败!")
	}

	// 组合所有菜单作为返回结果(独立分配新切片,避免修改 rootMenus 底层数组)
	allEntities := make([]sysModel.SysMenu, 0, len(rootMenus)+len(childMenus)+len(buttonMenus))
	allEntities = append(allEntities, rootMenus...)
	allEntities = append(allEntities, childMenus...)
	allEntities = append(allEntities, buttonMenus...)

	next = context.WithValue(ctx, i.InitializerName(), allEntities)
	return next, nil
}
