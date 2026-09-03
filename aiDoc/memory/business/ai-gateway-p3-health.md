# AI 网关·P3 健康检查

- 日期：2026-09-02
- 状态：已实现（go build/vet/test + typecheck 通过，待运行时验证）
- 关联：[[ai-gateway-mcp-aihelms-alignment-2]]（MCP 巡检已有部分）、[[ai-gateway-p3-adoption]]

## 需求

P3 收尾三件之二：健康检查——MCP 上游/模型部署/Docker 环境状态。用户三项拍板：模型部署=真实探测落库；环境维度=基础组件健康（不做 Docker）；报告=模板化。

## 对 AIHelms 的三处刻意偏离（避其坑）

1. **补「基础组件」卡**（LiteLLM `cli.Ping` /health/readiness + PG `SELECT 1` + Redis Ping，未配置=unknown）——AIHelms 探测强依赖 LiteLLM 但最关键依赖自身无监控
2. **模型部署健康=真实探测落库**——AIHelms 纯配置判断（有启用部署即"健康"）命名误导
3. **不做 Docker 自检**（AIHelms 在请求进程跑 subprocess）——server 容器内摸不到 docker.sock；「数据回流新鲜度」替代环境语义

## 落地

- **ModelDeployment 加健康三列**（health_status/last_health_check/health_check_error，对齐 MCP 模式）+ 常量 DeploymentHealthUnknown/Healthy/Unhealthy；AutoMigrate 自动加列
- **探测粒度=模型路由组**：`HealthCheckAllDeployments` 按启用部署 join 启用模型 DISTINCT model_key 分组，复用 `buildModelProbe`（从 TestDeployment 抽出的共享请求构造：category 分流端点+ASR 静音 wav 探测体）经 LiteLLM 数据面 ping，**结论写组内全部启用部署**（网关级 ping 无法定位组内单节点，单点故障由 LiteLLM allowed_fails/cooldown 兜底，注释明示）；单机模式 cli==nil 跳过；ctx.Err() 逐组检查
- **GetHealthSummary**：四卡（MCP=现有落库计数 is_active 分母/部署=新落库/组件=即时探测/新鲜度）+三明细（MCP 行/部署行 join 模型名/组件行含 latency+message）
- **数据回流新鲜度**：读 `gateway_sync_state` llm_logs/mcp_logs 两游标 `updated_at`（每 5 分钟 tick 无论有无新数据都推进——静默时段无日志不误报，区别于日志行时间）；≤10min healthy/≤60min warning/更久 danger/无记录 unknown
- task.Register("HealthCheckDeployments") + 种子 `43 * * * *`（与 MCP 巡检 23 分错峰；仅新库生效）
- 端点：GET /gateway/health/summary + POST /gateway/health/check-deployments（手动巡检返回组数）；菜单 route.ai-audit_health OrderNum 5 user 不授
- 前端 `_gateway/ai-audit/health/index.vue`：四状态卡（healthy/total 大数字+三态分解）+「立即巡检」按钮+NTabs 三明细（组件 Tab 内嵌新鲜度三格）；i18n page.gateway.health.* 三处同步
