# AI 网关·全局路由策略 RouterSettings

> 需求日期：2026-08-26。关联：[[ai-gateway-overview]]、[[ai-gateway-deployment-dialog-terminology]]、[[ai-gateway-p1-progress]]

## 背景

用户指出"没有全局看路由策略"。对照本地 `/home/remote/AIHelms` 的 RouterSettings 实现，为 devops-admin 补齐「全局路由策略」层：此前路由池出入池（`__disabled__`/`(Anthropic)` 后缀）+ 同名聚合 LB 已就位，但策略选择/故障摘除/降级链全空（LiteLLM Router 一直跑默认值 simple-shuffle）。

三路分析确认要补的就是 AIHelms `RouterSettings` 那一层——管理面 CRUD + 同步 LiteLLM `/router/settings`，不碰转发。用户 AskUserQuestion 定调：轻量流程 / 全量 7 字段一次到位 / 前端嵌模型管理页弹窗（同 AIHelms）。

## 已实现（go build / typecheck / eslint 通过）

### 后端（Go+Gin+GORM，对齐 AIHelms RouterSettings 单例模式）
- `server/model/gateway/router_settings.go`：`gateway_router_settings` 单行表（id=1,FirstOrCreate），字段 `routing_strategy`/`fallbacks`(JSONB)/`allowed_fails`/`cooldown_time`/`num_retries`/`timeout`/`config`(JSONB)，基座 `OPS_MODEL`（配置表惯例，无审计，对齐 `sys_security_config`）。`FallbackItem` 结构 + `DefaultRouterSettings`。
- `server/model/gateway/request/router_settings.go`：`RouterSettingsUpdate`（整体覆盖）。
- `server/model/gateway/response/router_settings.go`：`RouterSettingsView`（剥离基座，对齐前端 `Api.Gateway.RouterSettings`）。
- `server/service/gateway/router_settings.go`：`Get`(FirstOrCreate 单例) + `Update`(事务落库 + 同步 LiteLLM)；`toLitellm` 投影——`fallbacks [{model,fallbacks}]→[{model:[fallbacks]}]`（List[Dict]，2026-11-11 修复，见下）、驼峰键→蛇形键；LiteLLM 未配置(sync-enabled=false)时静默跳过仅落库。
- `server/utils/litellm/client.go`：`GetRouterSettings`(GET) + `UpdateRouterSettings`（初版 POST `/router/settings`；v1.1.2 起 LiteLLM 1.104 该端点 POST 返回 405，改走 POST `/config/update` 按 section 写，热更新即时生效）。
- `server/api/v1/gateway/router_settings.go` + `server/router/gateway/router_settings.go`：`GET/PUT /gateway/router/settings`，Swagger 注释落到具体类型。
- 三处 `enter.go` 注册（service/router/api）+ `initialize/gorm.go` AutoMigrate + `initialize/router.go` 路由聚合。

### 前端（Vue3+NaiveUI+UnoCSS）
- `web/src/typings/api/gateway.api.d.ts`：`RouterSettings`/`RouterSettingsParams`/`RoutingStrategy`(5值联合)/`FallbackItem`。
- `web/src/service/api/gateway/model.ts`：`fetchGet/UpdateRouterSettings`。
- `web/src/constants/business/gateway.ts`：`ROUTING_STRATEGY_OPTIONS`（simple-shuffle/latency-based/cost-based/least-busy/usage-based）。
- `web/src/views/_gateway/models/model/modules/router-settings-dialog.vue`：`NModal card w-640px` 弹窗，策略下拉 + allowedFails/cooldownTime/numRetries/timeout `NInputNumber` + fallback 降级链动态行（源模型 `NSelect` + 降级多选，选项取现有 `modelKey` 去重）；`watch(visible)` 懒加载策略+模型列表。
- `models/model/index.vue` 的 `header-extra` 加「路由策略」按钮（`alt-route-outline` 图标）打开弹窗。
- i18n `page.gateway.router.*` 节三处同步（zh-cn / en-us / app.d.ts schema），否则 typecheck 必挂。

## 设计决策

- **单例全局策略（不按模型）**：与 AIHelms + LiteLLM `/router/settings` 语义一致；路由池仍靠 `model_key` 同名聚合 + 后缀摘除（已有机制），策略只控制"怎么选池中部署 + 故障摘除 + 降级"。
- **投影原则**：DB 存前端友好对象格式 `fallbacks [{model,fallbacks}]`，同步 LiteLLM 时 `toLitellm` 转 `[[model,[fallbacks]]]` 蛇形；驼峰键 DB / 蛇形键 LiteLLM（对齐 `deployment_payload.go` 的投影纯函数层思路）。
- **更新用 POST** `/config/update`（初版 `/router/settings`，1.104 起 POST 405 后迁移），热更新即时生效，无需重启。
- **config 字段保留**（DB/类型/同步），前端暂不配 UI（YAGNI），留扩展。
- 借鉴 AIHelms 规避：它没做的前端全局视图，本次也不另起菜单页（用户选弹窗嵌模型管理页，符合轻量）。

## 修复记录

- **2026-11-11 降级链同步 422**：用户配置降级链时报「POST /config/update 返回状态 422」。dev litellm(1.104.2) 复现定位：`/config/update` 的 pydantic 校验 `router_settings.fallbacks` 为 **`List[Dict[str, List[str]]]`**（每项 `{"主模型": ["备模型"...]}`，LiteLLM 文档标准形态）；旧投影的列表套列表 `[[model,[fallbacks]]]` 与裸 dict 均被拒（Router 运行时虽支持但 config API 不收）。修复：`toLitellm` 改产 list-of-dict + 无条件下发（空列表合法，语义=清空，顺带修复"清空降级链无法同步清除旧值"）+ 空模型名防御；`client.go do()` 错误信息带上 LiteLLM 响应体（原来只回状态码，排查要翻服务端日志）。端到端已在 dev 实测 200 并恢复基线。教训：**Router 运行时接受的 fallbacks 形态 ≠ config API 校验接受的形态，投影层以 API 校验为准**。

## 关联
- 蓝本：`/home/remote/AIHelms`（`apps/models/db.py` RouterSettings + `apps/services/model_service.py` update_router_settings + `apps/services/litellm_client.py`）
- 模块设计：`aiDoc/modules/business-modules.md`「AI 网关模块」节
- 同期术语切割：[[ai-gateway-deployment-dialog-terminology]]（「路由名」→「模型 ID」，为路由策略让路，避免术语冲突）
