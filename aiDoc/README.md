# aiDoc

`aiDoc/` 是 devops-admin 仓库的**结构化 AI 文档层**，把长期有效的项目上下文从工具目录中抽离，按主题拆成可维护的约束文档。

## 使用方式

1. 先读 `AGENT.MD`
2. 再看本索引
3. 按任务**只打开相关子目录**
4. 不再把项目级规则塞回 `.claude/` 等工具目录

## 目录说明

| 目录 | 内容 |
|---|---|
| `relations/` | 仓库结构、技术栈、依赖关系、开发流程 |
| `modules/` | 后端分层规则、业务模块、插件结构 |
| `frontend-backend/` | 前后端契约、前端规范、工具函数复用规则 |
| `examples/` | 讲解型示例，告诉 AI 每层按什么标准组织与书写 |
| `memory/` | AI 记忆层，拆分为长期记忆与业务记忆 |

## 常用入口

- `relations/repo-profile.md`：项目定位、技术栈、核心特性、目录地图
- `relations/system-map.md`：系统结构关系图、前后端流向、插件对称关系
- `relations/development-workflow.md`：开发流程、分支与提交规范
- `modules/backend-layer-rules.md`：后端分层、`enter.go`、统一响应、Swagger 约束
- `modules/business-modules.md`：业务模块（服务器管理、AI 网关）
- `modules/plugin-development.md`：前后端插件结构与开发流程
- `frontend-backend/boundary.md`：前后端契约与字段类型约束
- `frontend-backend/frontend-rules.md`：Soybean 前端代码、状态、路由、样式规范
- `frontend-backend/frontend-utils.md`：`@sa/*` 与 `src/{service,utils,hooks}` 复用规则
- `examples/README.md`：示例层总入口
- `memory/README.md`：记忆层总入口
- `memory/project-memory.md`：记忆层说明入口
- `memory/demand-index.md`：业务需求索引

## 维护原则

- 稳定规则放这里，不放工具私有目录
- 临时会话草稿不入库，变成长期知识才记录
- 适用于所有 AI 的项目级规则，先写进 `AGENT.MD`，细节再拆到 `aiDoc/`
- 用户提出业务需求时，同步更新 `memory/business/` 与 `demand-index.md`
