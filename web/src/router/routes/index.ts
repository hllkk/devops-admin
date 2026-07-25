import type { CustomRoute, ElegantConstRoute, ElegantRoute } from '@elegant-router/types';
import { generatedRoutes } from '../elegant/routes';
import { layouts, views } from '../elegant/imports';
import { transformElegantRoutesToVueRoutes } from '../elegant/transform';
import { MODULE_CONFIG, type RouteModule } from '@/constants/module';

/**
 * custom routes
 *
 * @link https://github.com/soybeanjs/elegant-router?tab=readme-ov-file#custom-route
 */
const customRoutes: CustomRoute[] = [];

/** create routes when the auth route mode is static */
export function createStaticRoutes() {
  const constantRoutes: ElegantRoute[] = [];

  const authRoutes: ElegantRoute[] = [];

  [...customRoutes, ...generatedRoutes].forEach(item => {
    if (item.meta?.constant) {
      constantRoutes.push(item);
    } else {
      authRoutes.push(item);
    }
  });

  return {
    constantRoutes,
    authRoutes
  };
}

/**
 * Apply module layout:把 preset='disk' 模块的路由布局从 layout.base 换成 layout.disk。
 * 递归处理 children;module 为空(全局路由)不动。MODULE_CONFIG.preset 的唯一消费点。
 */
function applyModuleLayout(routes: ElegantConstRoute[]): ElegantConstRoute[] {
  return routes.map(route => {
    const module = (route.meta as { module?: RouteModule } | undefined)?.module;
    const next: ElegantConstRoute = { ...route };
    if (module && MODULE_CONFIG[module]?.preset === 'disk' && next.component) {
      next.component = next.component.replace('layout.base', 'layout.disk');
    }
    if (next.children) {
      next.children = applyModuleLayout(next.children as ElegantConstRoute[]);
    }
    return next;
  });
}

/**
 * Get auth vue routes
 *
 * @param routes Elegant routes
 */
export function getAuthVueRoutes(routes: ElegantConstRoute[]) {
  return transformElegantRoutesToVueRoutes(applyModuleLayout(routes), layouts, views);
}
