# home 默认 Tab 落「我的AI身份」+「我的应用」按权限收敛

- 日期：2026-08-28
- 状态：已实现（typecheck 通过）
- 反向链接：[[ai-gateway-identity-home]]、[[ai-gateway-square-into-home]]、[[ai-gateway-model-visibility-consumption]]

## 需求

用户提出：所有用户打开 `/home` 第一时间应看到「我的AI身份」页；只有具备后台管理（admin）/ 服务器管理（server）/ AI 网关后台管理（gateway）任一模块权限的用户，才展示「我的应用」Tab。

## 分析结论

权限判定**无需新增任何后端逻辑**：`getUserInfo` 下发的 `apps` 字段（`service/system/sys_user.go` `GetUserDetail`）即按「角色 → sys_role_menu → sys_menu 按 module 去重」聚合（超管=全部模块；某模块下有任意授权菜单即算有该模块权限），与需求语义严格等价。前端 `home/index.vue` 的 `myApps` computed 已消费该字段（含与 `ALL_MODULES` 对齐的脏值兜底），直接复用。

## 实现改动

仅改 `web/src/views/home/index.vue`：

- `homeTabs` 由常量数组改为 computed：`apps` 项仅在 `myApps.length > 0` 时插入，Tab 栏 `v-for` 自动收敛——无模块权限的普通用户只见「我的AI身份」「模型广场」两个 Tab。
- `activeTab` 默认值 `'apps'` → `'identity'`，打开 home 默认落在 AI 身份页。
- 「我的应用」内容 section 加 `v-if="myApps.length"`，Tab 收敛时内容不留空壳 DOM。

## 边界

- `activeTab` 是组件内纯 ref 不与路由/query 同步，改默认值无状态恢复冲突；`goSquare()` 切 square Tab 不受影响。
- 零接口、零菜单、零 i18n key 改动。
