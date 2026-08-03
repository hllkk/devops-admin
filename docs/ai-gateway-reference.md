# AI 网关模块开发参照

> 范围:devops-admin AI 网关模块(`/gateway/*`、`views/_gateway/`)的设计参照与功能输入。
> 状态:参照分析稿(2026-08-02)。基于对本地 `/home/remote/AIHelms`(企业 AI 资源纳管平台)前后端的实读 + `README.md`,结合记忆 `ai-gateway-reference-aihelms-first`、business 记忆 `ai-gateway-identity-home`。
> 定位参考:AIHelms 自述「企业 AI 资源纳管平台——管理 AI 资产、控制 AI 成本、衡量 AI 价值」。

---

## 1. 核心结论

**首选本地 `/home/remote/AIHelms`(同领域已落地完整实现)+ LiteLLM(它的模型代理底座,关键架构决策点)+ 阿里百炼(国内平台对照,项目已配 skills)**。当前项目已迈出第一步:`web/src/views/home/index.vue` 对标 AIHelms `MyIdentityView.vue` 复刻「我的 AI 身份」首页(全 mock)。

参照工作流(记忆 `ai-gateway-reference-aihelms-first`):**先列举 AIHelms 在目标功能的前后端实现 → 再针对 devops-admin 设计同页面同功能**;前端 Vue3 同构可高参照,后端 Python/FastAPI 只参照数据模型/接口契约/业务逻辑,不照搬代码。

---

## 2. 参照资源清单(按优先级)

| 优先级 | 参照源 | 位置 | 用途 | 注意 |
|---|---|---|---|---|
| **P0 首选** | AIHelms 前端 | `/home/remote/AIHelms/ui/packages/web/src/`(用户端)+ `ui/packages/admin/src/`(管理端) | 组件/交互/页面结构/接口契约**可直接借鉴**(Vue3 同构) | admin/web 双包有组件重复(如 ProviderIcon),抽取别照抄重复 |
| **P0 首选** | AIHelms 后端 | `/home/remote/AIHelms/apps/api/v1/`(路由)+ `apps/services/`(业务)+ `apps/models/db.py`(模型) | 数据模型/接口契约/业务逻辑设计参照 | Python/FastAPI,**只参照设计,不照搬代码**(当前 Go/gin) |
| **P0 关键决策** | LiteLLM | AIHelms `apps/services/litellm_client.py:1-100` 即对 LiteLLM 的封装 | **模型代理/路由/多供应商适配/对话/用量日志底座**——AIHelms 不自研这块 | 决定 devops-admin 是「Go + LiteLLM 服务」还是「自研 Go 代理」,见 §4 |
| P1 国内对照 | 阿里百炼 | 项目已配 skills:`bailian-docs-llm-wiki`/`bailian-model-recommend`/`bailian-train-deploy` | 国内 AI 平台功能对照 + 接入百炼模型参照 | 需要时用 Skill 工具调,不常驻 |
| P2 Go 代码参照 | One API / New API | GitHub(不在本地) | 仅当走「自研 Go 代理」路线时,作 Go 多模型网关代码参照 | 克隆需走代理(记忆 `github-clone-via-proxy-on-this-server`) |
| 现状 | 当前 home mock | `/home/devops-admin/web/src/views/home/index.vue` | 已对标 MyIdentityView 复刻 AI 身份首页(mock) | 后端 `/gateway/*` 未实现,待接真实接口 |

---

## 3. AIHelms 功能模块清单(设计输入)

> 路径前缀:AIHelms 前端 `ui/packages/{web,admin}/src/`,后端 `apps/api/v1/` + `apps/services/` + `apps/models/db.py`。

| 模块 | 前端 | 后端 | 设计要点 |
|---|---|---|---|
| AI 身份/个人中心 | `web/.../views/MyIdentityView.vue:1-390` | `api/v1/ai_keys.py:245-251`(`/my`)+ `services/ai_key_service.py` | **当前 home 已对标复刻(mock)**。AiKey 模型含 models/mcps/skills/agents 列表 + 预算(unified/per_type/per_resource 三模式) |
| 模型纳管/模型市场 | `web/.../views/ModelSquare.vue`+`admin/.../models/ModelManage.vue` | `api/v1/models.py:1-388`+`providers.py`+`credentials.py` | 供应商/凭证/模型注册分离;`ModelDeployment` 多部署负载均衡;token/per_call/monthly_quota 三计费;`RouterSettings` 路由策略 |
| MCP 纳管 | `web/.../views/MarketView.vue`+`admin/.../mcp/McpManage.vue` | `api/v1/mcp.py:1-363` | MCP Server 注册/工具发现/健康检查;`/connect-config` 下发 URL+认证头;工具级计费 `McpTool` |
| Skill 管理 | 同 MarketView+`admin/.../skills/SkillList.vue` | `api/v1/skills.py:1-303` | zip 包上传含 `agent_install_prompt`;AI Policies 安全审查;`/install-info` 下发 prompt+下载链接 |
| API Key/密钥 | `admin/.../ai-keys/AiKeyManage.vue`+`api-keys/ApiKeyManage.vue` | `api/v1/ai_keys.py:1-465`+`api_keys.py:1-105` | **AI Key(用户 AI 身份,绑资源+预算,经 LiteLLM 认证)与 API Key(管理员调管理接口,`key_hash` 加密)分离** |
| AI 对话 | 无独立界面(第三方客户端接入) | `services/litellm_client.py:55-87`(`chat_completion`) | **不自研对话**,靠 LiteLLM 代理;SSE 流式;日志落 `LlmCallLog` |
| Token 计费/预算 | `MyIdentityView.vue:332-370`+`admin/.../efficiency/CostView/BudgetView` | `api/v1/efficiency.py:205-342`+`usage_logs.py` | **内外双轨定价**(`internal_cost`/`external_cost` 分离);软限制预警+硬限制停用;部门/项目级配额 |
| 推理引擎 | 无直接界面 | `services/litellm_client.py:1-100` | **不直接纳管 vLLM/SGLang**,经 LiteLLM;模型部署配 `litellm_params`;连通性测试 `access_test.py:49-138` |
| 申请/审批 | `MyIdentityView.vue:372-387`+`admin/.../resource-approval/` | `api/v1/resource_applications.py:1-180` | model/mcp/skill/agent 四类申请;批量审批;通过后**自动加到用户 AI Key** |
| 多模型路由 | `admin/.../models/ModelManage.vue` | `api/v1/models.py:350-378`+`models/db.py:302-315` | `routing_strategy`(simple-shuffle)/`fallbacks`/`num_retries`/`timeout`/`cooldown_time` |
| 审计/日志 | `admin/.../logs/LogsManage.vue`+`audit/AuditLogManage.vue` | `api/v1/usage_logs.py`+`audit_logs.py` | 四类用量日志:`LlmCallLog`/`McpCallLog`/`SkillUsageLog`/`AgentUsageLog`+管理审计 `AdminAuditLog` |
| RBAC | `admin/.../users/UserList.vue`+`roles/RoleList.vue` | `api/v1/users.py`+`roles.py`+`departments.py`+`projects.py` | 用户多部门多项目;`require_permission` 中间件;数据隔离基于部门/项目 |
| 智能体中心 | `web/.../views/AgentCenter.vue`+`admin/.../agents/AgentList.vue` | `api/v1/agents.py:1-289` | 多平台智能体(Dify/Claude);成本归属(owner/user);`chat_url` 跳转对话 |
| AI Policies 安全 | `admin/.../ai-policies/AiPoliciesView.vue`+`AuditReportView.vue` | `api/v1/ai_policies.py` | Skill 安全扫描(恶意指令/数据外传/权限过大);LLM 辅助审查;实验性 |
| 效能分析/报告 | `admin/.../efficiency/`(Overview/Adoption/Cost/Budget/Health/Reports) | `api/v1/efficiency.py:1-449`+`dashboard.py` | 概览/落地率/成本/预算/健康/报告六子模块;周月报告 LLM 生成;多维度下钻 |
| 业务场景 | `admin/.../business-scenarios/BusinessScenarioManage.vue` | `api/v1/business_scenarios.py` | 场景关联模型/MCP/Skill;`KeyScenario` 快速建场景 Key |
| 品牌许可证 | `admin/.../system/BrandingView.vue`+`LicenseView.vue` | `api/v1/branding.py`+`license.py` | 平台名/Logo/Favicon;企业版许可证控功能开关 |
| 导出任务 | `admin/.../audit/ExportTasksManage.vue` | `api/v1/export_tasks.py` | Celery 异步导出;状态跟踪/取消/重试 |

---

## 4. 关键架构决策:LiteLLM 底座(进设计前必须先定)

AIHelms 的**模型代理/路由/多供应商适配/对话/用量日志采集实际全由 LiteLLM 承担**(`apps/services/litellm_client.py`),AIHelms 自身只专注企业管理层(身份/预算/审批/效能/审计)。devops-admin 有两条路:

- **路线 A(建议,守 AGENT.MD 不过度设计)**:Go 后端 + LiteLLM(Python 独立服务)做模型代理。Go 专注企业管理层,调 LiteLLM 代理接口。**复用 LiteLLM 成熟的路由/计费/日志,最省力,AIHelms 已验证**。
- **路线 B(自研)**:参照 LiteLLM 的 `RouterSettings` 设计(routing_strategy/fallbacks/num_retries/timeout/cooldown_time),用 Go 实现模型代理。单语言栈,但工作量大;此时可参照开源 Go 网关 One API / New API。

此决策影响后端整体架构,进任何模块设计前先定。

---

## 5. 语言差异策略(记忆 `ai-gateway-reference-aihelms-first`)

| 层 | AIHelms | devops-admin | 参照方式 |
|---|---|---|---|
| 前端 | Vue3.4 + TS + Tailwind | Vue3 + TS + NaiveUI + UnoCSS + Elegant Router | **高参照**:组件/交互/接口契约可直接借鉴(home 已成功复刻 MyIdentityView) |
| 后端 | Python3.11 + FastAPI + Celery | Go + Gin + GORM | **只参照设计**:数据模型/接口契约/业务逻辑;不照搬代码 |
| 存储 | PostgreSQL16 + Redis7 | MySQL + Redis(`OPS_DB`/`OPS_REDIS`) | 数据模型设计可参照,表结构按本项目 GORM 约定重写 |
| 异步 | Celery | 本项目有定时任务 + notice-sse hub | 导出/安全扫描/报告生成参照思路,用本项目定时任务 |

---

## 6. 最值得借鉴的设计点

1. **AiKey 作核心身份载体**:用户身份+资源授权(models/mcps/skills/agents)+预算+速率限制统一封 `AiKey`,扩展性强;三预算模式(unified/per_type/per_resource)灵活。
2. **内外双轨定价**:`internal_cost`/`external_cost` 分离,满足企业内部结算+外部成本核算。
3. **AI Key 与 API Key 分离**:用户身份 Key vs 管理员接口 Key,用途/安全等级不同,别混。
4. **申请审批闭环**:用户申请→管理端批量审批→通过自动授权到 AI Key,闭环完整。
5. **效能分析六子模块下钻**:概览→落地率→成本→预算→健康→报告,从宏观到微观。
6. **AI 身份证视觉**:信息密度高(当前 home 已复刻)。

---

## 7. 必避的 AIHelms 短板

| # | AIHelms 短板 | devops-admin 对策 |
|---|---|---|
| 1 | 对话界面缺失(靠第三方客户端) | 可补简易对话窗(非技术用户友好) |
| 2 | MCP 健康检查弱(手动触发) | 加定时自动检查 + 告警(复用本项目定时任务 + notice-sse) |
| 3 | 推理引擎纳管浅(仅 LiteLLM 代理) | 需自建推理服务的企业补 GPU 监控/模型加载 |
| 4 | 日志查询性能(直接查表) | 量大时引入 TimescaleDB/ClickHouse,或本项目分表 |
| 5 | 预算实时性不足(靠 `CostSummaryDaily` 聚合) | 加实时预算检查中间件(硬限制拦截) |
| 6 | 无多租户隔离 | 多企业部署需重设数据隔离 |
| 7 | 路由缺可视化配置 | 补路由拓扑图/配置向导 |

---

## 8. 与 devops-admin 现状衔接

| 已落地 | 待做 |
|---|---|
| `web/src/views/home/index.vue` AI 身份首页(身份证/资源/用量KPI/趋势/申请,全 mock,对标 MyIdentityView) | 后端 `/gateway/identity/*`(个人 AI Key/授权/用量/趋势/申请)→ 替换 home mock |
| `page.home.identity.*` i18n 三处同步(zh-cn/en-us/app.d.ts schema) | 建 `web/src/service/api/gateway/identity.ts` + `typings/api/gateway.api.d.ts`,home 切真实接口 |
| `useEcharts`+`SvgIcon`+`useClipboard` 复用(无新依赖) | `views/_gateway/` 管理页:模型纳管/MCP/Skill/AI Key/效能/审计/智能体 |
| 架构决策未定 | 先定 LiteLLM 底座路线(§4) |

---

## 9. 后续模块设计输入(按依赖排序)

1. **架构决策**:LiteLLM 底座路线 A/B(先定,影响全局)
2. **AI 身份后端**:`/gateway/identity/*`——当前 home mock 待替换,依赖最小,可最先做
3. **模型纳管 + LiteLLM 接入**:`/gateway/models/*` + 供应商/凭证——AI 网关核心,依赖架构决策
4. **API Key/AI Key 管理**:`/gateway/ai-keys/*`——绑资源+预算,依赖模型纳管
5. **申请审批**:`/gateway/resource-applications/*`——依赖资源(模型/MCP/Skill)就绪
6. **MCP/Skill 纳管**:`/gateway/mcp/*`+`/gateway/skills/*`
7. **用量计费/预算/效能**:`/gateway/usage-logs/*`+`/gateway/efficiency/*`——依赖调用链打通
8. **审计/智能体/AI Policies**:后置

每模块进设计前,按 AGENT.MD 工作流用 `AskUserQuestion` 确认是否启用 superpowers 重流程。
