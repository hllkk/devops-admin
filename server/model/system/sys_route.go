package system

// MenuRouteMeta 前端 Elegant Router 的路由 meta(ElegantConstRoute.meta)子集。
// 字段名/JSON key 对齐 web/src/router/elegant/routes.ts 静态生成的 meta 结构,
// 由 SysMenu 转换填充。未用到的 soybean meta 字段(roles/activeMenu/fullScreen...)暂不定义,按需扩展。
type MenuRouteMeta struct {
	Title      string `json:"title"`                // 兜底标题(显示优先 i18nKey);取 routeKey
	I18nKey    string `json:"i18nKey,omitempty"`    // i18n key;取 SysMenu.MenuName(如 route.system_user)
	Icon       string `json:"icon,omitempty"`       // Iconify 图标名(如 mdi:xxx);来自 SysMenu.Icon
	LocalIcon  string `json:"localIcon,omitempty"`  // 本地 svg 图标名(如 menu-log);来自 SysMenu.Icon 的 local-icon- 前缀
	Order      int    `json:"order,omitempty"`      // 排序;取 SysMenu.OrderNum
	Module     string `json:"module,omitempty"`     // 业务模块归属(admin/disk/server/gateway);取 SysMenu.Module,前端 meta.module 驱动模块隔离
	HideInMenu bool   `json:"hideInMenu,omitempty"` // 菜单隐藏;SysMenu.Visible=="1"
	KeepAlive  bool   `json:"keepAlive,omitempty"`  // 页面缓存;SysMenu.IsCache=="0"
	Href       string `json:"href,omitempty"`       // 外链地址;SysMenu.IsFrame=="0" 时取规范化 Path
}

// MenuRoute 前端 Api.Route.MenuRoute(ElegantConstRoute & { id })。
// 由 SysMenu(M/C,过滤 F 按钮)按层级转换:目录=layout.base、顶层单级=layout.base$view.<key>、
// 子级叶子=view.<key>;name/path 由 SysMenu.Path 推导(对齐 Elegant Router 的 RouteKey)。
type MenuRoute struct {
	Id        string        `json:"id"`                  // 菜单 ID(雪花,string)
	Name      string        `json:"name"`                // 路由名 = RouteKey(由 Path 推导)
	Path      string        `json:"path"`                // 路由路径(绝对,"/"+Path)
	Component string        `json:"component,omitempty"` // 组件引用(layout.base / view.<key> / layout.base$view.<key>)
	Redirect  string        `json:"redirect,omitempty"`  // 重定向(目录节点前端自动算,后端通常不下发)
	Meta      MenuRouteMeta `json:"meta"`                // 路由 meta
	Children  []MenuRoute   `json:"children,omitempty"`  // 子路由(多级目录)
}

// UserRoute 前端 Api.Route.UserRoute { routes, home }。
// routes=当前用户有权访问的 MenuRoute 树;home=落地路由 key(对齐 VITE_ROUTE_HOME)。
type UserRoute struct {
	Routes []MenuRoute `json:"routes"`
	Home   string      `json:"home"`
}
