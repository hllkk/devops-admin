# 品牌改名 Devops Admin → AIOps

- 日期：2026-09-16
- 状态：已实现（go build/test + swag 重新生成 + eslint/typecheck 通过）
- 关联：[[notify-wecom-push]]（测试消息文案消费方）

## 需求

展示层品牌名统一改 AIOps：AI 网关菜单顶部 = **AIOps Gateway**、后台管理菜单顶部 = **AIOps Admin**、登录页与全局 loading 与标签页标题 = **AIOps**。

## 实现

- i18n `system.title` → `'AIOps'`（登录页两处/全局 loading `plugins/loading.ts`/about 弹窗 fallback 全部随之生效）；新增 `system.adminTitle`/`gatewayTitle`（zh/en/app.d.ts 三处同步）
- `global-logo/index.vue` 动态化：`route.meta.module === 'gateway'` → `AIOps Gateway`，其余（admin/server/未带模块）→ `AIOps Admin`（`RouteModule` 类型与 `constants/module.ts` 现成）
- `web/.env` `VITE_APP_TITLE=AIOps`（浏览器标签 title）
- `server/global/version.go` `AppName = "AIOps"`（关于弹窗/升级接口消费方）
- `server/main.go` swagger `@title/@description` 改 AIOps + `swag init --parseDependency --parseInternal` 重新生成 docs（顺带补齐此前欠账端点）
- `sys_notify_send.go` 企微测试消息两处 →「AIOps 测试消息」

## 明确不改的技术标识（改动即断链）

Go module 名、镜像名 `devops-admin/*`、发布包名、`parseDevopsUserId` 的 `devops_user_{id}`（LiteLLM 用量归因协议）、init 页 `dbName: 'devops_admin'`（IndexedDB）、web package.json name（soybean-admin 上游名）、`setting/index.vue` 注释中的历史默认值记录。
