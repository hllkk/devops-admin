package system

import (
	"testing"

	"github.com/hllkk/devops-admin/server/model/system"
)

// findRoute 在 MenuRoute 树中按 name 查找节点(测试辅助)。
func findRoute(routes []system.MenuRoute, name string) *system.MenuRoute {
	for i := range routes {
		if routes[i].Name == name {
			return &routes[i]
		}
		if c := findRoute(routes[i].Children, name); c != nil {
			return c
		}
	}
	return nil
}

func TestRouteKey(t *testing.T) {
	cases := map[string]string{
		"system/user":   "system_user",
		"admin":         "admin",
		"/log/loginlog": "log_loginlog",
		"/system/":      "system",
		"":              "",
	}
	for in, want := range cases {
		if got := routeKey(in); got != want {
			t.Errorf("routeKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRoutePath(t *testing.T) {
	cases := map[string]string{
		"system/user": "/system/user",
		"admin":       "/admin",
		"/log/":       "/log",
		"":            "/",
	}
	for in, want := range cases {
		if got := routePath(in); got != want {
			t.Errorf("routePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveIcon(t *testing.T) {
	cases := []struct {
		in            string
		wantIcon      string
		wantLocalIcon string
	}{
		{"mdi:monitor-dashboard", "mdi:monitor-dashboard", ""},
		{"local-icon-log", "", "menu-log"},
		{"local-icon-department", "", "menu-department"},
		{"#", "", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		icon, local := resolveIcon(c.in)
		if icon != c.wantIcon || local != c.wantLocalIcon {
			t.Errorf("resolveIcon(%q) = (%q,%q), want (%q,%q)", c.in, icon, local, c.wantIcon, c.wantLocalIcon)
		}
	}
}

// TestMenusToRoutes 核心断言:初始化数据(RuoYi 风格)转换后与前端 routes.ts 静态生成一致。
// 覆盖:顶层单级(admin)、顶层多级目录(system)+子页(system_user)、顶层 M 无子(timer 按单级)、
// F 按钮过滤、Iconify/localIcon 分流、visible/isCache/isFrame 映射。
func TestMenusToRoutes(t *testing.T) {
	menus := []system.SysMenu{
		{MenuId: 1, ParentId: 0, MenuType: "C", MenuName: "route.admin", Path: "admin", Icon: "mdi:monitor-dashboard", Visible: "0", IsCache: "1", IsFrame: "1", OrderNum: 1, Module: "admin"},
		{MenuId: 2, ParentId: 0, MenuType: "M", MenuName: "route.system", Path: "system", Icon: "carbon:cloud-service-management", Visible: "0", IsCache: "1", IsFrame: "1", OrderNum: 2, Module: "admin"},
		{MenuId: 3, ParentId: 2, MenuType: "C", MenuName: "route.system_user", Path: "system/user", Icon: "carbon:user", Visible: "0", IsCache: "1", IsFrame: "1", OrderNum: 1, Module: "admin"},
		{MenuId: 4, ParentId: 0, MenuType: "M", MenuName: "route.timer", Path: "timer", Icon: "fluent:clock-24-regular", Visible: "0", IsCache: "1", IsFrame: "1", OrderNum: 3, Module: "admin"},
		{MenuId: 5, ParentId: 0, MenuType: "M", MenuName: "route.log", Path: "log", Icon: "local-icon-log", Visible: "0", IsCache: "1", IsFrame: "1", OrderNum: 4, Module: "admin"},
		{MenuId: 6, ParentId: 3, MenuType: "F", MenuName: "用户查询", Perms: "system:user:query", Icon: "#"}, // F 按钮应被过滤
		{MenuId: 7, ParentId: 0, MenuType: "C", MenuName: "route.disk", Path: "disk", Icon: "mdi:harddisk", Visible: "0", IsCache: "1", IsFrame: "1", OrderNum: 5, Module: "disk"},
		{MenuId: 8, ParentId: 0, MenuType: "C", MenuName: "route.server", Path: "server", Icon: "mdi:server-network", Visible: "0", IsCache: "1", IsFrame: "1", OrderNum: 6, Module: "server"},
		{MenuId: 9, ParentId: 0, MenuType: "C", MenuName: "route.gateway", Path: "gateway", Icon: "mdi:robot-outline", Visible: "0", IsCache: "1", IsFrame: "1", OrderNum: 7, Module: "gateway"},
		// home:顶层单级·blank 布局·隐藏菜单·跨模块全局页(AI 个人中心首页),无 module(global route)。
		// 验证 Component 含 layout.blank 触发 blank 布局、Visible=1->hideInMenu、Module 留空。
		{MenuId: 10, ParentId: 0, MenuType: "C", MenuName: "route.home", Path: "home", Component: "layout.blank$view.home", Icon: "mdi:card-account-details-outline", Visible: "1", IsCache: "1", IsFrame: "1", OrderNum: 0},
	}

	s := RouteService{}
	routes := s.menusToRoutes(menus)

	// 顶层 8 个(home/admin/system/timer/log/disk/server/gateway),F 按钮不进路由
	if want := 8; len(routes) != want {
		t.Fatalf("顶层路由数 = %d, want %d (F 按钮应被过滤)", len(routes), want)
	}

	// disk/server/gateway:顶层单级 -> layout.base$view.<key>(对齐前端 imports.ts 的 views key)
	for _, name := range []string{"disk", "server", "gateway"} {
		r := findRoute(routes, name)
		if r == nil {
			t.Fatalf("缺少 %s 路由", name)
		}
		if want := "layout.base$view." + name; r.Component != want {
			t.Errorf("%s.Component = %q, want %q", name, r.Component, want)
		}
	}

	// module 归属:SysMenu.Module -> meta.module(admin 模块菜单=admin;disk/server/gateway 各自)
	if m := findRoute(routes, "admin"); m.Meta.Module != "admin" {
		t.Errorf("admin.Module = %q, want admin", m.Meta.Module)
	}
	if m := findRoute(routes, "system_user"); m.Meta.Module != "admin" {
		t.Errorf("system_user.Module = %q, want admin", m.Meta.Module)
	}
	for _, c := range []struct{ name, mod string }{{"disk", "disk"}, {"server", "server"}, {"gateway", "gateway"}} {
		r := findRoute(routes, c.name)
		if r == nil {
			continue
		}
		if r.Meta.Module != c.mod {
			t.Errorf("%s.Module = %q, want %q", c.name, r.Meta.Module, c.mod)
		}
	}

	// admin:顶层单级 -> layout.base$view.admin
	admin := findRoute(routes, "admin")
	if admin == nil {
		t.Fatal("缺少 admin 路由")
	}
	if admin.Component != "layout.base$view.admin" {
		t.Errorf("admin.Component = %q, want layout.base$view.admin", admin.Component)
	}
	if admin.Path != "/admin" {
		t.Errorf("admin.Path = %q, want /admin", admin.Path)
	}
	if admin.Meta.Icon != "mdi:monitor-dashboard" || admin.Meta.LocalIcon != "" {
		t.Errorf("admin icon 分流错误: icon=%q localIcon=%q", admin.Meta.Icon, admin.Meta.LocalIcon)
	}
	if admin.Meta.I18nKey != "route.admin" {
		t.Errorf("admin.I18nKey = %q, want route.admin", admin.Meta.I18nKey)
	}
	if len(admin.Children) != 0 {
		t.Errorf("admin 不应有 children, got %d", len(admin.Children))
	}

	// home:顶层单级·blank 布局 -> layout.blank$view.home(对齐前端 routes.ts 的 home 路由)
	// Component 含 layout.blank 触发 blank 布局,view 段由 routeKey(home) 规范化;Visible=1 -> hideInMenu
	home := findRoute(routes, "home")
	if home == nil {
		t.Fatal("缺少 home 路由")
	}
	if home.Component != "layout.blank$view.home" {
		t.Errorf("home.Component = %q, want layout.blank$view.home", home.Component)
	}
	if home.Path != "/home" {
		t.Errorf("home.Path = %q, want /home", home.Path)
	}
	if home.Meta.I18nKey != "route.home" {
		t.Errorf("home.I18nKey = %q, want route.home", home.Meta.I18nKey)
	}
	if !home.Meta.HideInMenu {
		t.Error("home.Visible=1 应映射 hideInMenu=true")
	}
	if home.Meta.Module != "" {
		t.Errorf("home.Module = %q, want 空(global route 跨模块可见,对齐前端无 meta.module)", home.Meta.Module)
	}

	// system:顶层多级目录 -> layout.base,有 children
	sys := findRoute(routes, "system")
	if sys == nil {
		t.Fatal("缺少 system 路由")
	}
	if sys.Component != "layout.base" {
		t.Errorf("system.Component = %q, want layout.base", sys.Component)
	}
	if len(sys.Children) != 1 {
		t.Fatalf("system.Children 数 = %d, want 1", len(sys.Children))
	}

	// system_user:子级叶子 -> view.system_user
	su := findRoute(routes, "system_user")
	if su == nil {
		t.Fatal("缺少 system_user 路由")
	}
	if su.Component != "view.system_user" {
		t.Errorf("system_user.Component = %q, want view.system_user", su.Component)
	}
	if su.Path != "/system/user" {
		t.Errorf("system_user.Path = %q, want /system/user", su.Path)
	}
	// F 按钮(用户查询)不应作为 system_user 的 children
	if len(su.Children) != 0 {
		t.Errorf("system_user 不应有 children(F 应过滤), got %v", su.Children)
	}

	// timer:顶层 M 无子 -> 按单级处理 layout.base$view.timer(关键 case)
	timer := findRoute(routes, "timer")
	if timer == nil {
		t.Fatal("缺少 timer 路由")
	}
	if timer.Component != "layout.base$view.timer" {
		t.Errorf("timer(menuType=M 无子)Component = %q, want layout.base$view.timer(应按单级)", timer.Component)
	}

	// log:local-icon-log -> meta.localIcon=menu-log
	log := findRoute(routes, "log")
	if log == nil {
		t.Fatal("缺少 log 路由")
	}
	if log.Meta.LocalIcon != "menu-log" || log.Meta.Icon != "" {
		t.Errorf("log 图标分流错误: icon=%q localIcon=%q, want localIcon=menu-log", log.Meta.Icon, log.Meta.LocalIcon)
	}
}

// TestMenusToRoutesMetaFlags 断言 visible/isCache/isFrame -> hideInMenu/keepAlive/href 映射。
func TestMenusToRoutesMetaFlags(t *testing.T) {
	menus := []system.SysMenu{
		{MenuId: 1, ParentId: 0, MenuType: "C", MenuName: "route.admin", Path: "admin", Visible: "1", IsCache: "0", IsFrame: "0", OrderNum: 1},
	}
	routes := (&RouteService{}).menusToRoutes(menus)
	a := routes[0]
	if !a.Meta.HideInMenu {
		t.Error("Visible=1 应映射 hideInMenu=true")
	}
	if !a.Meta.KeepAlive {
		t.Error("IsCache=0 应映射 keepAlive=true")
	}
	if a.Meta.Href != "/admin" {
		t.Errorf("IsFrame=0 应映射 href=/admin, got %q", a.Meta.Href)
	}
}

func TestResolveHome(t *testing.T) {
	s := RouteService{}
	// defaultRouter 优先(且在路由里)
	if h := s.resolveHome([]system.MenuRoute{{Name: "home"}, {Name: "admin"}}, "admin"); h != "admin" {
		t.Errorf("resolveHome(defaultRouter=admin) = %q, want admin", h)
	}
	// defaultRouter 为空 + 含 home -> home(落地页兜底)
	if h := s.resolveHome([]system.MenuRoute{{Name: "system"}, {Name: "home"}}, ""); h != "home" {
		t.Errorf("resolveHome(含 home,无 defaultRouter) = %q, want home", h)
	}
	// defaultRouter 不在路由里 + 含 home -> 兜底 home
	if h := s.resolveHome([]system.MenuRoute{{Name: "home"}, {Name: "admin"}}, "disk"); h != "home" {
		t.Errorf("resolveHome(defaultRouter=disk 无权) = %q, want home", h)
	}
	// 不含 home,无 defaultRouter -> 取第一个顶层路由
	if h := s.resolveHome([]system.MenuRoute{{Name: "system"}, {Name: "log"}}, ""); h != "system" {
		t.Errorf("resolveHome(无 home) = %q, want system", h)
	}
	// 空 -> 默认 home(routeHomeDefault)
	if h := s.resolveHome(nil, ""); h != "home" {
		t.Errorf("resolveHome(空) = %q, want home", h)
	}
}

func TestContainsRouteName(t *testing.T) {
	tree := []system.MenuRoute{
		{Name: "system", Children: []system.MenuRoute{{Name: "system_user"}}},
		{Name: "admin"},
	}
	if !containsRouteName(tree, "system_user") {
		t.Error("应能找到子级 system_user")
	}
	if containsRouteName(tree, "nope") {
		t.Error("不应找到不存在的 name")
	}
}

func TestResolveLayout(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"layout.blank$view._admin_system_user", "blank"},
		{"layout.disk$view._admin_disk", "disk"},
		{"layout.disk", "disk"},
		{"layout.base$view.admin", "base"},
		{"Layout", "base"},                   // RuoYi 目录占位
		{"_admin/system/user/index", "base"}, // RuoYi 普通菜单路径
		{"FrameView", "base"},                // RuoYi 外链/iframe 占位
		{"", "base"},
	}
	for _, c := range cases {
		if got := resolveLayout(c.in); got != c.want {
			t.Errorf("resolveLayout(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestMenusToRoutesLayout 断言 SysMenu.Component 里编码的布局意图(blank/disk)被 menusToRoutes 提取,
// 且 component 里脏的 view 段(录入的 views 目录路径下划线化)被 routeKey(Path)规范化覆盖:
//   - 顶层单级 -> layout.<layout>$view.<key>
//   - 多级目录根 -> layout.<layout>
//   - 子级叶子 -> view.<key>(无外壳,布局不生效)
func TestMenusToRoutesLayout(t *testing.T) {
	menus := []system.SysMenu{
		// 顶层单级·网盘布局:component 的 view 段 _admin_disk 是脏的,应被 routeKey(disk) 覆盖
		{MenuId: 1, ParentId: 0, MenuType: "C", MenuName: "route.disk", Path: "disk", Component: "layout.disk$view._admin_disk", OrderNum: 1, Module: "disk"},
		// 顶层单级·空白布局:脏 view 段 print_page 应被 routeKey(print) 覆盖
		{MenuId: 2, ParentId: 0, MenuType: "C", MenuName: "route.print", Path: "print", Component: "layout.blank$view.print_page", OrderNum: 2, Module: "admin"},
		// 顶层多级目录·网盘布局(有子)
		{MenuId: 3, ParentId: 0, MenuType: "M", MenuName: "route.storage", Path: "storage", Component: "layout.disk", OrderNum: 3, Module: "disk"},
		{MenuId: 4, ParentId: 3, MenuType: "C", MenuName: "route.storage_file", Path: "storage/file", Component: "_disk/storage/file/index", OrderNum: 1, Module: "disk"},
	}
	routes := (&RouteService{}).menusToRoutes(menus)

	if r := findRoute(routes, "disk"); r == nil {
		t.Fatal("缺少 disk 路由")
	} else if r.Component != "layout.disk$view.disk" {
		t.Errorf("disk(网盘布局)Component = %q, want layout.disk$view.disk", r.Component)
	}

	if r := findRoute(routes, "print"); r == nil {
		t.Fatal("缺少 print 路由")
	} else if r.Component != "layout.blank$view.print" {
		t.Errorf("print(空白布局)Component = %q, want layout.blank$view.print", r.Component)
	}

	if r := findRoute(routes, "storage"); r == nil {
		t.Fatal("缺少 storage 路由")
	} else if r.Component != "layout.disk" {
		t.Errorf("storage(网盘目录)Component = %q, want layout.disk", r.Component)
	}

	if r := findRoute(routes, "storage_file"); r == nil {
		t.Fatal("缺少 storage_file 路由")
	} else if r.Component != "view.storage_file" {
		t.Errorf("storage_file(子级叶子)Component = %q, want view.storage_file(布局不生效,view 由 routeKey 覆盖)", r.Component)
	}
}
