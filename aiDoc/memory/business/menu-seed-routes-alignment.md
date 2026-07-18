# 菜单初始化 seed 对齐前端 routes.ts

> 日期：2026-07-15 ｜ 状态：已实现（后端 seed 落地，dict/notice/operlog 接口待补）

## 需求

基于前端 `web/src/router/elegant/routes.ts`，完善后端系统初始化时的菜单初始化（`server/source/system/sys_menu.go` 的 `InitializeData` 的 `entities`）。

## 关键约束（决定怎么"完善"）

- 前端是 **static 路由模式**（`web/.env` `VITE_AUTH_ROUTE_MODE=static`），菜单/路由由前端 `routes.ts` 渲染。后端 menu seed **不驱动前端渲染**。
- 后端 menu seed 的真实作用：① 角色-菜单分配（`sys_role_menu`，admin 角色挂全部菜单）；② **casbin 接口权限推导**（C 菜单的 `apis` → 角色的 API 策略）。
- casbin matcher = `keyMatch2(r.obj, p.obj)`，支持 `:param` 占位（`/system/user/:id` 命中 `/system/user/123`）；`obj = 请求路径 strings.TrimPrefix(..., System.RouterPrefix)`；`apis[].Path` 须写"去 RouterPrefix 后"的形式，与 `middleware/casbin_rbac.go` 一致。`service/system/sys_casbin.go` `UpdateCasbin` 遍历角色关联的 C 菜单 `apis` 去重写策略。

## 决策（已与用户确认）

- **范围：标准补全**。新增：`admin` 仪表盘顶级；`system` 下 `dict`、`notice`；`log` 日志顶级 + `loginlog` + `operlog`。**不加** `disk`/`gateway`/`server`（前端 routes.ts 中无 order、无 icon 的占位路由）。
- **apis 策略：按规范预填**。`loginlog` 后端 API 已实现 → 填真实路径；`dict`/`notice`/`operlog` 后端 API 尚未实现 → 按 RuoYi 规范路径预填（接口落地后 casbin 自动生效，无需再改菜单）。
- **对齐前端**：`system` 顶级 `icon`→`carbon:cloud-service-management`、`order`→2；`setting` 子菜单 `order`→8。子菜单 order 全部对齐前端 routes.ts 的 `meta.order`。

## MenuId 规划（不与既有冲突）

- 顶级：`admin`=1（单页 C，无子，前端 `/admin`）；`system`=100（M）；`log`=200（M）
- system 子：user=1100 / role=1200 / menu=1300 / dept=1400 / post=1500 / **dict=1700(+1701-1705)** / **notice=1800(+1801-1803)** / setting=1600（order 8）
- log 子：**loginlog=2100(+2101-2103)** / **operlog=2200(+2201-2203)**

## 预填的 apis（RuoYi 规范）

- dict：`/system/dict/type/list`(GET)、`/system/dict/type`(POST/PUT)、`/system/dict/type/:ids`(DELETE)
- notice：`/system/notice/list`(GET)、`/system/notice`(POST/PUT)、`/system/notice/:ids`(DELETE)
- operlog：`/log/operlog/list`(GET)、`/log/operlog/:action`(DELETE)（与 loginlog 一致，合并 clean 与批量删除）
- loginlog（真实）：`/log/loginlog/list`(GET)、`/log/loginlog/:action`(DELETE)、`/log/loginlog/unlock/:username`(GET)

## 后端接口实现现状（影响 apis 真实性）

已实现真实 API：`user`、`setting`、`loginlog`。**仅有 model + seed、API/router/service 尚未实现**：`dict`、`notice`、`operlog`，以及 `menu`/`role`/`dept`/`post` 的 CRUD 接口。

## 2026-07-18 核对纠正（apis 机制未落地）

核对代码发现：本文档"关键约束"描述的"C 菜单 apis → casbin 策略"机制**未落地**——`SysMenu` 无 `Apis` 字段、`service/system/sys_casbin.go` 不存在、无 `UpdateCasbin`、`CasbinHandler` 在 `initialize/router.go:79` 被注释。当前 `source/system/sys_menu.go` 仅 seed 菜单(M/C)与按钮(F, 带 `Perms`)；`Perms` 供前端按钮显隐(`useAuth`)，**不参与 API 级鉴权**。启用 casbin 是独立功能，需先实现 apis 字段 + 策略推导 + 策略表初始化。本文档 07-15 写作时 apis 部分为规划态，实际代码未含。

## 相关

- [[menu-management]] 菜单模型建模（SysMenu + DTO + i18n）
- [[dict-management]] 字典管理（前端 i18n 已落地，后端延后）
