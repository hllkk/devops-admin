# module 与 component 职责分离（布局由 component 驱动）

> 日期：2026-07-26 ｜ 状态：已实现

## 需求

此前 `SysMenu.Component` 是**死字段**：菜单管理录入的 component（RuoYi 风格 `Layout`/`_admin/system/user/index`/`layout.blank$view.xxx`）后端 `menusToRoutes` 完全不读，component 按 menuType/层级硬编码推导（`layout.base`/`view.<key>`）。后果：菜单管理选的"空白布局"实际不生效；disk 布局更无录入入口，靠前端 `applyModuleLayout` 用 `module==='disk'` 在前端把 `layout.base` 改写成 `layout.disk` 补救——布局与模块归属耦合。

改为：**module 仅做模块归属（菜单隔离 + 角色授权树分组），component 决定布局外壳与 view 路径**。后端真正消费 component，前端解耦。

## 改动

### 后端（`service/system/sys_route.go`）
- 新增 `resolveLayout(component)`：从 component 提取布局外壳——含 `layout.blank`→`blank`、含 `layout.disk`→`disk`、其余（`Layout`/`xxx/index`/`FrameView`/空）→`base`
- `menusToRoutes` component 决策改为「布局外壳读 `SysMenu.Component`(resolveLayout)，view 段读 `routeKey(Path)`」：顶层有子→`layout.<layout>`、顶层无子→`layout.<layout>$view.<key>`、子级→`view.<key>`
- 单测 `sys_route_test.go` 加 `TestResolveLayout` + `TestMenusToRoutesLayout`（验证脏 view 段被 routeKey 覆盖、blank/disk 布局提取）

### 前端
- **删 `applyModuleLayout`**（`router/routes/index.ts`），`getAuthVueRoutes` 直接 transform；新增 `tagLayoutMeta` 给 component 含 `layout.disk` 的路由打 `meta.useDiskLayout=true`（深拷贝 meta + `as typeof next.meta` 断言绕过展开后 title 窄化与 RouteMeta.title 必填差异；vue-router 合并 matched meta，子路由自动继承）
- **`MODULE_CONFIG` 删 `preset`/`LayoutPreset`**（`constants/module.ts`），只留 `home`/`icon`（preset 唯一消费点 applyModuleLayout 已删）
- **菜单录入加"网盘布局"**：`menuLayoutRecord` 加 `'2'`（`constants/business.ts`）+ `MenuLayout='0'|'1'|'2'`（`system.api.d.ts`）；`menu-operate-drawer.vue` `processComponent` 加 disk 分支（`layout.disk$view.<path下划线>`），`module==='disk'` 新建时默认勾网盘布局，编辑回显识别 `layout.disk$view.` 前缀（slice 17）
- **`effectiveLayoutMode` 信号迁移**（`store/modules/theme/index.ts`）：从 `currentModule==='disk'` 改为 `router.currentRoute.value.meta?.useDiskLayout`；`typings/router.d.ts` RouteMeta 加 `useDiskLayout` 声明
- **修连带 bug**：`handleLayoutChange` 原把 `layoutType` 值直接赋给 `visible`，加 `'2'` 后会设成非法值，改为仅在空白布局(`'1'`)时 `visible='1'`，其余 `'0'`

### seed（`source/system/sys_menu.go`）
- disk 占位首页 component `_disk/disk/index` → `layout.disk$view._disk_disk`（与前端 disk 布局录入格式一致；后端 resolveLayout 识别 → 顶层单级生成 `layout.disk$view.disk`，view 段被 routeKey 覆盖）
- server/gateway 保持 base 布局，不动

## 关键约束

- **view 段始终由 `routeKey(Path)` 规范化（下发层）**：后端 `menusToRoutes` 用 `routeKey(Path)` 覆盖 view 段下发（如 `layout.disk$view.disk`），DB 里 component 的 view 段**不影响运行时路由**，只供菜单管理展示/回显。
- **DB 的 view 段保留原始路径（带斜杠），不要 `replaceAll('/','_')`**：前端 `processComponent` 对 disk/blank 布局生成 `layout.disk$view.<原始路径>`（如 `layout.disk$view._disk/disk`），详情/回显 `slice(17/18)` 直接取。曾用 `replaceAll('/','_')` 编码成 `_disk_disk` 是**有损**的——`_disk` 前导下划线和路径分隔下划线无法区分，回显 `replaceAll('_','/')` 还原成 `/disk/disk` → `views//disk/disk/index.vue` 双斜杠。原始路径无损。
- **详情展示要同时处理 disk/blank 前缀**（`index.vue` 菜单详情 component 项）：`layout.disk$view.`(slice 17) 与 `layout.blank$view.`(slice 18) 各补 `/index` 展示，与 admin 默认布局（原样 `_admin/.../index`）一致。
- **布局完全由 component 决定，与 module 解耦**：module 不再触发布局。disk 模块的菜单要在菜单管理选"网盘布局"（disk 模块默认勾）才有 disk 外壳；老库 disk 菜单若 component 未编码 disk 则落 base 布局（需 seed 同步或迁移 SQL）。
- **子级无外壳**：子级叶子 `view.<key>`，布局不生效，跟随父目录布局（多级目录的布局由根节点 component 决定）。
- **老库 component 兼容**：现有 RuoYi 值（`Layout`/`xxx/index`/`FrameView`/`layout.blank$view.xxx`）`resolveLayout` 全部识别（→ base/blank），无需 SQL 回填；唯独 disk 模块菜单若想要 disk 布局需显式录 `layout.disk` 编码。

## 相关

- [[module-isolation-backend-driven]] module 字段来源（本次让 component 接管布局，module 回归纯归属）
- [[role-menu-group-by-module]] 角色授权树按模块分组（module 的归属用途，不变）
- [[menu-management]] 菜单模型 SysMenu（Component 字段语义本次激活）
