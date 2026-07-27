import type { CustomRoute, ElegantConstRoute, ElegantRoute } from '@elegant-router/types';
import { generatedRoutes } from '../elegant/routes';
import { layouts, views } from '../elegant/imports';
import { transformElegantRoutesToVueRoutes } from '../elegant/transform';

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
 * Tag disk layout:给 component 含 layout.disk 的路由节点打 meta.useDiskLayout=true。
 * 布局完全由后端下发的 component 决定(不再按 module 改写);这里只做一次标记,
 * 供 themeStore.effectiveLayoutMode 判断是否定死 vertical-mix(disk 布局的固有菜单模式)。
 * 递归处理 children;vue-router 的 route.meta 会合并所有 matched 层级,故子路由自动继承父的标记。
 */
function tagLayoutMeta(routes: ElegantConstRoute[]): ElegantConstRoute[] {
  return routes.map(route => {
    const next: ElegantConstRoute = { ...route };
    if (typeof next.component === 'string' && next.component.includes('layout.disk')) {
      // 拷新 meta 打标记(避免污染原 route 的 meta 引用);断言绕过展开后 title 被窄化为 optional 与 RouteMeta.title 必填的差异
      next.meta = { ...next.meta, useDiskLayout: true } as typeof next.meta;
    }
    if (next.children) {
      next.children = tagLayoutMeta(next.children as ElegantConstRoute[]);
    }
    return next;
  });
}

/**
 * Auto-layout 候选:登录后才访问的全局公共功能页(无 meta.module),外壳跟随 currentModule。
 * 在此加一行即可让未来的消息中心等公共页同效。
 */
const AUTO_LAYOUT_ROUTES = new Set<string>(['user-center']);

/**
 * 把名单内全局页的 layout 从 base 改写成 auto(运行期由 AutoLayout 按 currentModule 选 base/disk)。
 * 改写点在三条初始化路径的公共出口 getAuthVueRoutes,一次覆盖 static/dynamic/constant,不碰生成产物。
 */
function rewriteAutoLayout(routes: ElegantConstRoute[]): ElegantConstRoute[] {
  return routes.map(route => {
    if (AUTO_LAYOUT_ROUTES.has(route.name) && typeof route.component === 'string') {
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
  return transformElegantRoutesToVueRoutes(tagLayoutMeta(rewriteAutoLayout(routes)), layouts, views);
}
