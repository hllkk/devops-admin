# 个人中心路由不可达修复（动态路由补齐全局公共页）

> 2026-09-03。BUG：点击头像下拉「个人中心」无反应（router.push No match）。参照 SoyDisk 的 `AUTO_LAYOUT_ROUTES` 合并机制修复，纯前端 1 文件。

## 根因

- `views/_builtin/user-center` 的路由由 elegant-router 生成为 **auth route**（无 `meta.constant`、`hideInMenu`、无 `meta.module`），不走 constant，也不进菜单
- dynamic 模式（`VITE_AUTH_ROUTE_MODE` 非静态）下 auth 路由只从后端 `getUserRoutes`（sys_menu）取，**后端菜单从不下发这类全局公共页** → 前端路由表里没有 `user-center` → 头像下拉 `routerPushByKey('user-center')` No match
- 项目此前已铺好该机制的另一半基建：`constants/module.ts` 已定义 `AUTO_LAYOUT_ROUTES = ['user-center']`，`router/routes/index.ts` 已有 `rewriteAutoLayout`（layout.base→layout.auto 改写）；唯独 route store 缺「合并」环节，名单成了摆设

## 修复（`web/src/store/modules/route/index.ts`，与 SoyDisk 逐行同构）

1. import 补 `AUTO_LAYOUT_ROUTES`
2. 新增 `mergeGlobalPublicAuthRoutes()`：从 `createStaticRoutes().authRoutes`（elegant 生成产物）按名单过滤，**仅补后端未下发的**（`existing` 集合去重，后端若下发则以后端为准不覆盖）
3. `initDynamicAuthRoute` 内 `addAuthRoutes(routes)` 之后调用（在 `handleConstantAndAuthRoutes` 前）

后续新增此类「登录后全局公共页」（如未来的消息中心）时：页面放 `_builtin/`，`constants/module.ts` 的 `AUTO_LAYOUT_ROUTES` 加路由名即可，无需后端菜单。

## 关联

[[user-center-profile]]（个人中心资料/头像接口，另一功能点）；SoyDisk 参照实现位于 `/home/SoyDisk/web/src/store/modules/route/index.ts` 同名函数。

## 验证

`pnpm typecheck` 通过；`createStaticRoutes()` 数据源（generatedRoutes 含 user-center、无 meta.constant 进 authRoutes）已核对。
