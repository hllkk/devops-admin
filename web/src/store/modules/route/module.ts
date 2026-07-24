import type { ElegantConstRoute } from '@elegant-router/types';
import type { RouteModule } from '@/constants/module';

/**
 * Resolve the module a route belongs to.
 *
 * Sole source of truth: `route.meta.module` — provided by the backend menu payload
 * (dynamic mode, /route/getUserRoutes 下发 SysMenu.module). **No path-prefix fallback** —
 * avoids guessing ownership from the URL.
 *
 * @returns the `RouteModule`, or `null` for global routes (404 / login / user-center ...).
 */
export function resolveModuleFromRoute(route: { meta?: { module?: RouteModule | null } | null }): RouteModule | null {
  return route.meta?.module ?? null;
}

/**
 * Filter routes by module (module isolation — each module shows only its own routes).
 *
 * - `meta.module === current module` ⇒ keep
 * - no `meta.module` ⇒ global route (404 / login / user-center ...), visible in all modules
 * - a parent left with no children after filtering is excluded
 *
 * Isomorphic to `filterAuthRouteByRoles` (route/shared.ts:23). This is the **single**
 * place module isolation happens; called once from `handleConstantAndAuthRoutes`.
 */
export function filterRoutesByModule(routes: ElegantConstRoute[], module: RouteModule): ElegantConstRoute[] {
  return routes.flatMap(route => filterRouteByModule(route, module));
}

function filterRouteByModule(route: ElegantConstRoute, module: RouteModule): ElegantConstRoute[] {
  const routeModule = route.meta?.module;
  const isGlobal = !routeModule;
  const isCurrent = routeModule === module;

  const filterRoute = { ...route };

  if (filterRoute.children?.length) {
    filterRoute.children = filterRoute.children.flatMap(item => filterRouteByModule(item, module));
  }

  if (filterRoute.children?.length === 0) {
    return [];
  }

  return isGlobal || isCurrent ? [filterRoute] : [];
}
