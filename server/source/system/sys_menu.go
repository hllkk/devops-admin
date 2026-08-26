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
		// 个人中心首页(顶层单级 C·blank 布局·隐藏菜单):AI 网关身份/用量门户页,走 layout.blank
		// 无全局 header/sider/tab。Component 含 layout.blank 触发 blank 布局,view 段由 routeKey(home)
		// 规范化为 home,转换后 = layout.blank$view.home(对齐前端 routes.ts)。Visible=1 对应前端 hideInMenu:
		// home 不进侧边栏,经头像下拉/URL 进入,故授权后全员可访问(见 sys_role_menu.go user 授权)。
		// Module 留空:home 是跨模块全局页(同 user-center),对齐前端 routes.ts 无 meta.module。
		// resolveModuleFromRoute 返回 null → global route(所有模块可见),且不更新 currentModule,
		// 避免从 server/gateway 模块进 home 时 currentModule 跳到 admin 导致 tearing。
		{
			ParentId:  0,
			MenuName:  "route.home",
			MenuType:  "C",
			Path:      "home",
			Component: "layout.blank$view.home",
			Icon:      "mdi:card-account-details-outline",
			Visible:   "1",
			OrderNum:  0,
		},
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
			ApiPrefix: "/timedTask, /timedTask/*",
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
		// 业务模块占位首页(顶层单级 C,对齐前端 views 的 _server/_gateway 占位页)
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
		// ── AI 网关模块:看板页 + 模型目录 + 密钥顶层单页(一级路由,path 单段以符合 elegant first-level 约束) ──
		{
			ParentId:  0,
			MenuName:  "route.gateway",
			MenuType:  "C",
			Path:      "gateway",
			ApiPrefix: "/gateway/dashboard, /gateway/dashboard/*",
			Component: "_gateway/gateway/index",
			Icon:      "mdi:view-dashboard-outline",
			Visible:   "0",
			OrderNum:  7,
			Module:    "gateway",
		},
		{
			ParentId:  0,
			MenuName:  "route.ai-key",
			MenuType:  "C",
			Path:      "ai-key",
			ApiPrefix: "/gateway/ai-key, /gateway/ai-key/*",
			Component: "_gateway/ai-key/index",
			Icon:      "mdi:key-variant",
			Visible:   "0",
			OrderNum:  8,
			Module:    "gateway",
		},
		{
			ParentId:  0,
			MenuName:  "route.models",
			MenuType:  "M",
			Path:      "models",
			Component: "Layout",
			Icon:      "mdi:brain",
			Visible:   "0",
			OrderNum:  9,
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
	// ApiPrefix 为该菜单对应后端接口的 casbin 策略 obj(逗号分隔多个 pattern,配合 keyMatch2 通配);
	// 角色授权菜单时由 syncRoleCasbinPolicy 推导为该角色的接口权限。按钮(F)不填,其接口由父菜单通配覆盖。
	childMenus := []sysModel.SysMenu{
		{
			ParentId:  menuNameMap["route.system"],
			MenuName:  "route.system_user",
			MenuType:  "C",
			Path:      "system/user",
			ApiPrefix: "/system/user, /system/user/*",
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
			ApiPrefix: "/system/role, /system/role/*",
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
			ApiPrefix: "/system/menu, /system/menu/*",
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
			ApiPrefix: "/system/dept, /system/dept/*",
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
			ApiPrefix: "/system/post, /system/post/*",
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
			ApiPrefix: "/system/dict/type, /system/dict/type/*, /system/dict/data, /system/dict/data/*",
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
			ApiPrefix: "/system/notice, /system/notice/*",
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
			ApiPrefix: "/system/setting, /system/setting/*",
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
			ApiPrefix: "/log/loginlog, /log/loginlog/*",
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
			ApiPrefix: "/log/operlog, /log/operlog/*",
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
			ApiPrefix: "/log/sysError, /log/sysError/*",
			Component: "_admin/log/errorlog/index",
			Icon:      "fluent:clipboard-error-24-regular",
			Visible:   "0",
			OrderNum:  3,
			Module:    "admin",
		},
		// ── AI 网关(密钥升顶层单页;模型目录下供应商/凭证/模型三页按链路排序;ApiPrefix 沿用后端接口前缀,与菜单 Path 解耦) ──
		{
			ParentId:  menuNameMap["route.models"],
			MenuName:  "route.models_provider",
			MenuType:  "C",
			Path:      "models/provider",
			ApiPrefix: "/gateway/provider, /gateway/provider/*",
			Component: "_gateway/models/provider/index",
			Icon:      "mdi:store-outline",
			Visible:   "0",
			OrderNum:  1,
			Module:    "gateway",
		},
		{
			ParentId:  menuNameMap["route.models"],
			MenuName:  "route.models_credential",
			MenuType:  "C",
			Path:      "models/credential",
			ApiPrefix: "/gateway/credential, /gateway/credential/*",
			Component: "_gateway/models/credential/index",
			Icon:      "mdi:key-chain-variant",
			Visible:   "0",
			OrderNum:  2,
			Module:    "gateway",
		},
		{
			ParentId:  menuNameMap["route.models"],
			MenuName:  "route.models_model",
			MenuType:  "C",
			Path:      "models/model",
			ApiPrefix: "/gateway/model, /gateway/model/*",
			Component: "_gateway/models/model/index",
			Icon:      "mdi:brain",
			Visible:   "0",
			OrderNum:  3,
			Module:    "gateway",
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
