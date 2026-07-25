/**
 * 业务模块定义 —— 多模块隔离的基础。
 *
 * 四个业务模块：admin（后台管理）/ disk（网盘）/ server（服务器管理）/ gateway（AI 网关）。
 * 模块用于：① 菜单/路由隔离（每模块只展示各自路由）；② 每模块布局个性化（layout preset + 结构覆盖）。
 */

/** 业务模块标识 */
export type RouteModule = 'admin' | 'disk' | 'server' | 'gateway';

/** 默认模块（无 module 信息时的回退，例如首次进入根路径） */
export const DEFAULT_MODULE: RouteModule = 'disk';

/** 所有模块（用于校验/遍历） */
export const ALL_MODULES: RouteModule[] = ['admin', 'disk', 'server', 'gateway'];

/**
 * 布局预设：值对齐布局注册表的 layout name（router/elegant/imports.ts 的 layouts 键）。
 * router/routes 的 applyModuleLayout 按模块 preset 把路由 component 从 layout.base 替换为对应布局。
 * - `base`：标准后台（layout.base；菜单模式可配、有 tab、有主题设置入口）
 * - `disk`：网盘布局（layout.disk；定死左侧菜单混合模式 vertical-mix、无 tab、无主题设置入口）
 *
 * 与模块隔离（菜单归属）正交：preset 决定布局外壳，不影响菜单按 module 过滤。
 */
export type LayoutPreset = 'base' | 'disk';

/** 模块布局配置 ——「模块 → 布局预设 / 首页 / 图标」唯一数据源。新增模块在此加一行。 */
export interface ModuleConfig {
  /** 布局预设(= layout name):'base' 标准后台 | 'disk' 网盘(vertical-mix 定死 / 无 tab / 无主题入口) */
  preset: LayoutPreset;
  /** 模块首页路由名（用于模块切换、logo 回首页） */
  home: string;
  /** 模块图标（Iconify），用于切换器/导航 */
  icon: string;
}

export const MODULE_CONFIG: Record<RouteModule, ModuleConfig> = {
  admin: { preset: 'base', home: 'admin', icon: 'mdi:monitor-dashboard' },
  disk: { preset: 'disk', home: 'disk', icon: 'mdi:harddisk' },
  server: { preset: 'base', home: 'server', icon: 'mdi:server-network' },
  gateway: { preset: 'base', home: 'gateway', icon: 'mdi:robot-outline' }
};
