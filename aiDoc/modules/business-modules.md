# 业务模块 (business-modules)

> devops-admin 的业务按模块组织，每个模块一节，记录 model/接口/边界。随业务开发补充。
>
> **实现状态总览（2026-07-16 校准）**
> - **当前已落地主线：system 核心基座（`1d632d9` 重构后）**——用户/角色/部门/岗位模型 + 安全配置/数据权限/JWT 黑名单/错误日志/定时任务 Service 骨架（详见下方「系统模块（重构后现状）」节）。**注意**：重构前的菜单/字典模型、初始化向导、`/init/*`、`/auth/*`（httpOnly cookie + go-captcha）、`sys_setting` 等实现**在重构后未保留**，对应 API/Router 待重建（历史轨迹见 `aiDoc/memory/business/` 与 `demand-index.md`）；`api/` 层当前为空。

## 系统模块（重构后现状）

`1d632d9`「完成项目基础架构重构与模块初始化」起，系统模块以**核心基座**形式重建，命名统一 `OPS_` 前缀。当前为 **Model + 部分 Service 骨架**，业务 API/Router 尚未补齐。

### 已建模型（`model/system/`）

- 业务实体：`SysUser`（`OPS_AUDIT_MODEL` + 自定义 `UserId`，字段对齐前端 `Api.System.User`/RuoYi）、`SysRole`（旧形态：手写时间戳、主键 `RoleId` json `id`）、`SysDepartment`、`SysPosition`。
- 系统表：`JwtBlacklist`、`SysSecurityConfig`（安全配置，替代旧 `sys_setting`）、`SysError`（错误日志）、`SysDataAccessLog`（数据访问审计）、`SysTimedTask`/`SysTimedTaskLog`（定时任务）、`SysLoginLog`（登录日志）、`SysOperLog`（操作日志）。
- 关联表（显式 struct）：`SysUserRole`、`SysRoleDepartment`、`SysUserDepartment`。
- **未保留**：`SysMenu`/`SysDictType`/`SysDictData`/`SysRoleMenu`（重构前曾建，重构后待重建）。

### 已建 Service 骨架（`service/system/`）

- `data_scope`：数据权限引擎（按 `dept_id`/`create_by` 构建身份与可见范围，实现见 `utils/datascope`）。
- `jwt_black_list`：JWT 黑名单（refresh 轮换失效）。
- `sys_security_config`：安全配置（密码策略/登录失败锁定/IP 黑白名单，存 `SysSecurityConfig` 表）。
- `sys_error`：错误日志。
- `sys_timed_task`（+ `http` + `runner`）：定时任务调度与 HTTP 触发。
- `auto_code`：代码生成。

### 待补

- 业务 API/Router：`api/` 当前**为空**，user/role/dept/post/menu/dict 的 API+Router 待按 `Api.System.*` 反推实现（前端页面已齐备）。
- 菜单/字典：模型与权限 seed 待重建（前端 `page.system.menu/dict` 已就绪）。
- 认证与初始化链路：`/auth/*`（httpOnly cookie 登录 + go-captcha 验证码）、`/init/*`（初始化向导）等 HTTP 接口重构后未保留，待重建（历史设计见 `aiDoc/memory/business/` 的 `httponly-cookie-auth`、`go-captcha-login`、`system-init-flow`、`init-wizard-redis`）。

### 系统设置（待重建）

重构前的全局配置中心 `sys_setting`（`name`=分类 + `value`=JSON，五类 `general`/`security`/`authentication`/`ldap`/`notify`）**在重构后未保留**。当前安全相关配置由 `SysSecurityConfig` 承载，其余分类（通用/认证/LDAP/通知）的落地方式待重新设计，历史设计要点见 `demand-index.md`。

---

## AI 网关模块

> 2026-08-06 立项。企业 AI 资源纳管（管资产 / 控成本 / 量价值），参照 `/home/remote/AIHelms`（FastAPI+LiteLLM+PG+Redis 蓝本，devops-admin 后端 Go 只能参照设计）。立项决策与现状见 `memory/business/ai-gateway-overview.md`。

### 底座选型（已确认）

引入 **LiteLLM 官方镜像**作模型代理底座，**不自研转发**。多 provider 兼容、OpenAI/Anthropic 双格式、token 计费、流式转发、路由负载均衡、虚拟 Key 鉴权全交给 LiteLLM；devops-admin 后端只做"管理面"与用量统计。这是整个模块的基座决策，复用成熟开源、避免重造转发引擎。

### 管理面 vs 转发面分离

| 面 | 承载 | 职责 |
|---|---|---|
| 转发面 | LiteLLM 容器（端口 4000） | 客户端直连 `litellm_public_url`，LiteLLM 鉴权虚拟 Key + 路由部署 + 记录 spend |
| 管理面 | devops-admin Go 后端 | Provider/Credential/Model/AiKey 的 CRUD + 同步到 LiteLLM 管理 API + 定时拉用量日志 + 成本换算 |

devops-admin 侧表存**管理元数据 + LiteLLM 引用 ID**（`litellm_key_id`/`litellm_model_id`/`litellm_synced`），不存原始转发逻辑。

### 四层数据模型

`Provider → Credential → Model + ModelDeployment → AiKey`

- **Provider**（供应商）：`name`/`provider_type`(openai/anthropic/vllm…)/`is_active`。纯接入元数据，**不承载计费/预算**（2026-08-25 对齐 AIHelms：计费类型与预算口径统一在部署级，见 ModelDeployment；历史版本的 `billing_type`/`monthly_budget`/`monthly_used` 已移除，老库残留列无害）
- **Credential**（凭证）：`credential_name`（全局唯一，对应 LiteLLM credential_name）/`credential_values`（JSONB：api_key、api_base）/`provider_id`/`litellm_synced`/`is_active`
- **Model**（模型）：`model_id`（全局唯一，对应 LiteLLM model_name）/`name`/`category`/`capabilities`/`is_published`/`visibility_type`(all/selected)/`requires_approval`
- **ModelDeployment**（部署）：`model_id` + `credential_id` + `litellm_params`（LiteLLM 路由参数 JSONB：model、api_base…）+ `model_info`（内外定价 JSONB）+ `billing_type`(token/per_call/monthly_quota)+`cost_per_call`/`monthly_call_quota`+ `is_active`。一个 Model 多 Deployment → LiteLLM 同 `model_name` 多部署形成负载均衡池；**计费类型与按次/配额成本全在部署高级设置维护**（usage 回流按部署级 `billing_type` 计算成本）
- **AiKey**（AI 身份）：`key_type`（主/场景 × 个人/部门/项目）/`owner_type`+`owner_id`（复用 SysUser/SysDepartment）/`litellm_key_id`+`litellm_key_alias`（格式 `{owner_type}:{owner_id}/{name}`）/`models`/`mcps`/`skills` 授权 JSONB/`budget_limit`+`budget_hard_limit`/`rate_limit_mode`+`tpm_limit`+`rpm_limit`

### LiteLLM 集成边界

Go HTTP 客户端封装（用 `LITELLM_MASTER_KEY` 鉴权）调 LiteLLM 管理 API：

- AiKey 同步：`/key/generate`、`/key/delete`、`/key/update`
- ModelDeployment 同步：`/model/new`、`/model/delete`、`/model/{id}/update`
- Credential 同步：`/credentials`、`/credentials/{name}`

同步策略：创建/更新立即同步到 LiteLLM，删除级联删 LiteLLM 侧；禁用凭证级联禁用关联部署（AIHelms 在 LiteLLM 侧 model_name 加 `__disabled__` 后缀实现）。

### 用量与成本

- LiteLLM `store_model_in_db: true` + `store_prompts_in_spend_logs: true`，模型配置与 spend 日志落 PG
- `SysTimedTask` 定时（默认 5 分钟）从 LiteLLM 增量拉日志，关联 `ai_key_id`/`deployment_id`（替代 AIHelms 的 Celery）
- **内外双轨定价**：`external_cost`（USD，LiteLLM 记录）+ `internal_cost`（¥，平台定价 ¥/百万 token），`usd_to_cny_rate` 换算
- 定价单位转换：平台 ¥/百万token ↔ LiteLLM USD/token（同步到 LiteLLM 时除以汇率与百万）
- 订阅/积分制厂商（智谱积分/百炼 Credits/套餐）的计量适配与套餐真实余量旁路见 [`ai-gateway-billing-integration.md`](ai-gateway-billing-integration.md)

### 部署

`deploy/docker-prod/docker-compose.yml` 加 `litellm` 服务（`ghcr.io/berriai/litellm`），**共用现有 PostgreSQL**（devops-admin 已是 PG+Redis，与 AIHelms 同构），挂 `litellm.yaml`（`store_model_in_db`/`store_prompts_in_spend_logs`）；`LITELLM_MASTER_KEY`/`LITELLM_SALT_KEY` 进 `.env`。对外接入点经 nginx 反代或直暴露 4000。

### 基座复用

- AiKey 归属 `owner_id` 直挂 `SysUser`/`SysDepartment`，认证走 httpOnly cookie（不另建用户体系）
- 分层遵循 `Router → API → Service → Model`，业务表用 `OPS_AUDIT_MODEL` 基座 + 雪花主键 + GORM
- 统一响应 `{code,data,msg}`、分页 `request.PageInfo`、Swagger `PageResult`
- 前端复用 `@sa/axios`/Elegant Router/Soybean 体系，页面落 `views/_gateway/`，接口封装落 `service/api/gateway/`

### 分期规划（4 期；P1–P3 主线，P4 可选）

对照 AIHelms 全功能地图，devops-admin 已有 system 基座（用户/角色/部门/认证/数据权限/操作·登录日志），故 AIHelms 的 `users`/`departments`/`projects`/`roles`/`auth`/`audit_logs`/`license`/`branding` 不重做，只聚焦 **AI 特有**功能。

**P1 · 核心网关五件套（已确认，进行中）**——打通"供应商→凭证→模型→AI 密钥→用量"主链路：

1. 供应商管理 Provider（CRUD + 接入格式；计费/预算不在此层，已收敛到部署级）
2. 凭证管理 Credential（CRUD + credential_values 加密 + 同步 LiteLLM `/credentials`）
3. 模型管理 Model+ModelDeployment（CRUD + 部署路由参数 + 内外定价 + 同步 LiteLLM `/model/*` + 连通性测试）
4. AI 密钥管理 AiKey（主Key自动建/场景Key申请 + owner 挂用户·部门 + 模型授权 + 预算 + 限流 + 同步 LiteLLM `/key/*`）
5. 用量统计 + 看板（`SysTimedTask` 定时从 LiteLLM 拉 spend + 成本换算 + Dashboard）

附：LiteLLM 容器接入（deploy + config）。前端 `_gateway` 下 5 个管理页 + home mock 切真实 `/gateway/identity/*`。产出：管理员接入模型/签发 Key；员工在 home 看 AI 身份+用量；客户端拿 Key 直连 LiteLLM 用模型。

**P2 · AI 市场**——企业内 AI 工具统一注册/审批/分发：

- MCP Server 管理（注册/工具列表/健康检查 + MCP 调用日志）
- Skill 管理（发布上架/版本/作者/下载地址 + Skill 使用日志）
- 资源申请审批（员工申请 Skill/MCP 权限 + 批量审批流）
- AiKey 扩展 `mcps`/`skills` 授权（P1 已预留字段，P2 补对应实体）

产出：员工在 AI 市场浏览/申请 MCP/Skill，管理员审批；Key 的 MCP/Skill 授权生效。

**P3 · 成本效能与预算管控**——让 AI 投入产出"看得见、控得住"（AIHelms 核心价值主张）：

- 成本分析（内外双轨明细：按人/部门/项目/模型/Key 下钻 + Token 列 + 人员 Top10）
- 预算管控（多维预算：人/部门/项目/模型；软限预警 + 硬限超支停用，硬限对接 LiteLLM `max_budget`）
- 覆盖率/采用度（部门覆盖率/人员激活率/模型分布/日均调用量/人均 token）
- 健康检查（MCP 上游/模型部署/Docker 环境状态）
- 效能报告（周/月自动生成 + 多维下钻 + 导出任务）

产出：决策层看成本报表、设预算、看效能报告；超支自动停用。

**P4 · 安全审计与智能体（可选扩展）**：

- AI 策略（SkillSpector 扫描 Skill 识别恶意指令/数据外传/权限过大 + LLM 审查 + 高风险 prompt 拦截）
- 智能体中心（创建/配置/生命周期/使用统计）
- 业务场景（场景模板 + 资源配置包，批量给 Key 配资源）

产出：Skill 上架前安全审查；智能体纳管；按业务场景批量配 Key。

### 现状（2026-08-06）

- 前端：`views/home/index.vue` AI 身份首页 mock 已落地（参照 AIHelms `MyIdentityView`，见 `memory/business/ai-gateway-identity-home.md`）；`views/_gateway/gateway` 占位页；路由 `/gateway`+`/home` 预留；`service/api` 下无 gateway 目录、无业务契约 typings
- 后端：`/gateway/*` 全缺；`config/ai.go` + `service/system/sys_ai.go` + `auto_code_llm.go` 是错误日志 AI 分析（非网关）
- 待办：后端按四层模型落地 P1 + LiteLLM 容器接入 + home mock 切真实接口 + 建管理页 `views/_gateway/`
- 进展：slice1 已落地——`config/litellm.go` + `utils/litellm/client.go` + Provider 四层（router/api/service/model，2026-08-22 核对）
- 进展：slice2 已落地（2026-08-22）——Credential 四层 + `credential_payload.go` 纯函数层（投影/掩码/合并，7 单测）+ `gateway_provider_prefix` 表 42 行种子（`source/gateway` 包，/initdb 与重启双路幂等）+ `litellm.credential-key` 配置（credential_values AES-256-GCM 密文落库、出网仅敏感 key 掩码）+ resync 兜底端点 + `GetProviderFields` 动态表单透传 + Provider 删除保护回补。已规避 AIHelms 两坑（明文落库/明文回显）。全链路 dev 验证通过（同步/懒同步/删除顺序/resync/单机模式）
- 进展：slice3 已落地（2026-08-23）——`gateway_model`/`gateway_model_deployment`/`gateway_model_visibility` 三表 + `deployment_payload.go` 纯函数层（三态路由名/凭证引用/前缀化/v1/双轨定价换算/镜像/掩码还原/脱敏，8 单测）+ DeploymentService（CRUD+连通测试+共享同步管线 buildDeploymentParams/pushDeployment）+ ModelService（CRUD/发布/软删三连/改名改类级联）+ 凭证级联回补（启停/换绑→部署路由 `__disabled__` 摘池，删除被引用拒删）+ `usd-to-cny-rate` 配置。LiteLLM 1.98 实测注意：PATCH 后转发路由即时生效，但 `GET /model/info` 管理接口有显示缓存（重启才刷新），验证时勿被管理接口读数误导
- 进展：slice4 已落地（2026-08-23）——`gateway_ai_key` 表 + AiKeyService（GetMyIdentity 惰性建主Key/CreateSceneKey/UpdateAiKey/DeleteAiKey/GetAvailableModels）+ 包级同步管线（expandModelsWithAnthropic 变体扩展/buildKeyAlias/syncKeyToLitellm/ensureMainKeyExists/syncPublicModelToMainKeys）+ PublishModel 回补发布公开模型自动授权主Key。偏离 AIHelms：key_value AES-GCM 加密存储(home 需明文+LiteLLM只返回一次)、主Key 惰性建(identity/my 触发)、场景Key 默认可用(P1 无审批)、4 种 key_type(无 project，devops-admin 无此实体)、per-model 限流用 JSONB map(不建独立表)。LiteLLM 1.98 实测：`/key/generate` 用 `token_id`(非 `key_id`)作密钥标识，`/key/info` 同样有显示缓存(直接查 LiteLLM DB `LiteLLM_VerificationToken` 表见真值)。P1 五件套后端已齐(用量看板见 slice5)
- 进展：slice5a 已落地（2026-08-23）——`gateway_llm_log`(原始用量日志,request_id唯一约束+ON CONFLICT幂等) + `gateway_sync_state`(复合游标KV) + LiteLLMSpendLog 只读映射(不AutoMigrate) + UsageSyncService(SyncLLMLogs复合游标keyset分页+入库时归因AiKey/User/Deployment+成本重算+对账ReconcileLLMLogs两步查询避跨库) + 第二DB连接(OPS_SPEND_DB,litellm.spend-dsn,dev独立litellm库/prod共享devops_admin库留空复用主库) + 定时任务种子(SyncLLMLogs每5分钟/ReconcileLLMLogs每小时)。成本P1简化:external/internal都用deployment.model_info四键算(internal P1同external,P3扩展internal_*区分对客/对内)。LiteLLM 1.98 实测:`/spend/logs` HTTP API只返回日级聚合(无法拉原始行做游标分页)→必须直接查`LiteLLM_SpendLogs`表;master key行在查询阶段按api_key过滤跳过;GORM Create+OnConflict DoNothing的RowsAffected计数不准(幂等场景=0正确,首次插入偏保守)。slice5b(聚合缓存+预算超限停用闭环+看板)依赖本slice原始日志表
- 进展：slice5b 已落地（2026-08-23，P1 收尾完成）——`gateway_cost_summary_daily`(日汇总缓存,uint自增主键,Raw INSERT SELECT不走雪花) + UsageAggregateService(AggregateUsage:滚动重建近60天DELETE+INSERT自愈→recomputeBudgetUsed按budget_duration窗口SUM external_cost覆盖→enforceBudgetHardLimit超限复用syncKeyToLitellm停用max_budget=0+SysOperLog审计) + DashboardService(overview/trend/top/budget,scope=self看自己/all看全部,非超管强制self) + AggregateUsage定时种子每5分钟。补AIHelms的坑:超限自动停用闭环(发现超限→停用Key→max_budget=0→审计日志)。时区:日桶`date_trunc('day',started_at AT TIME ZONE 'Asia/Shanghai')`按Shanghai切(规避UTC错位8h)。踩坑:聚合表DELETE必须Unscoped物理删(OPS_BASE软删会占行累加);top的Where用字符串拼接非GORM `%s`占位符;审计直接Create SysOperLog避免import service/system成环。**P1五件套全部完成(后端)**
- 变更：主 Key 改管理员创建制（2026-08-25）——用户决策：取消 identity/my 惰性建主 Key，所有用户必须超级管理员/管理员在后台(密钥管理 CreateSceneKey)创建后才能拿到 Key。落地：GetMyIdentity 查无主 Key 返回未开通态(MyIdentityView 新增 `opened=false`，KeyValue/授权空，AvailableModels 照常)；`ensureMainKeyExists`→`loadMainKey`(只读+已有主 Key 幂等自愈补公开模型，不创建)；CreateSceneKey 建主 Key(personal_main/dept_main)未显式指定 models 时默认授权全部公开模型(承接原惰性建默认语义，管理员建完即可用)。前端 home：loadIdentity 按 `opened` 判断，未开通 mainKey=null 走"暂无 AI 身份，请联系管理员开通"空态(noIdentity 文案 zh/en 已改)。slice4 记录中的"主Key 惰性建(identity/my 触发)"描述自此作废
- 进展：前端 home 切真实已落地（2026-08-23）——home/index.vue identity tab 从 mock 切真实接口：`fetchGetMyIdentity`(GET /gateway/ai-key/identity/my 惰性建主Key+明文)+`fetchGetDashboardOverview`(scope=self)+`fetchGetDashboardTrend`(scope=self, ECharts趋势date/cost)；字段适配后端 MyIdentityView(删 budgetScope/budgetModelsTotal/budgetMcpsTotal/mcps/skills/applications 的 mock 字段,改用 budgetLimit/budgetHardLimit/models)；mcps/skills/申请区占位"敬请期待"(P2资源申请)。新增 service/api/gateway/identity.ts+dashboard.ts+index.ts 导出；typings/gateway.api.d.ts 加 MyIdentity/AvailableModel/DashboardOverview/TrendItem/TopItem/BudgetItem/AiKey 类型。顺手修 _gateway/gateway 占位页 flatRequest 的 res.rows 预存 bug(res.data?.rows)。typecheck 通过+vite dev 启动成功。管理页(供应商/凭证/模型/密钥/看板)后续逐 slice 做(多路由页+菜单对齐 system)
- 变更：前端菜单重组（2026-08-25）——用户反馈原"AI 身份/AI 模型"双菜单撕裂(配供应商在 AI 模型、配其凭证却跳 AI 身份)。重组为三顶层：AI 看板(/gateway 单页) + 模型供给(/models 目录,原"AI 模型"改名,含供应商/凭证/模型) + 密钥管理(/ai-key 顶层单页,原"AI 身份"降级只装 AiKey)。Credential 从 identity 迁入 models;identity 目录删除。API 前缀/casbin 策略不变;已有库需手动同步 sys_menu。注:凭证已定不独立成页,内聚进供应商管理(落地:供应商页改为 TableSiderLayout 左右分栏,左侧供应商列表含凭证数,右侧选中供应商的凭证面板,参照 AIHelms ProviderManage);独立凭证菜单/目录已删,后端 credential 接口保留(凭证面板+deployment 表单复用);Provider 加 supported_formats 字段(凭证 format 从中选一);凭证增改用 Dialog 固定字段(名称/接入格式/API Base/API Key,掩码回传保留旧值);deployment 表单补"供应商→联动凭证"选择链。2026-08-25 落地,typecheck+go build 通过
- 变更：接入格式扩至四种（2026-08-25）——新增 LM Studio / Ollama 两种本地推理接入格式(openai/anthropic/lmstudio/ollama)。对齐 AIHelms accessFormats 语义但规避其坑：AIHelms 凭证选 lmstudio 格式时 provider_prefix_map 查表 miss(表里只有 (lmstudio,openai,chat) 行)，靠 credential_info.custom_llm_provider='openai' 兜底且 needs_v1 不生效(用户须手填带 /v1 的 base 否则 404)；本项目直接补种子行 (lmstudio,lmstudio,chat/embedding)→prefix=openai+needs_v1=true 走统一表驱动管线，不引入 custom_llm_provider 映射。Ollama：种子 (ollama,ollama,chat/embedding)→prefix=ollama+needs_v1=false(自有 /api 路由,默认端口 11434,api_base 填 http://host:11434)；无鉴权格式(ollama)前端隐藏 API Key 输入且提交不带 api_key(常量 FORMAT_NEEDS_NO_KEY)；LM Studio 默认端口 1234 走 OpenAI 兼容端点 /v1。落地：model 常量 CredentialFormatLmstudio/Ollama、种子 2 行(OnConflict DoNothing 幂等,重启自愈补齐)、前端 CREDENTIAL_FORMAT_OPTIONS+FORMAT_API_BASE_PLACEHOLDER+FORMAT_NEEDS_NO_KEY、i18n 三处同步(formatLmstudio/formatOllama)、CredentialFormat 类型扩展、凭证对话框按格式切 placeholder/隐藏 Key 框
- 进展：部署表单完善（2026-08-26）——①编辑回填修复：`DeploymentView` 补 `providerId` 联表字段（toView 经凭证带出），前端 `handleUpdateModelWhenEdit` edit 分支回填 `providerId` 并 `loadCredentials(providerId)` 刷新凭证下拉，修复"编辑部署时供应商下拉空、凭证无匹配项"（根因：`providerId` 为前端临时态、后端视图原不返回、编辑回填未反查）。②按Token计费内外定价落地：部署表单 `billingType=token` 时展示"外部官方定价"4键（绑 `litellm_params.input_cost_per_token` 等，¥/百万token，同步 LiteLLM 推送时换算 USD/token）+"内部结算定价"4键（绑 `model_info.internal_*`）；后端 `calcCosts` internal 分支启用 `internal_*`（有则用、未填回落 external 兼容历史数据）；提交时空值剔除防 `model_info` 污染。键名对齐 AIHelms，后端 `MergeCostsToModelInfo` 镜像机制已就绪。i18n 三处同步，typecheck + go build 通过
- 进展：使用场景 KeyScenario 落地（2026-08-26）——场景从"场景 Key 的自由文本 name"升级为密钥域字典实体：`gateway_key_scenario`(name/description/is_active) + AiKey 加 `scenario_id` 逻辑关联 + 列表/详情/identity 场景名回填(fillScenarioNames)。维护入口=密钥管理页双 Tab「密钥列表/场景管理」(key-scenario-panel，对齐 AIHelms AiKeyManage 场景 Tab)；建 Key 表单场景 Key 时"场景"下拉必选(启用中场景)、名称默认带出场景名可手改(手改后不跟随)、主 Key 恒无场景。同名按未软删行唯一(应用层查重+部分唯一索引 idx_keyscenario_name 兜底，停用行占名防二义、软删行不占名可重建——规避 AIHelms unique+is_active 软删撞死坑)；删除被未软删密钥引用时拒删。接口挂 `/gateway/ai-key/scenario/*` 子资源(静态段注册在 `:id` 前)，落在密钥管理菜单既有 api_prefix 内 casbin 零改动；scenarioId 前后端字符串传输(`json:",string"` 闭环，前端下拉 value 用 String(id))。i18n 三处同步，typecheck + go build + 路由测试通过

### 实现参照·AIHelms（2026-08-22 深度分析补，实现 P1 前必读）

> 来源：对 `/home/remote/AIHelms`（apps/，FastAPI）五路并行深度分析。总原则：**平台 DB 是唯一事实源，LiteLLM 是投影**——配置单向推送、日志单向回流，无双向协商。以下 6 条是设计正文未细化、但实现时必须照做的机制。

1. **LiteLLM 投影三原则**（参照 `apps/services/litellm_client.py`、`credential_service.py`）
   - 单向推送：每次 CRUD 同一事务边界内联同步 LiteLLM，不做定时全量对账；提供手动 resync 端点兜底
   - `litellm_key_id`/`litellm_model_id`/`litellm_synced` 是投影指针+状态位，不是第二份真相
   - 删除顺序：**先在 LiteLLM 侧禁用成功，再动本地行**（禁用失败→报冲突阻止删除）；凭证懒同步（`litellm_synced=False` 或投影≠DB 值才推）天然幂等
   - ⚠️ AIHelms 的坑：同步失败有的阻断回滚、有的静默记日志继续→网关漂移无补偿。Go 版统一走 `litellm_synced` 脏标记 + 重试补偿，不留静默路径

2. **路由池后缀约定**（参照 `model_service.py` `_get_litellm_model_name`/`sync_credential_routing`）
   - 部署不可路由（部署或凭证停用）→ `model_name` 加 `__disabled__` 后缀摘出 LB 组，**不删除重建**（`litellm_model_id` 稳定，历史成本/日志可回溯）
   - anthropic 格式部署进 `{model_id}(Anthropic)` 独立 model group（协议隔离，不同协议不能混组 LB）
   - **Key 授权与按模型限流都要做 Anthropic 变体扩展**：模型有 anthropic 格式活跃部署时，向 Key 下发 `models` 与 `model_tpm/rpm_limit` 时额外追加 `"{model_id}(Anthropic)"` 变体（防换协议绕开授权/限流）
   - 同一 Model 多 Deployment 注册相同 `model_name`（= `model_id`）即同一 LB 组；LB 选择完全委托 LiteLLM Router

3. **供应商差异表驱动**（参照 `provider_prefix_map` 表 + `litellm_credential_payload.py`）
   - `(provider_type, format, category) → prefix + needs_v1` 做成数据表（AIHelms 种子在 `docker/db/init.sql`，新增供应商走 SQL migration），不在代码里 switch
   - 凭证表单字段运行时从 LiteLLM `GET /public/providers/fields` 动态拉取渲染
   - 派生值边界收敛：¥/百万token → USD/token 换算、vLLM+anthropic 的 `extra_headers.authorization` 派生，只存在于发往 LiteLLM 前的纯函数（`BuildPayload()`），**绝不回写平台 DB**；平台内部永远人民币口径
   - 绑定平台凭证的部署强制剔除 inline `api_key`、写 `litellm_credential_name` 引用——改一次凭证全部部署生效

4. **用量回流幂等三件套**（参照 `apps/tasks/llm_log_tasks.py`，P1 用量 slice 直接照抄）
   - 游标锚定 `COALESCE(endTime, startTime)`：长 Agent 调用 endTime 远晚于 startTime，按 startTime 推进游标会永久跳过晚落盘行
   - 复合游标 `(time, request_id)` 行值比较 keyset 分页（批 1000、单次最多 50 批，游标不前进告警中止），游标存 KV 表（AIHelms `sync_state`）
   - 定时对账兜底：每小时 `NOT EXISTS(request_id)` 回灌近 30 天漏单
   - 落库 `request_id` 唯一约束 + `ON CONFLICT DO NOTHING`；归因链：`metadata.user_api_key_alias`→AiKey、`metadata.user_api_key_user_id`→User、`model_id`→Deployment（**入库时就解析归因**，别依赖查询侧按模型名兜底匹配——AIHelms 查询侧要试 6 种形态，是脆弱点）
   - master key / default_user_id 的行跳过，归因失败置 NULL 不强行归属

5. **成本口径**（参照 `llm_log_tasks.py` `_calc_internal/external_cost`、`model_service.py` 定价部分）
   - **本地重算，不信任 LiteLLM 的 spend 列**：内外成本都从自家价格表（deployment 级 `model_info` JSONB）计算；价格调整提供 `recalc` 全量回放
   - `billable_input = prompt_tokens − cache_read − cache_creation`（下限 0）；OpenAI/Anthropic 两种缓存字段格式都要解析
   - 金额一律高精度：AIHelms 用 PG `Numeric(12,6)` + Decimal；**Go 用 shopspring/decimal 或 int64 微元，禁 float64 累加**
   - ⚠️ 时区坑：AIHelms 按 UTC `date_trunc` 切日桶与业务日（上海 0 点）错位 8 小时。Go 版：存储统一 UTC，日桶显式按业务时区切（`time.LoadLocation` 后 truncate）

6. **聚合与预算都是派生缓存**（参照 `apps/tasks/efficiency_tasks.py`）
   - 日汇总表（AIHelms `cost_summary_daily`）滚动窗口 DELETE+INSERT 重建（近 60 天，每 5 分钟）而非增量 upsert：自愈（漏聚合/口径变更/补数据可修复）；聚合表不带独立状态机，游标只存在原始同步层
   - 部门/项目维度**不写进聚合表**，查询时 EXISTS/递归 CTE 读时归因——人员调岗不污染历史成本
   - `budget_used` 由定时任务按滚动窗口（1d/7d/30d）从原始日志 SUM **重算覆盖**（幂等可回放，无调用记录的 Key 归零），不做事件驱动累加
   - 硬停用借 LiteLLM `max_budget=0`（平时不设，避免两套记账口径）；⚠️ AIHelms 没做"超限自动停用"闭环，Go 版必须补：聚合任务发现超限 → 调停用流水线 → 通知

**AIHelms 关键文件索引**（相对 `/home/remote/AIHelms/apps/`）：LiteLLM 客户端 `services/litellm_client.py`｜凭证 payload `services/litellm_credential_payload.py`｜部署同步 `services/model_service.py`（`_build_litellm_params_for_sync`/`_get_litellm_model_name`）｜Key 服务 `services/ai_key_service.py`（`_expand_models_with_anthropic`/`sync_public_resource_to_all_keys`）｜日志回流 `tasks/llm_log_tasks.py`｜聚合预算 `tasks/efficiency_tasks.py`｜连通性测试 `services/access_test_{precheck,error_mapper,service}.py`｜表结构 `models/db.py`（935 行约 50 表）

