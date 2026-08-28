# AI 网关·调用日志明细页 + 看板调用量增强

- 日期：2026-08-27
- 状态：已实现（go build/vet/test + typecheck/eslint 全过）
- 反向链接：[[ai-gateway-overview]]、[[ai-gateway-p1-hardening]]

## 需求

上线前管理员要"关注用户的调用量"：后端 `GET /gateway/usage/list`（slice5 就绪）前端零使用且不在任何菜单 api_prefix——每笔调用明细看不到；看板 Top 榜仅成本排序、趋势图无 token 曲线。

## 设计决策

- **调用日志页**：新顶层菜单 `route.usage`（OrderNum 11，`_gateway/usage/index`），ApiPrefix `/gateway/usage, /gateway/usage/*`（含手动回流/对账按钮的 POST，均管理员操作）。
  - **权限边界**：usage 前缀不入 casbin 登录白名单、user 角色不授本菜单（普通用户不能经它查全量明细）；super/admin 由 sys_role_menu 种子循环全菜单自动获得，sys_role_menu.go 零改动。
  - **后端视图化**：`GetUsageLogList` 返回从 `[]LlmLog` 改为 `[]LlmLogView`（+userName/aiKeyName/deploymentName 回填），`fillLlmLogNames` 每页三次 IN 查询（sys_users.nick_name/gateway_ai_key.name/gateway_model_deployment.deploy_name）避免 N+1；metadata 不出网。
  - **前端筛选**：datetimerange（→RFC3339 `toISOString`，后端按 RFC3339 解析）+ 用户下拉（fetchGetUserSelect 懒加载，filterable）+ 模型模糊 + 供应商精确；表格列=时间/用户/密钥/模型/渠道/调用类型/输入·输出Token/成本/耗时；header 带「立即回流」「漏单对账」按钮（Popconfirm，timeout:0）。
- **看板增强**：`TrendItem`/`TopItem` 加 `tokens`；`GetTrend` 聚合 `SUM(total_tokens)`；`GetTop` 加 `sort` 参数（cost/requests/tokens，map 白名单→Order 列，默认 cost），api 层 `c.DefaultQuery("sort","cost")` 透传。前端 dashboard-trend metric 单选加 Token 档；dashboard-top 加排序单选组+Token 列（toLocaleString），gateway/index.vue 挂 `topSort` ref 双 watch。
- **已有库补丁 SQL**：`deploy/patches/2026-08-27-gateway-menus.sql`（square+usage 菜单 NOT EXISTS 幂等插入、super/admin 全授+user 仅 square、casbin_rule 直插三 pattern、执行后需重启服务加载菜单与 casbin）。menu_id 手造雪花风格值（202608271100000000x），与运行时生成区间不重叠。
- elegant 路由四件（imports/routes/transform/elegant-router.d.ts）由 vite 插件监听 views 目录自动重生成，创建页面文件即得，无需手补。

## 待办（关联）

- ~~执行补丁 SQL~~：当前为 dev 系统、尚未上线，用户决定稍后重新初始化系统（种子全量生效），不执行 `deploy/patches/2026-08-27-gateway-menus.sql`（2026-08-27 确认）。文件保留供未来生产参考。
- ~~ProviderBalance 实测~~：已由用户实测通过。
- 模型广场、调用日志页的 UI/功能优化：用户计划后续自行慢慢迭代，不作为 AI 侧主动待办。
