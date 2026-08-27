# AI 网关·套餐余量旁路 ProviderBalance（百炼 Token Plan Credits）

> 需求日期：2026-08-27。关联：[[ai-gateway-overview]]、[[ai-gateway-p1-progress]]
> 设计正文：`aiDoc/modules/ai-gateway-billing-integration.md`（本条为该 spec「套餐真实余量旁路」一节的落地记录）

## 背景

用户要求落地百炼 Token Plan 的 Credits 计费机制。设计稿已定三条原则：厂商计量折 ¥ 套现有算路（Credits 折 ¥ 塞部署层 `model_info` 四键，纯用法零代码）、标价成本与套餐真实余量分离、余量走旁路只读。

Token Plan 总消耗获取途径（实测+官方 OpenAPI 元数据确认，2026-08-27）：
- **`GetSubscriptionSeatDetails`**（`GET modelstudio.cn-beijing.aliyuncs.com/tokenplan/subscription/seat-detail`，阿里云 AK/SK，ACS3 签名）：每坐席 `EquityList[CREDITS]` 带 `CycleTotalValue`/`CycleSurplusValue`，**已用 = 总 − 剩余**
- **`ListSubscriptionSharedPackages`**（`.../shared-packages` 注意复数）：共享包同构
- **组织总消耗 = Σ坐席已用 + 共享包已用**；`GetTokenPlanAccountDetail` 只有组织结构无 Credits
- `bl token-plan list-seats` 即坐席接口的 CLI 封装；`bl usage token-plan`（1.18.0+）是**个人版**口径 + Console 浏览器鉴权，不能用于服务端定时采集，别混用

## 已实现（go build/test、typecheck、eslint 全过）

### 后端
- `server/model/gateway/provider_balance.go`：`gateway_provider_balance` 快照现状表（每 `(provider_id,item_type,item_key)` 一行，同步整批 Unscoped DELETE+INSERT 重建，同 CostSummaryDaily 派生缓存模式；OPS_BASE+自增主键）+ `BalanceSyncConfig`（AK/SK/region 明文结构）+ 白名单 `BalanceSyncProviderTypes{dashscope}`
- `gateway_provider` 加列 `balance_sync_config`（text，AES-256-GCM 密文，密钥复用 `litellm.credential-key`）
- `server/service/gateway/provider_balance.go`：GetProviderBalances/GetBalanceSummaryAll（COUNT FILTER 汇总）/配置读写（`MaskSecret` 掩码占位保留旧明文，对齐凭证 MergeCredentialValues 语义）/`SyncProviderBalance`（翻页拉取+事务重建，normalizeUnixMillis 秒/毫秒自适应）/`SyncAllProviderBalances`（逐家同步失败不阻断）
- ACS3-HMAC-SHA256 签名纯函数 `acs3CanonicalRequest`/`acs3Authorization` + `canonicalQueryString`（字典序+RFC3986），单测覆盖
- API：`GET /gateway/provider/:id/balance`、`GET|PUT .../balance-config`、`POST .../balance-sync`（超时前端 timeout:0）、`GET /gateway/dashboard/balance-summary`（**非超管返回空数组**——供应商资产属管理视角）
- `task.Register("SyncProviderBalances")` + 种子 `17 8 * * *`（每日，旁路失败不阻断）；菜单 api_prefix `/gateway/provider/*`、`/gateway/dashboard/*` 已覆盖，未动菜单

### 前端
- `provider-balance-panel.vue` 挂供应商页右侧（仅 `BALANCE_SYNC_PROVIDER_TYPES.has(providerType)` 即 dashscope 显示，与凭证面板纵向堆叠 flex，余量面板 flex-shrink-0）
- `dashboard-balance.vue` 看板汇总卡（DashboardBudget 上方，非超管后端空数组→前端仍渲染空态卡）
- i18n 三处同步（zh-cn/en-us/app.d.ts，`page.gateway.balance.*`）

## 硬边界（不变）

不进 calcCosts、不触发 enforceBudgetHardLimit、不并入 gateway_cost_summary_daily；卡片显式标注"厂商侧口径，与网关标价成本互不并"。

## 待办

- 真实数据验证：需在面板录入有百炼权限的阿里云 AK/SK（RAM 需 `AliyunBailianFullAccess` 或只读等价）后手动同步
- 设计稿 P3 其余项未动：`model_info.billing_unit_note` 展示键、智谱余量接入（≥2 家前不扩）
