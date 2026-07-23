package system

import (
	"context"
	"strconv"
	"strings"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
)

// RouteService 动态路由下发(/route):把 SysMenu(RuoYi 契约)实时转换为前端
// Soybean Elegant Router 的 MenuRoute(ElegantConstRoute)树下发。存储层(SysMenu)不动,
// 转换在 getUserRoutes 调用时完成——菜单管理页继续用 RuoYi 风格编辑,动态路由用转换后的数据。
type RouteService struct{}

const (
	routeLocalIconPrefix = "local-icon-" // SysMenu.Icon 本地图标前缀,转 meta.localIcon("menu-<name>")
	routeIconNone        = "#"           // RuoYi 无图标占位,转换时忽略
	routeHomeDefault     = "admin"       // 落地页 key,对齐 web/.env VITE_ROUTE_HOME=admin
	routeLayoutBase      = "layout.base" // Soybean 基础布局组件引用
)

// routeKey 由 SysMenu.Path 推导 Elegant Router 的 RouteKey/视图 key:
// 去首尾斜杠,把 "/" 换成 "_"。与 web/src/router/elegant/transform.ts 的 RouteKey 生成规则一致
// (RouteKey 按 views 目录层级生成,如 _admin/system/user/index.vue -> system_user)。
//
//	"system/user" -> "system_user"   "admin" -> "admin"   "log/loginlog" -> "log_loginlog"
func routeKey(path string) string {
	return strings.ReplaceAll(strings.Trim(path, "/"), "/", "_")
}

// routePath 规范化为绝对路径:"system/user" -> "/system/user","admin" -> "/admin"。
func routePath(path string) string {
	s := strings.Trim(path, "/")
	if s == "" {
		return "/"
	}
	return "/" + s
}

// resolveIcon 分流 SysMenu.Icon:local-icon-<name> -> (localIcon="menu-<name>");
// 其余非空非 "#" -> (icon=原值)。
func resolveIcon(icon string) (iconOut, localIconOut string) {
	if icon == "" || icon == routeIconNone {
		return "", ""
	}
	if strings.HasPrefix(icon, routeLocalIconPrefix) {
		return "", "menu-" + strings.TrimPrefix(icon, routeLocalIconPrefix)
	}
	return icon, ""
}

// menusToRoutes 将菜单平表(已按 menuOrder 排序)组装并转换为 MenuRoute 树。
// F 按钮不进路由(仅 perms,走 userInfo.permissions);单/多级按 children 有无判定
// (对齐 transform.ts isSingleLevelRoute),不盲信 menuType——顶层 M 无子(如 timer)按单级处理。
func (s *RouteService) menusToRoutes(menus []system.SysMenu) []system.MenuRoute {
	// 仅 M/C 进路由,过滤 F 按钮
	routeMenus := make([]system.SysMenu, 0, len(menus))
	for _, m := range menus {
		if m.MenuType == "F" {
			continue
		}
		routeMenus = append(routeMenus, m)
	}

	childrenOf := make(map[int64][]system.SysMenu, len(routeMenus))
	for _, m := range routeMenus {
		childrenOf[m.ParentId] = append(childrenOf[m.ParentId], m)
	}

	var build func(parentId int64) []system.MenuRoute
	build = func(parentId int64) []system.MenuRoute {
		nodes := childrenOf[parentId]
		out := make([]system.MenuRoute, 0, len(nodes))
		for _, n := range nodes {
			key := routeKey(n.Path)
			icon, localIcon := resolveIcon(n.Icon)
			r := system.MenuRoute{
				Id:   strconv.FormatInt(n.MenuId, 10),
				Name: key,
				Path: routePath(n.Path),
				Meta: system.MenuRouteMeta{
					Title:     key,
					I18nKey:   n.MenuName,
					Icon:      icon,
					LocalIcon: localIcon,
					Order:     n.OrderNum,
				},
			}
			if n.Visible == "1" {
				r.Meta.HideInMenu = true
			}
			if n.IsCache == "0" {
				r.Meta.KeepAlive = true
			}
			if n.IsFrame == "0" {
				r.Meta.Href = routePath(n.Path)
			}
			// component 由层级 + 是否有子节点决定(对齐 transform.ts 的 layout./view. 解析)
			kids := build(n.MenuId)
			switch {
			case len(kids) > 0:
				r.Children = kids
				if n.ParentId == 0 {
					r.Component = routeLayoutBase // 多级目录:仅 layout
				} else {
					r.Component = "view." + key // 兜底:C 理论无子(子为 F 已过滤)
				}
			case n.ParentId == 0:
				r.Component = routeLayoutBase + "$view." + key // 顶层单级页:layout$view 拼接
			default:
				r.Component = "view." + key // 子级叶子页
			}
			out = append(out, r)
		}
		return out
	}
	return build(0)
}

// GetUserRoutes 按当前用户角色过滤 SysMenu 后转换为 MenuRoute 树下发。
// 超管(任一角色 SuperAdmin)返回全部菜单;普通用户按 角色->sys_role_menu->sys_menu 过滤,
// 并向上回溯补齐祖先目录(避免授权子菜单时父目录缺失导致树断裂/面包屑丢失)。
func (s *RouteService) GetUserRoutes(ctx context.Context, userId int64) (result system.UserRoute, err error) {
	// 1. 取用户角色(Preload Roles)
	var user system.SysUser
	if err = global.OPS_DB.WithContext(ctx).Preload("Roles").
		Where("id = ?", userId).First(&user).Error; err != nil {
		return
	}
	isSuper := false
	roleIds := make([]int64, 0, len(user.Roles))
	for _, r := range user.Roles {
		roleIds = append(roleIds, r.RoleId)
		if r.SuperAdmin {
			isSuper = true
		}
	}

	// 2. 取菜单(M/C,F 不进路由):超管全量,否则按 roleIds 过滤 + 向上回溯补祖先
	query := global.OPS_DB.WithContext(ctx).Model(&system.SysMenu{}).
		Where("menu_type IN ?", []string{"M", "C"}).
		Order(menuOrder)
	var menus []system.SysMenu
	if isSuper {
		err = query.Find(&menus).Error
		if err != nil {
			return
		}
	} else {
		// 授权菜单 id(角色经 sys_role_menu 关联)
		var grantedIds []int64
		if len(roleIds) > 0 {
			if qerr := global.OPS_DB.WithContext(ctx).Model(&system.SysRoleMenu{}).
				Where("sys_role_id IN ?", roleIds).
				Distinct("sys_menu_id").Pluck("sys_menu_id", &grantedIds).Error; qerr != nil {
				err = qerr
				return
			}
		}
		if len(grantedIds) == 0 {
			result = system.UserRoute{Routes: []system.MenuRoute{}, Home: routeHomeDefault}
			return
		}
		ids := s.collectWithAncestors(ctx, grantedIds)
		err = query.Where("menu_id IN ?", ids).Find(&menus).Error
		if err != nil {
			return
		}
	}

	result.Routes = s.menusToRoutes(menus)
	result.Home = s.resolveHome(result.Routes)
	return
}

// resolveHome 落地页 key:优先 admin(对齐 VITE_ROUTE_HOME),否则取第一个顶层路由 key。
func (s *RouteService) resolveHome(routes []system.MenuRoute) string {
	for _, r := range routes {
		if r.Name == routeHomeDefault {
			return routeHomeDefault
		}
	}
	if len(routes) > 0 {
		return routes[0].Name
	}
	return routeHomeDefault
}

// collectWithAncestors 收集 ids 及其全部祖先 menu_id(沿 parent_id 上溯,菜单无 ancestors 列只能逐层查)。
func (s *RouteService) collectWithAncestors(ctx context.Context, ids []int64) []int64 {
	all := make(map[int64]bool, len(ids))
	for _, id := range ids {
		all[id] = true
	}
	current := ids
	for len(current) > 0 {
		var parents []int64
		global.OPS_DB.WithContext(ctx).Model(&system.SysMenu{}).
			Where("menu_id IN ?", current).Pluck("parent_id", &parents)
		next := make([]int64, 0, len(parents))
		for _, p := range parents {
			if p > 0 && !all[p] {
				all[p] = true
				next = append(next, p)
			}
		}
		current = next
	}
	result := make([]int64, 0, len(all))
	for id := range all {
		result = append(result, id)
	}
	return result
}

// IsRouteExist 判断 routeName 是否存在于当前用户有权访问的路由名集合(对齐前端 fetchIsRouteExist,路由守卫用)。
func (s *RouteService) IsRouteExist(ctx context.Context, userId int64, routeName string) (bool, error) {
	ur, err := s.GetUserRoutes(ctx, userId)
	if err != nil {
		return false, err
	}
	return containsRouteName(ur.Routes, routeName), nil
}

// containsRouteName 深度优先查路由树是否存在指定 name。
func containsRouteName(routes []system.MenuRoute, name string) bool {
	for _, r := range routes {
		if r.Name == name {
			return true
		}
		if len(r.Children) > 0 && containsRouteName(r.Children, name) {
			return true
		}
	}
	return false
}
