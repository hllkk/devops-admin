# 模块隔离 module 字段后端化 + 动态路由

> 日期：2026-07-24 ｜ 状态：已实现

## 需求

前端三模块(admin/server/gateway)隔离原先靠前端声明表 `MODULE_ROUTES`(route name→module,见 `constants/module.ts`)兜底。改为后端菜单 `SysMenu.module` 字段驱动,经动态路由 `/route/getUserRoutes` 下发 `meta.module`,前端 `filterRoutesByModule` 据此隔离。归属来源从前端静态声明迁移到后端数据。

## 改动

- `model/system/sys_menu.go`:`SysMenu` 加 `Module` 列(gorm `column:module`,json `module,omitempty`)
- `source/system/sys_menu.go`:seed 每条 M/C 菜单填 module(admin/system/log/timer 及子菜单→`admin`;server/gateway 各自);F 按钮不填(不进路由)
- `model/system/sys_route.go`:`MenuRouteMeta` 加 `Module`;`service/system/sys_route.go` 的 `menusToRoutes` 下发 `Module: n.Module`(测试 `sys_route_test.go` 加 module 断言)
- 前端删 `MODULE_ROUTES`/`ROUTE_TO_MODULE`/`tagRoutesByModule`(`constants/module.ts`、`store/modules/route/module.ts`、`store/modules/route/index.ts`);`typings/router.d.ts` 注释更新。隔离只靠 `filterRoutesByModule` 读 `meta.module`
- `web/.env`:`VITE_AUTH_ROUTE_MODE=dynamic`(分两步:先补 server/gateway seed 切 dynamic 验证,再 module 后端化)

## 关键约束

- **绑定 dynamic**:删前端表后,`tagRoutesByModule` 已删,static 模式下 `meta.module` 无来源(`routes.ts` 静态路由不含 module)→ 隔离失效。`.env` 不可切回 static
- **老库 module 回填**:`AutoMigrate` 加列后现有行 module 为空,需按 path 前缀回填(CASE SQL:`server%/gateway%` 各自,ELSE admin),否则空 module 被当全局路由→隔离静默失效。全新初始化的库 seed 已带 module,无需回填

## 现状

- server/gateway 为占位模块(各一条占位 index.vue + seed 菜单),后端业务待建
- 后端 dynamic 路由下发链路已完整(`/route/getUserRoutes`/`getConstantRoutes`/`isRouteExist` + `menusToRoutes` 转换 + 按角色过滤 + 向上回溯补祖先)
- `getUserInfo` 已下发 `roles + permissions`(按钮权限配套齐备)

## 相关

- [[menu-seed-routes-alignment]] 菜单 seed 对齐(其"apis→casbin"机制仍未落地;本需求的 module 字段已落地)
- [[menu-management]] 菜单模型(SysMenu)
