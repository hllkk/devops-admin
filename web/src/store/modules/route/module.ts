import type { ElegantConstRoute } from '@elegant-router/types';
import { ROUTE_TO_MODULE, type RouteModule } from '@/constants/module';

/**
 * Resolve the module a route belongs to.
 *
 * Sole source of truth: `route.meta.module` — set by `tagRoutesByModule` from
 * `MODULE_ROUTES` (static mode), or provided by backend menu payload (dynamic mode).
 * **No path-prefix fallback** — avoids guessing ownership from the URL.
 *
 * @returns the `RouteModule`, or `null` for global routes (404 / login / user-center ...).
 */
export function resolveModuleFromRoute(route: { meta?: { module?: RouteModule | null } | null }): RouteModule | null {
  return route.meta?.module ?? null;
}

/**
 * Tag routes with `meta.module` from `MODULE_ROUTES` (transition source of module ownership).
 *
 * - Tags **only** routes that have no `module` yet → in dynamic mode, backend-provided
 *   `meta.module` is preserved, so this step is a no-op and migration is smooth.
 * - Routes absent from `MODULE_ROUTES` keep `module = undefined` → treated as global.
 *
 * Call after `createStaticRoutes()` and before `handleConstantAndAuthRoutes()`.
 */
export function tagRoutesByModule(routes: ElegantConstRoute[]): ElegantConstRoute[] {
  return routes.map(tagRouteByModule);
}

function tagRouteByModule(route: ElegantConstRoute): ElegantConstRoute {
  const next: ElegantConstRoute = { ...route };
  const existingMeta = next.meta;

  // Only tag routes that already have meta but no module (preserve backend-provided module in dynamic mode)
  if (existingMeta && !existingMeta.module) {
    const moduleName = route.name ? ROUTE_TO_MODULE[route.name as string] : undefined;
    if (moduleName) {
      next.meta = { ...existingMeta, module: moduleName };
    }
  }

  if (next.children?.length) {
    next.children = next.children.map(tagRouteByModule);
  }

  return next;
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
