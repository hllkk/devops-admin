import type { CustomRoute, ElegantConstRoute, ElegantRoute } from '@elegant-router/types';
import { generatedRoutes } from '../elegant/routes';
import { layouts, views } from '../elegant/imports';
import { transformElegantRoutesToVueRoutes } from '../elegant/transform';
import { AUTO_LAYOUT_ROUTES } from '@/constants/module';

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
 * 把名单内全局页(AUTO_LAYOUT_ROUTES,见 constants/module)的 layout 从 base 改写成 auto
 * (运行期由 AutoLayout 按 currentModule 选 base/disk)。
 * 改写点在三条初始化路径的公共出口 getAuthVueRoutes,一次覆盖 static/dynamic/constant,不碰生成产物。
 */
function rewriteAutoLayout(routes: ElegantConstRoute[]): ElegantConstRoute[] {
  return routes.map(route => {
    if (AUTO_LAYOUT_ROUTES.includes(route.name) && typeof route.component === 'string') {
      return { ...route, component: route.component.replace('layout.base', 'layout.auto') };
    }
    return route;
  });
}

/**
 * Get auth vue routes
 *
 * @param routes Elegant routes
 */
export function getAuthVueRoutes(routes: ElegantConstRoute[]) {
  return transformElegantRoutesToVueRoutes(rewriteAutoLayout(routes), layouts, views);
}
