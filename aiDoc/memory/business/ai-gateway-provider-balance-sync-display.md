# AI 网关·套餐余量同步时间显示修复 + 进面板自动同步

> 需求日期：2026-09-03。关联：[[ai-gateway-provider-balance]]（余量旁路首期落地）、[[ai-gateway-overview]]

## 背景

用户反馈两个问题（供应商管理页余量面板）：

1. **配了采集 AK/SK、点了同步，"最近同步"始终显示"从未同步"**。根因：`summary.syncedAt` 取自快照表 `gateway_provider_balance` 的 `synced_at DESC` 首行，而同步事务在厂商侧返回空（账号无坐席/无共享包）时**不写任何行**——同步时间依附于快照行数，快照为空就永远查不到时间。
2. **每次都要手点同步按钮很扯**，要求进入供应商管理页自动同步一次。

## 已实现（go build/test、typecheck、swag 重新生成全过）

### 修复：同步时间与快照行解耦

- `gateway_provider` 加列 `balance_synced_at`（`Provider.BalanceSyncedAt *time.Time`，`json:"-"` 不出网，AutoMigrate 自动加列）
- `SyncProviderBalance` 事务内在重建快照后同事务 `Update("balance_synced_at", now)`——厂商侧确无坐席/共享包也记录"已同步"
- `buildSummaryFromProvider` 的 `SyncedAt` 优先取 `p.BalanceSyncedAt`；为空回退快照行查询（兼容加列前存量数据，下次同步后自然切到新列）

### 自动同步（auto 静默模式）

- `SyncProviderBalance(ctx, id, auto bool)`；`POST /gateway/provider/:id/balance-sync?auto=true`
- auto 语义（三吞一节流，手动路径行为完全不变）：
  - 类型不支持 / 解密失败 / 未配置 AK/SK → 直接返回当前快照 summary，不报错（前端进面板不弹错）
  - **节流**：`p.BalanceSyncedAt` 距今 < `balanceAutoSyncMinInterval`(5min) → 跳过真实拉取返回现状（防频繁进出页面/切供应商打爆百炼 OpenAPI 配额）
  - 拉取/写库失败 → Error 日志 + 返回当前快照（自动场景不打扰用户；手动同步照常报错）
- `SyncAllProviderBalances` 定时任务传 `false`（每日 17:08 全量真同步，不受节流影响）
- 前端 `provider-balance-panel.vue`：`watch(providerId, immediate)` 时 `getBalanceData()` + `autoSync()` 并行——先显示旧快照、后台静默同步完成后刷新；`autoSync` 复用 `syncing` 状态（按钮转圈可见）；`fetchSyncProviderBalance` 加 `auto` 参数（`params: { auto: true }`）
- 手动"同步"按钮走非 auto，总是真实拉取 + 成功提示，行为不变

## 边界与注意

- 节流仅对 auto 生效；手动同步永不受节流
- 供应商切换即触发 autoSync（面板 watch providerId），5 分钟节流兜底
- 本次顺带重新生成 swagger docs，补齐近几个提交（adoption/health/report/MCP）的 docs 欠账
