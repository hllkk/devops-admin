import type { App } from 'vue';
import {
  type LocationQueryRaw,
  type RouterHistory,
  createMemoryHistory,
  createRouter,
  createWebHashHistory,
  createWebHistory
} from 'vue-router';
import { createBuiltinVueRoutes } from './routes/builtin';
import { createRouterGuard } from './guard';

const { VITE_ROUTER_HISTORY_MODE = 'history', VITE_BASE_URL } = import.meta.env;

const historyCreatorMap: Record<Env.RouterHistoryMode, (base?: string) => RouterHistory> = {
  hash: createWebHashHistory,
  history: createWebHistory,
  memory: createMemoryHistory
};

/**
 * 自定义 query 序列化:用 encodeURIComponent 编码键值(RFC3986 语义),
 * 使地址栏 query 统一为 %2F/%20 风格(对齐 jmal、百度网盘等主流网盘),
 * 而非 vue-router 默认把空格编成 `+`、不编码 `/` 的表单风格。
 * parseQuery 沿用 vue-router 默认实现,已能正确还原 %2F→/、%20→空格、+→空格。
 */
function stringifyQuery(query: LocationQueryRaw | undefined): string {
  if (!query) return '';
  const pairs: string[] = [];
  for (const key of Object.keys(query)) {
    const encodedKey = encodeURIComponent(key);
    const value = query[key];
    if (value === null || value === undefined) {
      // null/undefined 值只输出 key,对齐 vue-router 默认
      pairs.push(encodedKey);
    } else if (Array.isArray(value)) {
      for (const item of value) {
        if (item === null || item === undefined) {
          pairs.push(encodedKey);
        } else {
          pairs.push(`${encodedKey}=${encodeURIComponent(String(item))}`);
        }
      }
    } else {
      pairs.push(`${encodedKey}=${encodeURIComponent(String(value))}`);
    }
  }
  if (pairs.length === 0) return '';
  return `${pairs.join('&')}`;
}

export const router = createRouter({
  history: historyCreatorMap[VITE_ROUTER_HISTORY_MODE](VITE_BASE_URL),
  routes: createBuiltinVueRoutes(),
  stringifyQuery
});

/** Setup Vue Router */
export async function setupRouter(app: App) {
  app.use(router);
  createRouterGuard(router);
  await router.isReady();
}
