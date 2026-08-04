# module 与 component 职责分离（布局由 component 驱动）

> 日期：2026-07-26 ｜ 状态：已实现

## 需求

此前 `SysMenu.Component` 是**死字段**：菜单管理录入的 component（RuoYi 风格 `Layout`/`_admin/system/user/index`/`layout.blank$view.xxx`）后端 `menusToRoutes` 完全不读，component 按 menuType/层级硬编码推导（`layout.base`/`view.<key>`）。后果：菜单管理选的"空白布局"实际不生效——布局与模块归属耦合。

改为：**module 仅做模块归属（菜单隔离 + 角色授权树分组），component 决定布局外壳与 view 路径**。后端真正消费 component，前端解耦。

## 改动

### 后端（`service/system/sys_route.go`）
- 新增 `resolveLayout(component)`：从 component 提取布局外壳——含 `layout.blank`→`blank`、其余（`Layout`/`xxx/index`/`FrameView`/空）→`base`
- `menusToRoutes` component 决策改为「布局外壳读 `SysMenu.Component`(resolveLayout)，view 段读 `routeKey(Path)`」：顶层有子→`layout.<layout>`、顶层无子→`layout.<layout>$view.<key>`、子级→`view.<key>`
- 单测 `sys_route_test.go` 加 `TestResolveLayout` + `TestMenusToRoutesLayout`（验证脏 view 段被 routeKey 覆盖、blank 布局提取）

### 前端
- **删 `applyModuleLayout`**（`router/routes/index.ts`），`getAuthVueRoutes` 直接 transform（布局外壳完全由后端下发的 component 决定，前端不再二次改写）
- **`MODULE_CONFIG` 删 `preset`/`LayoutPreset`**（`constants/module.ts`），只留 `home`/`icon`（preset 唯一消费点 applyModuleLayout 已删）

### seed（`source/system/sys_menu.go`）
- server/gateway 保持 base 布局，不动

## 关键约束

- **view 段始终由 `routeKey(Path)` 规范化（下发层）**：后端 `menusToRoutes` 用 `routeKey(Path)` 覆盖 view 段下发（如 `layout.base$view.server`），DB 里 component 的 view 段**不影响运行时路由**，只供菜单管理展示/回显。
- **DB 的 view 段保留原始路径（带斜杠），不要 `replaceAll('/','_')`**：前端 `processComponent` 对 blank 布局生成 `layout.blank$view.<原始路径>`，详情/回显 `slice(18)` 直接取。曾用 `replaceAll('/','_')` 编码是**有损**的——前导下划线和路径分隔下划线无法区分，回显 `replaceAll('_','/')` 会多出前导斜杠。原始路径无损。
- **详情展示处理 blank 前缀**（`index.vue` 菜单详情 component 项）：`layout.blank$view.`(slice 18) 补 `/index` 展示，与 admin 默认布局（原样 `_admin/.../index`）一致。
- **布局完全由 component 决定，与 module 解耦**：module 不再触发布局。需要空白布局的菜单在菜单管理选"空白布局"（MenuLayout='1'），否则落 base 布局。
- **子级无外壳**：子级叶子 `view.<key>`，布局不生效，跟随父目录布局（多级目录的布局由根节点 component 决定）。
- **老库 component 兼容**：现有 RuoYi 值（`Layout`/`xxx/index`/`FrameView`/`layout.blank$view.xxx`）`resolveLayout` 全部识别（→ base/blank），无需 SQL 回填。

## 相关

- [[module-isolation-backend-driven]] module 字段来源（本次让 component 接管布局，module 回归纯归属）
- [[role-menu-group-by-module]] 角色授权树按模块分组（module 的归属用途，不变）
- [[menu-management]] 菜单模型 SysMenu（Component 字段语义本次激活）
