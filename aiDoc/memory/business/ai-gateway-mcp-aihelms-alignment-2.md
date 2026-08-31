# MCP 管理二次对标 AIHelms 借鉴四项（同步态/创建拉工具/定时巡检/分类下拉）

- 日期：2026-08-30
- 状态：已实现（build/vet/test + typecheck/eslint 通过）

## 背景

全量对比 AIHelms(`/home/remote/AIHelms` apps/services/mcp_service.py + mcp_tasks.py + admin views/mcp/*) 与本项目 MCP 管理后确认：核心架构双方同构（LiteLLM 管理面超集，协议交互全委托 /mcp-rest/test/*）；本项目在凭证安全(AES-GCM+掩码)、授权模型(allow_all_keys 恒 false+双向对齐+自愈)、三档发布、管理页(远程分页/搜索/批量)上已强于 AIHelms。用户多选确认借鉴四项；**MCP 调用日志+成本回流+call_count 维持 P3 另立**（AIHelms 的 mcp_call_logs+SpendLogs 回流+日志页是大项，且其实现有去重无索引/status 恒 success 两坑，须重新设计）。

## 实现口径

1. **litellm_synced 同步态真实维护（修 bug）**：此前 CreateMCPServer `row.LitellmSynced = true` 只改内存不落库、Update/重推失败仅 Warn 无沉淀 → 前端「同步状态」列恒 false。现新增 `markMCPSyncState(ctx,db,id,err)` helper（成功 synced=true 清错误/失败记错误），五处接入：Create 事务内 Updates 三列一并落库；Update 推送成功后 mark；RefreshMCPTools/UpdateMCPToolBilling 重推 mcp_info 后 mark（保留失败仅 Warn 不回滚工具重建的语义）。Update 回填终态段同步置 true。前端列表未同步且有 litellmSyncError 时 NTooltip 展示错误详情（对标 AIHelms「未同步」徽标）。
2. **创建成功后自动拉工具**：CreateMCPServer 事务提交后串联 `RefreshMCPTools`（失败仅 Warn 不阻塞创建，对标 AIHelms create→refresh），响应 view 带 toolCount；前端新建成功按 toolCount 分支：>0 显示「工具列表已刷新，共 N 个工具」(复用 refreshSuccess key)、=0 warning 引导去工具面板重试(新 key addToolsMissed)。单机模式(cli==nil)恒走 warning 分支。
3. **定时健康巡检**：`HealthCheckAllMcps(ctx)`(全量 is_active 逐个复用 HealthCheckMCPServer 经 LiteLLM 探测落库，ctx.Err() 逐个检查防超时拖死；unhealthy 是探测结论非错误不进告警；单机模式直接跳过) + `task.Register("HealthCheckMcps")` + timed_task 种子 Spec `23 * * * *`。规避 AIHelms 定巡两坑：漏 streamable→http 映射误判（本项目探测统一走 litellmMCPTransport 归一）、page_size=500 上限（本项目无分页直查）。**仅新库生效，已有库须在定时任务面板手动创建**。
4. **MCP category 受控下拉**：后端 `GET /gateway/mcp/categories`（distinct 非空升序，与 Skill 同口径轻量受控不建表；gin 静态段与 :id 共存静态优先）；前端 operate-drawer NInput→NSelect(filterable+tag+clearable，打开时拉取，失败静默退化手输)、mcp-search 分类筛同样换 NSelect；casbin 由菜单 api_prefix `/gateway/mcp/*` 通配覆盖零改动。i18n 新增 form.categoryPlaceholder/addToolsMissed 三处同步。

## 明确不借鉴（本轮再确认）

- 凭证明文回显、OAuth 三件套/tags/business_scenario_id 僵尸字段、拉 200 条前端过滤——AIHelms 坑。
- basic auth_type、删除改 LiteLLM 软禁用(allow_all_keys=false 保 SpendLogs 历史)——待 P3 回流落地时一并考虑。
- 顺手修：mcp/index.vue handleHealthCheck `data` 遮蔽 lint 错误(改名 healthResult)；skill-operate-drawer initModel use-before-define(函数移到 publishSnap 声明后，存量未提交改动的 lint 挂)。

相关：[[ai-gateway-mcp-server]]、[[skill-aihelms-alignment]]、[[ai-gateway-p1-hardening]]
