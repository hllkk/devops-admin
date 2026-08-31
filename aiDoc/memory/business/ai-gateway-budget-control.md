# AI 网关·P3 多维预算管控

- 日期：2026-08-31
- 状态：已实现（build/vet/test+typecheck/eslint 全过，待运行时验证）
- 关联：[[ai-gateway-mcp-usage-sync]]（MCP 回流纳入预算）、[[ai-gateway-cost-analysis]]（成本分析归因口径）、[[ai-gateway-p1-hardening]]（Key 级硬限停用）

## 需求

P3「成本效能与预算管控」第三件：部门/用户级预算规则 + 软限预警通知 + 硬限超限停用 + MCP 成本纳入 Key 预算。

## 用户决策

- **MCP 纳入 Key 预算**：`recomputeBudgetUsed` 改为 LlmLog+McpLog 双表 SUM（`budget_used = SUM(llm.external_cost) + SUM(mcp.external_cost)`）。
- **预算页面落点**：看板页现有预算 Tab 扩展为「Key 级/部门级/用户级」三个子 Tab（不另立菜单）。
- **部门预算范围**：直挂部门（部门 Key 消耗 + 该部门成员个人 Key 消耗，不含子部门）。

## 实现

### MCP 纳入 Key 预算（usage_aggregate.go）

`recomputeBudgetUsed` 改为双表 SUM：每个有预算 Key 先查 `LlmLog` 再查 `McpLog`，`budget_used = llmSum + mcpSum`。MCP 日志量小时直接扫 McpLog（暂不进聚合表）。

### 预算规则表（model/gateway/budget_rule.go）

- `gateway_budget_rule`：`(rule_id, scope_type=dept/user, scope_id, budget_limit, budget_hard_limit, budget_duration(1d/7d/30d), soft_warn_percent(默认80), is_active)`；scope_type+scope_id 唯一；OPS_AUDIT_MODEL 基座。
- `gateway_budget_alert`：`(alert_id, rule_id, period_key, alert_type=soft_warn/hard_limit)`；rule_id+period_key+alert_type 唯一（同周期同类型只告警一次）。

### 服务层（service/gateway/budget_rule.go）

- CRUD：`GetBudgetRuleList`（含读时聚合 budgetUsed + 预警状态）/`CreateBudgetRule`/`UpdateBudgetRule`/`DeleteBudgetRules`。
- `CheckBudgetAlerts`：遍历活跃规则 → `calcScopeCost` 读时聚合 scope 内总成本（LLM+MCP 双表，复用成本分析归因 JOIN 口径） → 超软限且本周期未告警 → 返回 `BudgetAlertResult`（API 层发通知，规避 service↔gateway import 环） → 超硬限 → 停用 scope 内活跃 Key + SysOperLog 审计。
- 通知目标：部门规则 → 部门 create_by，用户规则 → 该用户本人，超管兜底。
- 周期去重键：月 `YYYY-MM`、周 `YYYY-Wnn`、日 `YYYY-MM-DD`。
- 菜单：挂看板 api_prefix（`/gateway/budget`），casbin 零改动。

### API 层（api/v1/gateway/budget_rule.go）

- `GET /gateway/budget/list`、`POST/PUT/DELETE /gateway/budget`、`GET /gateway/budget/summary`（三维度汇总：Key 级复用现有 GetBudget + 部门/用户级规则列表）、`POST /gateway/budget/check`（手动触发预警检查 + 发通知）、`POST /gateway/budget/aggregate`（手动聚合 + 预算检查）。
- 定时任务：`CheckBudgetAlerts` 每 5 分钟（与 AggregateUsage 同频，聚合后检查）；`AggregateUsage` 已含 MCP 纳入预算。

### 前端

- 看板预算 Tab 升级为 NTabs（Key/部门/用户）：Key 级复用现有 BudgetItem 列表；部门/用户级用 BudgetRuleView 列表（含执行率进度条、预警状态标记、硬限标签）。
- `budget-rule-drawer`：新增/编辑预算规则（维度选择 + 部门树/用户选择器 + 预算上限 + 周期 + 预警阈值 + 硬限开关 + 启停）。
- 部门/用户 Tab 的 header 加「+新增规则」按钮。
- typings：`BudgetRuleView`/`BudgetRuleSearchParams`/`BudgetRuleOperateParams`。
- API：`fetchGetBudgetRuleList`/`fetchCreateBudgetRule`/`fetchUpdateBudgetRule`/`fetchDeleteBudgetRules`/`fetchGetBudgetSummary`。
- i18n：`page.gateway.budget.*`（tabKey/tabDept/tabUser/add/edit/scopeType/scopeDept/scopeUser/scopeName/budgetLimit/budgetUsed/duration/duration1d/7d/30d/softWarnPercent/hardLimit/alertStatus/normal/softWarned/hardLimited/isActive/form.*），三处同步。

## 待办

- 运行时验证：需创建部门/用户预算规则后触发聚合检查，核对软限预警通知+硬限停用闭环。
- 行操作按钮（编辑/删除）后续按需加到规则列表。
- 效能报告+导出（P3 剩余最后一件）。
