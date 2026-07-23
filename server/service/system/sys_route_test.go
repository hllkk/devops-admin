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
		in              string
		wantIcon        string
		wantLocalIcon   string
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
		{MenuId: 1, ParentId: 0, MenuType: "C", MenuName: "route.admin", Path: "admin", Icon: "mdi:monitor-dashboard", Visible: "0", IsCache: "1", IsFrame: "1", OrderNum: 1},
		{MenuId: 2, ParentId: 0, MenuType: "M", MenuName: "route.system", Path: "system", Icon: "carbon:cloud-service-management", Visible: "0", IsCache: "1", IsFrame: "1", OrderNum: 2},
		{MenuId: 3, ParentId: 2, MenuType: "C", MenuName: "route.system_user", Path: "system/user", Icon: "carbon:user", Visible: "0", IsCache: "1", IsFrame: "1", OrderNum: 1},
		{MenuId: 4, ParentId: 0, MenuType: "M", MenuName: "route.timer", Path: "timer", Icon: "fluent:clock-24-regular", Visible: "0", IsCache: "1", IsFrame: "1", OrderNum: 3},
		{MenuId: 5, ParentId: 0, MenuType: "M", MenuName: "route.log", Path: "log", Icon: "local-icon-log", Visible: "0", IsCache: "1", IsFrame: "1", OrderNum: 4},
		{MenuId: 6, ParentId: 3, MenuType: "F", MenuName: "用户查询", Perms: "system:user:query", Icon: "#"}, // F 按钮应被过滤
	}

	s := RouteService{}
	routes := s.menusToRoutes(menus)

	// 顶层 4 个(admin/system/timer/log),F 按钮不进路由
	if want := 4; len(routes) != want {
		t.Fatalf("顶层路由数 = %d, want %d (F 按钮应被过滤)", len(routes), want)
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
	// 含 admin -> home=admin
	if h := s.resolveHome([]system.MenuRoute{{Name: "system"}, {Name: "admin"}}); h != "admin" {
		t.Errorf("resolveHome(含 admin) = %q, want admin", h)
	}
	// 不含 admin -> 取第一个
	if h := s.resolveHome([]system.MenuRoute{{Name: "system"}, {Name: "log"}}); h != "system" {
		t.Errorf("resolveHome(无 admin) = %q, want system", h)
	}
	// 空 -> 默认 admin
	if h := s.resolveHome(nil); h != "admin" {
		t.Errorf("resolveHome(空) = %q, want admin", h)
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
