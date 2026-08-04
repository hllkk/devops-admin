/**
 * 业务模块定义 —— 多模块隔离的基础。
 *
 * 三个业务模块：admin（后台管理）/ server（服务器管理）/ gateway（AI 网关）。
 * 模块用于：① 菜单/路由隔离（每模块只展示各自路由）；② 每模块菜单结构覆盖（theme moduleOverrides）。
 * 布局外壳(layout.base/blank)由菜单 component 决定,不再绑定 module——见 router/routes tagLayoutMeta;
 * 例外:全局公共页(个人中心等)走 layout.auto,跟随 currentModule,见下方 AUTO_LAYOUT_ROUTES。
 */

/** 业务模块标识 */
export type RouteModule = 'admin' | 'server' | 'gateway';

/** 默认模块（无 module 信息时的回退，例如首次进入根路径） */
export const DEFAULT_MODULE: RouteModule = 'admin';

/** 所有模块（用于校验/遍历） */
export const ALL_MODULES: RouteModule[] = ['admin', 'server', 'gateway'];

/** 模块配置 ——「模块 → 首页 / 图标」唯一数据源。新增模块在此加一行。 */
export interface ModuleConfig {
  /** 模块首页路由名（用于模块切换、logo 回首页） */
  home: string;
  /** 模块图标（Iconify），用于切换器/导航 */
  icon: string;
}

export const MODULE_CONFIG: Record<RouteModule, ModuleConfig> = {
  admin: { home: 'admin', icon: 'mdi:monitor-dashboard' },
  server: { home: 'server', icon: 'mdi:server-network' },
  gateway: { home: 'gateway', icon: 'mdi:robot-outline' }
};

/**
 * 走 auto 布局(跟随 currentModule)的全局公共页路由名。
 *
 * 这些是「登录后才访问、无 meta.module」的公共功能页(个人中心 / 未来的消息中心),
 * 外壳不写死 base,而是由 AutoLayout 按 currentModule 在 base/disk 间切换。
 *
 * elegant-router 0.3.8 只有全局 defaultLayout、无 per-route layout 钩子,故 layout.auto 由
 * router/routes/index.ts 的 rewriteAutoLayout 在后处理层把 component 从 layout.base 改写而来。
 * 新增此类页面时,在此加一行路由名即可。
 */
export const AUTO_LAYOUT_ROUTES: readonly string[] = ['user-center'];
