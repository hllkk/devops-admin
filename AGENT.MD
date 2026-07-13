# AGENT.MD

## 目的

本文件是本仓库内 AI 协作规则的**唯一真源**。

`.claude/`（以及未来可能的 `.codex/`、`.cursor/`、`.trae/`）下的规则文件仅作为兼容适配层，**不能再次演变成各自独立维护的 project rule 副本**。

## 读取顺序

按下面顺序加载项目上下文：

1. `AGENT.MD`
2. `aiDoc/README.md`
3. 按任务**只打开相关子目录**中的文件：
   - `aiDoc/relations/`（仓库结构、技术栈、开发流程）
   - `aiDoc/modules/`（后端分层、业务模块、插件）
   - `aiDoc/frontend-backend/`（前后端契约、前端规范、工具复用）
   - `aiDoc/examples/`（讲解型范例）
   - `aiDoc/memory/`（长期偏好与业务需求记忆）
4. 仅在当前工具确实依赖时，再读取工具目录下的适配文件。

若内容冲突，**以 `AGENT.MD` 为准**。

## 仓库概览

- `server/`：Go + Gin + GORM 后端（gin-vue-admin 同款分层范式），module 名 `devops-admin`
- `web/`：SoybeanAdmin 2.x 前端（Vue3 + Vite + TS + NaiveUI + UnoCSS + Elegant Router），pnpm monorepo
- `deploy/`：部署资产
- `docs/`：项目文档与设计记录
- `aiDoc/`：AI 协作文档层（本规则的结构化细则）

## 工程规则

### 架构

- 保持后端分层：`Router -> API -> Service -> Model`，依赖单向，**严禁跨层调用**
- `enter.go` 继续作为分组注册与组合入口
- API 层处理 HTTP 逻辑，**Service 层不得依赖 `gin.Context`**
- 对外接口的 **Swagger 注释必须与真实行为一致**
- 优先沿用 gin-vue-admin 现有模式，不做无关大改

### 前后端协作

- 明确请求与响应契约
- 保持统一响应结构：{ code, data, msg }
- 保持统一分页结构：{ page, pageSize, total, list }
- 前后端字段名与数据类型保持一致；Go 指针 nil ↔ TS `null`
- 优先复用 `web/` 的 `@sa/*` 包与 `src/{service,utils,hooks}`，**禁止重复造轮子**
- 涉及跨栈边界变更时，**同步更新 `aiDoc/frontend-backend/`**

### 前端上游同步

- 同步 soybean-admin 上游用 `scripts/sync-upstream.sh`：`--preview` 预览、`--run` 执行
- `web/` 是**直接复制导入**、未经 `git subtree add`，与上游无合并基点；脚本默认走 **apply 模式**：`git diff <track sha> <上游HEAD> | git apply --directory=web`，仅当检测到合并基点时才回退 `git subtree pull`
- 上游增量小且改动文件本地未动 → 0 冲突自动应用并提交；存在冲突则逐文件报告、**不改动工作区**，人工解决后重跑
- 细则与冲突处理见 `aiDoc/relations/development-workflow.md`

### 插件与模块

- 后端插件放 `server/plugin/<name>/`
- 前端插件按 `web/src/plugins/<name>/` 
- 稳定的模块职责、入口与边界说明放 `aiDoc/modules/`

### 示例文档

- `aiDoc/examples/` 是讲解型示例层，告诉 AI 每一层按什么标准**组织**代码
- 示例**不是逐字复制**，而是示范写法
- 新增某层文件前，先读对应示例再实现

### 记忆规则

- `aiDoc/memory/long-term/`：长期稳定的用户偏好、协作方式、跨任务约束，**一主题一文件**
- `aiDoc/memory/business/`：每次用户提出的业务需求（新增/修改模块、接口、页面、流程、业务规则、插件能力等），**一需求一文件**
- 用户提出业务需求时，AI **必须**新增或更新一条 `business` 记录，并同步更新 `aiDoc/memory/demand-index.md`
- 业务需求沉淀成长期偏好后，再提炼到 `long-term/`
- `README.md`、`project-memory.md`、`demand-index.md` 只承担说明或索引职责，**不承载多条记忆正文**
- 更新记忆时只改对应那条文件与必要索引，**不要重写汇总正文覆盖其他历史记忆**

### 文档维护

- `AGENT.MD` 只保留高层、稳定、跨任务的规则
- 细节上下文写入 `aiDoc/`
- 结构关系放 `aiDoc/relations/`；模块细则放 `aiDoc/modules/`
- 长期与业务记忆放 `aiDoc/memory/`
- 项目级 AI 规则变化时，**先改 `AGENT.MD`，再按需更新适配层**

### 代码读取约束

- **无论什么情况，都不要直接读取 `node_modules/` 中的代码**
- 了解第三方库行为时，优先看项目源码中的调用方式、锁文件、配置、官方文档或包的公开类型/说明，而不是进 `node_modules/` 逐文件读实现
- 后端同理：优先看 `go.mod` 与项目内调用，不深挖 vendor 实现

## AI 文档索引

- `aiDoc/README.md`
- `aiDoc/relations/repo-profile.md`
- `aiDoc/relations/development-workflow.md`
- `aiDoc/relations/system-map.md`
- `aiDoc/modules/backend-layer-rules.md`
- `aiDoc/modules/business-modules.md`
- `aiDoc/modules/plugin-development.md`
- `aiDoc/frontend-backend/boundary.md`
- `aiDoc/frontend-backend/frontend-rules.md`
- `aiDoc/frontend-backend/frontend-utils.md`
- `aiDoc/examples/README.md`
- `aiDoc/memory/README.md`
- `aiDoc/memory/project-memory.md`
- `aiDoc/memory/demand-index.md`
