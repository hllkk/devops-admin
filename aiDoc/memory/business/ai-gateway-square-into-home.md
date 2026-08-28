# AI 网关·模型广场并入 home 第三个 Tab

- 日期：2026-08-28
- 状态：已实现（go build/test + typecheck 通过）
- 反向链接：[[ai-gateway-model-visibility-consumption]]、[[ai-gateway-resource-application]]、[[ai-gateway-usage-log-page]]

## 需求

用户提出：模型广场是纯用户侧页面（浏览可见模型/查看接入信息/申请订阅），不该展示在 AI 网关后台管理菜单里，应展示在 `/home` 个人门户中。经确认采用**第三个 Tab「模型广场」**方案（与「我的应用」「我的AI身份」并列），侧边栏删除模型广场菜单、删除 square 独立路由与页面。

## 设计决策

- **页面形态**：home 顶部 Tab 从 2 个扩为 3 个；广场内容组件化为 `web/src/views/home/modules/model-square-panel.vue`（含搜索/卡片网格/接入信息弹窗/申请订阅弹窗），逻辑从 `_gateway/square/index.vue` 原样搬迁，i18n key 从 `page.gateway.square.*` 迁移为 `page.home.square.*`（新增 `tab` key；删除组件从未引用的 `contactAdmin`）。
- **数据共享**：`identity`（identity/my）由 home 主页统一加载后经 prop 传入面板，避免与身份卡重复请求；面板挂载时只拉 `model/active` + 我的申请(pending)；**懒挂载**（`squareLoaded` ref，首次切到该 Tab 才挂组件）避免 home 首屏请求堆积。申请提交成功 `emit('applied')` → home 重拉「我的申请」列表。
- **「我的申请」空态按钮**：`goSquare()` 由 `routerPushByKey('square')` 改为 `activeTab='square'`（切 Tab，不再走路由跳转）。
- **布局适配**：home 容器 `max-w-5xl`，卡片网格 `grid-cols-1 sm:grid-cols-2 lg:grid-cols-3`（原独立页 4 列）；弹窗不受容器宽度影响。
- **后端菜单清理**：sys_menu.go 删 `route.square` 块；sys_role_menu.go 删 user 角色的 square 授权（`squareMenuID`）；casbin 白名单注释同步（`/gateway/model/active` 本就在登录白名单，接口权限与菜单解耦，零策略改动）。
- **已有库补丁**：`deploy/patches/2026-08-28-square-menu-removal.sql`（条件 DELETE：sys_role_menu → casbin_rule(`/gateway/model/active` 策略仅由 square ApiPrefix 派生，删之保一致) → sys_menu；幂等、执行后重启）。同 [[ai-gateway-usage-log-page]] 待办：当前 dev 未上线，暂不执行，留作生产参考。
- **elegant 路由四件**：vite 插件监听 views 目录自动重生成（dev server 运行中，删目录即生效），`route.square`/RouteKey/RouteMap 同步收敛。

## 边界

- 接口 `/gateway/model/active`、`identity/my`、`application/apply|my` 均在 casbin 登录白名单，本次后端零接口改动。
- user 角色不再需要 square 菜单授权即可用广场（home 本就是全员基础页，身份 Tab 数据同样可达）。
