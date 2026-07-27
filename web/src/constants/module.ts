/**
 * 业务模块定义 —— 多模块隔离的基础。
 *
 * 四个业务模块：admin（后台管理）/ disk（网盘）/ server（服务器管理）/ gateway（AI 网关）。
 * 模块用于：① 菜单/路由隔离（每模块只展示各自路由）；② 每模块菜单结构覆盖（theme moduleOverrides）。
 * 布局外壳(layout.base/disk/blank)由菜单 component 决定,不再绑定 module——见 router/routes tagLayoutMeta。
 */

/** 业务模块标识 */
export type RouteModule = 'admin' | 'disk' | 'server' | 'gateway';

/** 默认模块（无 module 信息时的回退，例如首次进入根路径） */
export const DEFAULT_MODULE: RouteModule = 'disk';

/** 所有模块（用于校验/遍历） */
export const ALL_MODULES: RouteModule[] = ['admin', 'disk', 'server', 'gateway'];

/** 模块配置 ——「模块 → 首页 / 图标」唯一数据源。新增模块在此加一行。 */
export interface ModuleConfig {
  /** 模块首页路由名（用于模块切换、logo 回首页） */
  home: string;
  /** 模块图标（Iconify），用于切换器/导航 */
  icon: string;
}

export const MODULE_CONFIG: Record<RouteModule, ModuleConfig> = {
  admin: { home: 'admin', icon: 'mdi:monitor-dashboard' },
  disk: { home: 'disk', icon: 'mdi:harddisk' },
  server: { home: 'server', icon: 'mdi:server-network' },
  gateway: { home: 'gateway', icon: 'mdi:robot-outline' }
};
