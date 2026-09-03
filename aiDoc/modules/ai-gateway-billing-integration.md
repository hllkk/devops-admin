# AI 网关·订阅/积分制厂商计费集成 spec

> 状态：设计稿（2026-08-24）。「套餐真实余量旁路」一节已于 2026-08-27 落地（见文末「落地记录」）。
> 只聚焦"厂商计费五花八门怎么接"这一个点，不重述四层模型与成本算法全貌——见 `business-modules.md`「AI网关模块」节。

## 背景

厂商计费方式差异大：智谱历史 V2 按调用次数、新版按积分；百炼 token plan 用 Credits 统一计量。当前项目成本靠 `usage_sync.go:calcCosts` 两条算路（token/per_call），从 `deployment.model_info` 四键或 `CostPerCall` 算"标价虚拟成本"，不碰厂商真实扣费/套餐余量。调研 `/home/remote/AIHelms` 确认其同样未做积分/Credits/套餐余量（GAP），无可抄实现，本项目需自行定边界。

## 设计原则（三条，不可破）

1. **厂商计量统一折成¥，套现有算路**——不为积分/Credits/订阅新建数据模型或新算路。
2. **标价成本与套餐真实余量分离**——标价成本只算虚拟标价（预算卡口/趋势/归因用），套餐真实余量做旁路只读（采购决策用），两口径互不并。
3. **复用既有 ¥→USD 换算链路**——DB 永远 ¥，同步层 `ConvertCostsForLitellm` 换算 USD/token 下发，不另起口径。

## 映射规则：厂商计量 → 字段

| 厂商计量 | BillingType | 定价落点 | calcCosts 走哪条 |
|---|---|---|---|
| 智谱 V2（按次） | `per_call` | `CostPerCall`（¥/次） | per_call 分支 |
| 智谱新版积分（每次固定扣积分） | `per_call` | `CostPerCall` = 积分 × 积分单价(¥) | per_call 分支 |
| 智谱新版积分（按 token 折积分） | `token` | `model_info` 四键折 ¥/百万token | token 分支 |
| 百炼 Token Plan 团队版（Credits 制） | `token` | `model_info` 四键：Credits 消耗示例反推 × 坐席 ¥/Credits（见下「百炼 Credits 折算」） | token 分支 |

**关键**：积分/Credits 只是厂商的"虚拟币"，要做的事只有一件——定一个人工维护的"汇率"（积分单价 ¥/积分，或 Credits→token 比例）把它折成 ¥，塞进现有字段。没有自动同步汇率的机制。

### 百炼 Credits 折算（Token Plan 团队版）

来源：`help.aliyun.com/zh/model-studio/token-plan-team-overview` 的 Credits 计算示例。注意百炼有两套计费——按量付费是 token 计费（元/百万token），Token Plan 团队版是 Credits 制（以 Credits 统一计量、按 token 消耗抵扣），两者不可混。本节针对 Credits 制。

**两步折算**：

1. 从官方 Credits 消耗示例反推模型 Credits/百万token 单价。以 qwen3.6-plus 示例：输入 8349 token→1.67 Credits ⇒ ≈200 Credits/百万token；缓存读 40794→0.82 ⇒ ≈20 Credits/百万token；输出 573→0.69 ⇒ ≈1204 Credits/百万token。
2. 用坐席套餐 ¥/Credits 折成 ¥/百万token：标准 ¥150/25000=¥0.006/Credit，高级 ¥0.0055，尊享 ¥0.0056。

**qwen3.6-plus 四键（标准坐席口径，¥/百万token）**：`input_cost=1.2` `output_cost=7.2` `cache_read_cost=0.12` `cache_creation_cost=1.5`（缓存创建按输入 125% 规则借用，示例未直接给，标近似）。

**口径说明**：Credits/token 单价对所有坐席档位一致，仅 ¥/Credits 随档位变；不同模型 Credits 单价不同且无公开全量表，需逐模型从控制台账单或示例拿。算出的是"套餐摊销标价"，非按量单价，亦非财务对账口径。

## 字段变更

**不改现有表结构**——`gateway_model_deployment` 已有 `billing_type`(token/per_call) + `cost_per_call` + `model_info` 四键，覆盖全部场景。

仅追加一个**纯展示元数据键**（可选）：
- `model_info.billing_unit_note`（string）：口径说明，如"按积分""按 Credit""按次"。看板标注用，不参与计算。

## 同步层换算点（已存在，仅确认边界）

- `deployment_payload.go:ConvertCostsForLitellm`：¥/百万token → USD/token（÷ `usd_to_cny_rate` ÷ 1e6），只作用发往 LiteLLM 的副本，DB 永远 ¥。
- `MergeCostsToModelInfo`：四键镜像进 `model_info`（平台成本计算读这里）。
- 积分/Credits 折成 ¥ 塞 `model_info` 后**自动走这条换算，无需改同步层**。
- per_call 的 `CostPerCall`(¥) 不进 token 四键换算——但因项目"本地重算不信任 LiteLLM spend"，即便推给 LiteLLM 的 per_call 值口径不严丝合缝，也不影响项目成本正确性（`calcCosts` 读本地 `dep.CostPerCall`）。

## 套餐真实余量旁路（新地，已落地）

标价成本口径不含套餐真实余量，单独走旁路：

- **新表** `gateway_provider_balance`（套餐余量快照）：`provider_id` / `plan_type`(token_plan/subscription/credit) / `total` / `used` / `remaining` / `synced_at` / `raw`(JSONB 厂商原始返回)。
- **采集**：百炼 `bl token-plan list-seats`（走阿里云 AK/SK，非模型 Key）；智谱走后台 API（无 API 则人工录入或留空）；其余按需。
- **采集任务**：`SysTimedTask` 定时（如每日）拉取写快照，失败不阻断（旁路，不影响成本链路）。
- **展示**：看板独立"套餐余量"卡片，只读，显式标注"来自厂商侧，与网关标价成本口径不同"。
- **边界（硬约束）**：不进 `calcCosts`、不触发 `enforceBudgetHardLimit`、不并入 `gateway_cost_summary_daily`。

> 降级：若初期仅接百炼一家有余量 API，可暂不建表，先用配置/日志承载；待 ≥2 家再落表。

### 落地记录（2026-08-27）

百炼一家已直接落表（未走降级），明细见 `aiDoc/memory/business/ai-gateway-provider-balance.md`。要点：

- 表为**快照现状**语义：每 `(provider_id, item_type=seat/shared_package, item_key)` 一行，同步整批 DELETE+INSERT 重建（同 `CostSummaryDaily` 派生缓存模式），坐席粒度可归因到成员。
- 采集不走 bl CLI，Go 直连 OpenAPI：`modelstudio.cn-beijing.aliyuncs.com` + ACS3-HMAC-SHA256 签名（自实现，无 SDK 依赖），接口 `GetSubscriptionSeatDetails`(`/tokenplan/subscription/seat-detail`) 与 `ListSubscriptionSharedPackages`(`/tokenplan/subscription/shared-packages` 复数)。
- **总消耗口径**：每条目 `CycleTotalValue - CycleSurplusValue`（EquityList 中 EquityType=CREDITS），组织总消耗 = Σ坐席 + Σ共享包。时间戳秒/毫秒两义已自适应归一。
- AK/SK 存 `gateway_provider.balance_sync_config`（AES-256-GCM 密文，密钥复用 `litellm.credential-key`；出网掩码、保存掩码占位保留旧明文）。
- 展示双层：供应商管理页余量面板（配置+明细+手动同步，仅 dashscope 类型显示）+ 看板汇总卡（非超管后端返回空数组）。
- 定时任务 `SyncProviderBalances`（每日 08:17，失败不阻断）。
- **同步时间与快照解耦 + 进面板自动同步**（2026-09-03）：`gateway_provider.balance_synced_at` 列记录最近同步时间（厂商侧返回空也落，快照表无行不再导致"从未同步"假象）；`balance-sync?auto=true` 静默模式供面板挂载自动触发（未配置/5min 节流/失败均不报错返当前快照），明细见 `aiDoc/memory/business/ai-gateway-provider-balance-sync-display.md`。

## 口径与风险

- **标价成本 ≠ 套餐真实扣费**：套餐内调用的标价 ¥ 是虚拟成本，不等于套餐摊销。网关成本数字用于预算卡口/趋势/Top 归因，**不用于财务对账**。
- **积分汇率人工维护**：汇率变动需 `recalc` 全量回放成本（复用现有成本重算能力）。
- **`gateway_provider.monthly_budget/monthly_used`**：现 USD 口径且未闭环（与 AIHelms 同为空架子）。建议改 ¥ 统一口径并补 `monthly_used` 回填（从 `cost_summary_daily` 按 provider 维度 SUM），接供应商维度告警（不自动停，自动停留在 AiKey 层）。归属 P3。

## 分期归属

| 项 | 期 |
|---|---|
| 字段映射（积分/Credits 折 ¥ 塞定价键） | 随各厂商部署接入时（P1 已具备字段） |
| `model_info.billing_unit_note` 元数据键 | P3（看板标注时） |
| `gateway_provider_balance` 表 + 采集任务 + 看板卡片 | ~~P3~~ 已落地（2026-08-27，仅百炼；扩第 2 家时再抽象采集器接口） |
| Provider 月预算口径改 ¥ + 回填闭环 | P3（注：`monthly_budget/monthly_used` 字段已在部署层重构中移除，此项按部署层 `cost_per_call`/四键现状重新评估） |
