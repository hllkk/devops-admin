# AI 网关·立项与方案确认

> 2026-08-06。参照 `/home/remote/AIHelms` 引入 AI 网关，经分析确认可行，方案已拍板。

## 背景

用户要求实现 AI 网关功能，参考项目 `/home/remote/AIHelms`（企业 AI 资源纳管平台：FastAPI + LiteLLM + PostgreSQL + Redis + Celery，前端 Vue3 + TailwindCSS）。已完成三路分析：AIHelms 后端核心实现、AIHelms 前端页面、devops-admin 现状与部署栈。

## 方案决策（用户 AskUserQuestion 确认）

- **底座**：引入 LiteLLM 官方镜像作模型代理底座，**不自研转发**（与 AIHelms 同思路）。多 provider 兼容、token 计费、流式转发、路由负载均衡全交给 LiteLLM。
- **第一期范围**：核心五件套——供应商/凭证/模型(含部署)/AI 密钥/用量统计+看板。

## 关键约束

- 后端 Go 只能参照 AIHelms 设计重写（Python→Go），不搬代码；但对接 LiteLLM 只是 HTTP 调管理 API，Go 侧很轻。
- **管理面/转发面分离**：Go 做 Provider/Credential/Model/AiKey 的 CRUD + 同步到 LiteLLM + 用量统计；转发由 LiteLLM 承担，客户端直连 LiteLLM。
- **四层数据模型**：Provider → Credential → Model + ModelDeployment → AiKey。devops-admin 侧表存"管理元数据 + LiteLLM 引用 ID"（`litellm_key_id`/`litellm_model_id`/`litellm_synced`）。
- AiKey 归属 `owner_id` 复用 `SysUser`/`SysDepartment`，认证走现有 httpOnly cookie。
- 部署：`deploy/docker-prod/` 加 litellm 容器，**共用现有 PostgreSQL**（devops-admin 已是 PG+Redis，与 AIHelms 同构，引入成本低）。
- 困难点：双向同步一致性（禁用/删除级联）、用量日志准确性（需理解 LiteLLM spend 表）、前端 5+ 页面全栈重写（NaiveUI+UnoCSS，参照但不能搬 AIHelms 的 TailwindCSS 页面）。详见 `modules/business-modules.md`「AI 网关模块」节。

## 现状（2026-08-06）

- 前端 `views/home/index.vue` AI 身份首页 mock 已落地（见 [[ai-gateway-identity-home]]）；`views/_gateway/gateway` 占位页；路由 `/gateway`+`/home` 预留；`service/api` 下无 gateway 目录、无业务契约 typings。
- 后端 `/gateway/*` 全缺；`config/ai.go` + `service/system/sys_ai.go` + `auto_code_llm.go` 是错误日志 AI 分析，非网关。

## 分期（4 期；P1–P3 主线，P4 可选）

对照 AIHelms 全功能地图 + devops-admin 已有 system 基座（用户/角色/部门/认证/数据权限/操作·登录日志），AIHelms 的 `users`/`departments`/`projects`/`roles`/`auth`/`audit_logs`/`license`/`branding` 不重做，只做 **AI 特有**功能。

- **P1 · 核心网关五件套**（已确认，进行中）：供应商/凭证/模型含部署/AI 密钥/用量统计+看板 + LiteLLM 容器接入。详见 `modules/business-modules.md`。
- **P2 · AI 市场**：MCP Server 管理 + Skill 管理 + 资源申请审批 + AiKey 的 `mcps`/`skills` 授权扩展（P1 预留字段）。
- **P3 · 成本效能与预算管控**：成本分析（内外双轨明细下钻 + Token 列 + 人员 Top10）+ 预算管控（多维预算人/部门/项目/模型，软限预警/硬限超支停用对接 LiteLLM `max_budget`）+ 覆盖率/采用度 + 健康检查 + 效能报告（周/月自动生成 + 导出）。
- **P4 · 安全审计与智能体（可选）**：AI 策略（SkillSpector 扫描 Skill + LLM 审查 + 高风险 prompt 拦截）+ 智能体中心 + 业务场景模板。

## 关联

- 模块设计正文：`modules/business-modules.md`「AI 网关模块」
- 已落地前端：[[ai-gateway-identity-home]]
- 参照蓝本：`/home/remote/AIHelms`（后端 `apps/services/litellm_client.py`、`apps/models/db.py`；前端 `ui/packages/{admin,web}`）
