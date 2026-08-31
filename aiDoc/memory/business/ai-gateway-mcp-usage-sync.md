# AI 网关·P3 MCP 调用日志回流

- 日期：2026-08-31
- 状态：已实现（build/vet/test+typecheck/eslint 全过，待运行时验证）
- 关联：[[ai-gateway-mcp-server]]（MCP 管理与 LiteLLM 同步）、[[ai-gateway-usage-log-page]]（调用日志页）、[[ai-gateway-cost-analysis]]（成本分析消费）

## 需求

P3 从 P2 挪入的欠账：MCP 调用日志回流 + call_count + 成本口径。MCP 调用由客户端直连 LiteLLM 网关完成，平台不经手流量，日志靠定时任务从 LiteLLM_SpendLogs 回流。

## 用户决策

- MCP 日志前端放现有「调用日志」页加 LLM/MCP 双 Tab（不另立菜单，casbin 零改动）。
- 成本分析总览 KPI/趋势采用 **LLM+MCP 合并口径**（MCP 单独看走「按MCP」维 tab）。
- MCP 计费补内部单价双轨：MCPServer/MCPTool 新增 `InternalCostPerCall`（nil=同 external，对齐部署 internal_* 回落语义）。

## 关键事实

- **MCP 与 LLM 日志同表**：LiteLLM 把 MCP 调用也写 `LiteLLM_SpendLogs`，`mcp_namespaced_tool_name` 非空即 MCP 行——回流底座（spend DB 连接、keyset 分页、归因函数）全部复用。
- **修复口径污染 bug**：`fetchSpendBatch`/`ReconcileLLMLogs` 此前未排除 MCP 行——一旦有人调过 MCP，这些行会被当 LLM 日志混入 gateway_llm_log（deployment 归因失败、成本记 0、请求/token 计入 LLM 口径）。本次补 `"mcp_namespaced_tool_name" IS NULL OR = ''` 互斥分流（对账同样补）。
- `LiteLLMSpendLog` 只读映射补 `McpNamespacedToolName`/`Status` 两列。

## 实现

### 回流（service/gateway/mcp_usage_sync.go）

- `SyncMcpLogs`：独立游标（sync_state key=`mcp_logs`，loadSyncStateByKey 泛化）复用 keyset 分页；`toMcpLog` 归因（attributeAiKey/attributeUser 复用 + `attributeMcpTool`）→ `calcMcpCosts` 成本自算 → ON CONFLICT(request_id) 幂等落库 → call_count 按 server 增量 → last_used_at 回填。
- `ReconcileMcpLogs`：近 30 天两步查询漏单回灌（与 LLM 对账同模式）。
- 归因 `attributeMcpTool`：`namespaced_name` **整串精确匹配** `gateway_mcp_tool.namespaced_name`（唯一索引）→ 联动 server；工具未登记时按最长 serverName 前缀反查 server 兜底；**不 split 切名**（规避 serverName 含 `_` 时切错位的 AIHelms 坑）。
- 成本 `calcMcpCosts`：per_call 工具级优先（BillingType 非空 或 配了单价即视为覆盖）→ server 级 → free/无单价 0；internal 单价 nil 回落 external；不采信 LiteLLM spend 列。单测 8 场景。
- 定时种子：SyncMcpLogs `*/5 * * * *` + ReconcileMcpLogs `0 * * * *`（仅新库生效）。
- 规避 AIHelms mcp_tasks 全部坑：request_id 无约束 N+1 查重（我们唯一索引+幂等）、固定 10min 回看窗口停摆丢数（复合游标）、无对账（每小时）、status 硬编码 success（透传 LiteLLM status 列）、naive 时区（UTC 落库）。

### 查询消费

- `GET /gateway/usage/mcp/list|sync|reconcile`（挂 usage 菜单 api_prefix，casbin 零改动）：分页明细（用户/密钥/服务器/工具/状态/时间筛选 + userName/aiKeyName 回填）。
- 成本分析：`cost detail` 加 `dimension=mcp`（扫 gateway_mcp_log 实时聚合 server 两级第一级，行内展开工具子表 `GET /gateway/cost/detail/mcp-tools`）；**overview KPI/趋势合并 llm+mcp**（LLM 读聚合表 + MCP 扫日志表，业务日 Go 层合并；MCP 暂不进聚合表——调用量远小于 LLM，量大了再进）。MCP 部门归因同锚点口径（部门Key归部门/个人Key归主部门）；业务日区间经 Go 层转 UTC 时间下界过滤（走 started_at 索引，规避逐行 AT TIME ZONE cast）。
- MCP 维无 token：明细表 token 三列条件隐藏，tokens 排序键回落 requests。

### 前端

- 调用日志页 `_gateway/ai-audit/usage`：顶部 NTabs（LLM/MCP）+ 双面板 v-show 保状态；`mcp-log-panel.vue`（时间/用户/服务器下拉/工具/状态筛选 + 明细表 + MCP 版回流/对账按钮）；query 预填扩展 `tab=mcp`+`mcpServerId`（成本分析 MCP 维行「日志」跳转直达）。
- 成本分析 `cost-detail-table`：dimension 加 mcp（第七维），server 行行内展开工具子表（懒加载缓存）。
- MCP 管理页：列表加「调用次数」列（CallCount）；服务器抽屉/工具计费弹窗补内部单价输入（per_call 时展示，留空=同外部价）。
- i18n：`page.gateway.mcpLog.*` 新段 + usage.tabLlm/tabMcp + mcp.internalCostPerCall(+Tip) + mcp.col.callCount + cost.detail.dimension.mcp，三处同步。

## 待办

- 运行时验证：需真实 MCP 调用产生 SpendLogs 行（dev 注册 MCP server + 客户端经网关调用工具）后触发回流核对归因/成本/call_count。
- MCP 调用量级若显著增长：gateway_mcp_log 进聚合表（cost_summary_daily 加 resource_type 维）。
