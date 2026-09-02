# AI 网关·用量日志保留期清理

- 日期：2026-09-01
- 状态：已实现（build/vet/test 全过，待运行时验证）
- 关联：[[ai-gateway-mcp-usage-sync]]（MCP 回流管线）、[[ai-gateway-usage-log-page]]（调用日志页）、[[ai-gateway-cost-analysis]]（成本分析依赖聚合表长历史）

## 需求

LLM/MCP 调用日志回流上线后 `gateway_llm_log`/`gateway_mcp_log` 无限增长，补齐保留期清理（对标 AIHelms `llm_log.cleanup`，每日 04:23）。

## 对比结论（本次分析的出发点）

漏单对账（增量游标 + 每小时差集回灌）devops-admin 全面优于 AIHelms；AIHelms 剩余可借鉴点中**日志保留期清理是 devops-admin 唯一真缺口**，另两点（LiteLLM 库游标索引、游标与落库同事务）收益小暂不做。

## 用户决策

- prod 配置默认 90 天（目的就是止住增长，留 0 等于忘了配就白做）；代码默认 0=不清理（删除业务数据的默认行为应显式开启），dev 保持 0。
- 不发 patch SQL：`deploy/patches/` 目录已在历史提交中整体移除，已有库同步走「定时任务面板手动创建」惯例（种子 Description 已标注）。

## 实现要点

- **配置** `litellm.log-retention-days`（config/litellm.go）：0/负=禁用；prod 模板 `deploy/docker-prod/config/config.yaml` 显式 90。
- **生效值联动对账窗口**（AIHelms 没有的防御）：`effectiveRetentionDays = max(配置值, log-reconcile-window+7)`——保留期 < 对账窗口时已清行会被 `ReconcileLLMLogs`/`ReconcileMcpLogs` 从 SpendLogs 重灌，形成"删了又灌"抖动循环。纯函数 4 组单测。
- **`CleanupUsageLogs`**（usage_sync.go）：cutoff=now-生效天数，两表物理删（Unscoped，同聚合表派生缓存口径）。返回 `{retentionDays, llmDeleted, mcpDeleted}`。
- **分批删除**（比 AIHelms 一条 DELETE 稳）：`DELETE WHERE log_id IN (SELECT log_id WHERE started_at<cutoff ORDER BY log_id LIMIT 5000)` 循环至不足一批，单表单轮上限 200 批（100 万行）防首启大存量的长事务/WAL 洪峰。两表 `started_at` 均有索引。
- **定时**：种子 spec `23 4 * * *`（避开 03:37/08:17 既有任务）；timer.go 注册 Method executor。
- **不清的东西**：`gateway_sync_state`（游标在最新端）、`gateway_cost_summary_daily`（滚动重建只动近 60 天，老行是成本分析长尾数据源——明细可查期 90 天 < 聚合历史）、`skill_usage_log`（本地记账非回流日志）。

## 保留期下限依据（90 的由来）

| 约束 | 窗口 | 原因 |
|---|---|---|
| 预算滚动重算 | 最长 30d | recomputeBudgetUsed 从 LlmLog+McpLog SUM，清了预算用额缩水、硬限失真 |
| 聚合重建 | 60d | 超窗老聚合行保留，成本长历史靠聚合表 |
| 对账回灌 | 30d | 生效值联动兜住 |

## 运维

- 已有库启用：定时任务面板手动建任务选 `CleanupUsageLogs`（或 INSERT sys_timed_task 行）。
- 调整保留期改 config.yaml `log-retention-days`；关闭设 0。
- 观测：每次执行 Info 日志含 retention/cutoff/llmDeleted/mcpDeleted。
