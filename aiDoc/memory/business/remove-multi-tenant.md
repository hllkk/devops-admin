# 清理基座多租户（Multi-Tenant）残留代码

- 日期：2026-07-12
- 状态：已实现（前端清理 + typecheck 全绿），后端无多租户代码、无需改动
- 关联分支：feat/multi-module-isolation

## 背景

本仓库是基座项目。前端（SoybeanAdmin 二开）的登录/注册/社交登录链路中残留了一套多租户选择逻辑（`tenantId` + 租户下拉 + `/auth/tenant/list` 拉取），但后端 Go 服务**完全没有**多租户实现（无模型/迁移/路由/中间件，`/auth/tenant/list` 端点根本不存在）。属前端单边遗留，需清理干净，保持基座单租户。

## 范围盘点（结论）

- **后端 `server/`**：`grep -rni tenant` 全仓 **0 命中**。无任何多租户代码，不动。
- **前端 `web/`**：共 **12 个文件**有 `tenant` 引用，**全部是内联引用**（嵌在更大的 auth/typing 文件里），无独立可整删的租户模块/页面/store/路由/i18n。逐一外科式删除。

## 改动清单

**TypeScript 类型（4 文件）**
- `typings/api/auth.d.ts`：删 `LoginForm.tenantId`、`Tenant`、`LoginTenant` 三个接口。
- `typings/api/api.d.ts`：删 `CommonTenantRecord<T>` 类型。
- `typings/api/system.api.d.ts`：`User` 基类型 `CommonTenantRecord` → `CommonRecord`；删 `Social.tenantId`、`Oss.tenantId`。
- `typings/storage.d.ts`：删 `Local.tenantId`。

**API service（2 文件）**
- `service/api/auth.ts`：删 `fetchTenantList()`（`GET /auth/tenant/list`）。
- `service/api/system/social.ts`：`fetchSocialAuthBinding` 去掉 `tenantId` 形参与 `params.tenantId`，仅留 `domain`。

**Store / Hook（2 文件）**
- `store/modules/auth/index.ts`：`login()` 拼 `loginData` 时删 `tenantId: loginForm.tenantId ?? '000000'`。
- `hooks/common/form.ts`：删 `formRules.tenantId` 规则。

**Vue 页面（4 文件）**
- `views/_builtin/login/modules/pwd-login.vue`：删租户下拉 UI、`tenantEnabled`/`tenantOption`/`tenantLoading`、`handleFetchTenantList`、`RuleKey` 中 `tenantId`、记住密码 payload 的 `tenantId`、`fetchSocialAuthBinding` 的 `tenantId` 实参；同步移除不再使用的 `SelectOption` 类型导入与 `fetchTenantList` 导入。
- `views/_builtin/login/modules/register.vue`：同上模式清理；移除 `SelectOption`、`fetchTenantList` 导入。
- `views/_builtin/social-callback/index.vue`：删从 social state base64 解出的 `tenantId` 提取与回传。
- `views/_builtin/user-center/modules/social-card.vue`：`fetchSocialAuthBinding` 去掉 `userInfo.user?.tenantId` 实参；连带删除因此变成无用的 `useAuthStore` 导入与 `userInfo`。

## 默认值 `000000`

原前端在 5 处硬编码默认租户号 `'000000'`（auth store、social.ts、pwd-login、register、social-callback），随清理一并移除。

## 验证

- `grep -rni tenant web/src`（含 .ts/.vue/.d.ts/.tsx/.js）→ **0 命中**。
- `pnpm --dir web typecheck`（`vue-tsc --noEmit --skipLibCheck`）→ **exit 0**，无类型错误。
- 后端登录请求体不再含 `tenantId`；后端 auth 请求结构体本就无该字段，登录不受影响。

## 注意 / 演进

- 社交登录的 `state` 透传 blob 原含 `tenantId`；清理后仅保留 `domain`。若未来后端 `/auth/social/callback` 需要租户上下文，需重新设计（当前后端无此需求）。
- `loginRember` localStorage 旧值可能仍含历史 `tenantId` 字段；`Object.assign(model, ...)` 会带上多余键但无害（`PwdLoginForm` 已无 `tenantId`），无需迁移。
- `service/api/index.ts` barrel 经 `export * from './auth'` 自动不再导出 `fetchTenantList`，无需改动。
- `CommonTenantRecord` 已删，若将来需要「带租户列的通用记录类型」须重新定义。
