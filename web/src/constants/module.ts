/**
 * 业务模块定义 —— 多模块隔离的基础。
 *
 * 四个业务模块：admin（后台管理）/ disk（网盘）/ server（服务器管理）/ gateway（AI 网关）。
 * 模块用于：① 菜单/路由隔离（每模块只展示各自路由）；② 每模块布局个性化（layout preset + 结构覆盖）。
 */

/** 业务模块标识 */
export type RouteModule = 'admin' | 'disk' | 'server' | 'gateway';

/** 默认模块（无 module 信息时的回退，例如首次进入根路径） */
export const DEFAULT_MODULE: RouteModule = 'admin';

/** 所有模块（用于校验/遍历） */
export const ALL_MODULES: RouteModule[] = ['admin', 'disk', 'server', 'gateway'];

/**
 * 布局预设：控制布局外壳结构。
 * - `standard`：标签页后台（有 tab / footer / 全局 header）
 * - `workbench`：沉浸式工作台（无 tab / footer，紧凑 header，适合网盘等）
 *
 * 与模块隔离（菜单归属）正交：同一 preset 可被多模块使用，同一模块切换 preset 不影响菜单。
 */
export type LayoutPreset = 'standard' | 'workbench';

/** 模块布局配置 ——「模块 → 布局预设 / 首页 / 图标」唯一数据源。新增模块在此加一行。 */
export interface ModuleConfig {
  /** 布局预设，决定布局外壳结构 */
  preset: LayoutPreset;
  /** 模块首页路由名（用于模块切换、logo 回首页） */
  home: string;
  /** 模块图标（Iconify），用于切换器/导航 */
  icon: string;
}

export const MODULE_CONFIG: Record<RouteModule, ModuleConfig> = {
  admin: { preset: 'standard', home: 'admin', icon: 'mdi:monitor-dashboard' },
  disk: { preset: 'workbench', home: 'disk', icon: 'mdi:harddisk' },
  server: { preset: 'standard', home: 'server', icon: 'mdi:server-network' },
  gateway: { preset: 'standard', home: 'gateway', icon: 'mdi:robot-outline' }
};

/**
 * 模块路由归属表 —— 过渡期的模块归属来源。
 *
 * 背景：Elegant Router 会剥离 `_xxx` 视图分组前缀，运行时无法从路由名/路径
 * 推断模块归属；且本地 `@elegant-router/vue/vite` 的 `onRouteMetaGen` 拿不到源文件路径。
 * 故用此表显式声明「路由名 → 所属模块」，由 `tagRoutesByModule` 在运行时写入 `meta.module`。
 *
 * 这是未来后端菜单表（含 `module` 字段）的**前端声明式雏形**：后端建菜单体系后，
 * 本表内容迁移为后端菜单的 module 字段，届时删除本表与 `tagRoutesByModule`，
 * 隔离 / 主题代码零改动（它们只读 `meta.module`）。
 *
 * 规则：
 * - 全局路由（404 / login / user-center / notice-user 等）**不**登记 → `meta.module` 为空 → 所有模块可见。
 * - 新增某模块的路由时，在对应数组里追加其路由名。
 */
export const MODULE_ROUTES: Record<RouteModule, string[]> = {
  admin: ['admin', 'system', 'system_user', 'log', 'timer'],
  disk: ['disk'],
  server: ['server'],
  gateway: ['gateway']
};

/**
 * 反查表：路由名 → 模块（运行时打标用，由 MODULE_ROUTES 派生，勿手改）。
 */
export const ROUTE_TO_MODULE: Record<string, RouteModule> = Object.entries(MODULE_ROUTES).reduce(
  (acc, [module, names]) => {
    names.forEach(name => {
      acc[name] = module as RouteModule;
    });
    return acc;
  },
  {} as Record<string, RouteModule>
);
