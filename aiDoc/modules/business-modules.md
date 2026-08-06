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

- **Provider**（供应商）：`name`/`provider_type`(openai/anthropic/vllm…)/`billing_type`(token/per_call/monthly_quota)/`monthly_budget`/`monthly_used`/`is_active`
- **Credential**（凭证）：`credential_name`（全局唯一，对应 LiteLLM credential_name）/`credential_values`（JSONB：api_key、api_base）/`provider_id`/`litellm_synced`/`is_active`
- **Model**（模型）：`model_id`（全局唯一，对应 LiteLLM model_name）/`name`/`category`/`capabilities`/`is_published`/`visibility_type`(all/selected)/`requires_approval`
- **ModelDeployment**（部署）：`model_id` + `credential_id` + `litellm_params`（LiteLLM 路由参数 JSONB：model、api_base…）+ `model_info`（内外定价 JSONB）+ `is_active`。一个 Model 多 Deployment → LiteLLM 同 `model_name` 多部署形成负载均衡池
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

1. 供应商管理 Provider（CRUD + 计费类型 + 月预算）
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

