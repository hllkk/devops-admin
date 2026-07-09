# 系统关系图 (system-map)

## 根目录职责

- `server/`: Go + Gin 后端，包含路由、API、Service、Model、初始化和插件
- `web/`: SoybeanAdmin 前端，包含页面、路由、状态、接口封装、工具函数和插件
- `deploy/`: 部署相关资产
- `docs/`: 面向项目的人类文档与设计记录
- `aiDoc/`: 面向 AI 协作的结构化上下文

## 后端关系

后端保持现有分层方向：

1. `router/` 负责路由注册与中间件挂载
2. `api/` 负责参数绑定、请求校验、响应输出
3. `service/` 负责业务逻辑
4. `model/` 负责持久化模型和请求模型

`enter.go` 文件继续承担组合与暴露入口的职责。

### 额外目录

| 目录 | 用途 |
|---|---|
| `cmd/` | 命令行入口（如 `cmd/mcp/` MCP server 启动） |
| `mcp/` | MCP server：让 AI 助手与本平台交互；开发新功能前先通过 MCP 获支持 |
| `task/` | 定时任务（按模块需求，如过期数据清理） |
| `resource/` | 静态资源 / 模板 |

分层细则见 `aiDoc/modules/backend-layer-rules.md`。

## 前端关系

前端（SoybeanAdmin）一般遵循以下流向：

1. `src/service/api/` 负责接口调用（经 `@sa/axios`）
2. `src/store/modules/` 负责共享状态（Pinia）
3. `src/router/` 负责路由与权限入口（Elegant Router）
4. `src/views/` 负责页面
5. `src/utils/`、`src/hooks/`、`@sa/*` 负责可复用工具

前端细则见 `aiDoc/frontend-backend/frontend-rules.md`。

## 插件对称关系

如果某个能力以插件方式存在，尽量保持前后端结构对称：

- 后端：`server/plugin/<name>/`
- 前端：`web/src/plugins/<name>/`

当某个插件的职责和边界趋于稳定后，再把说明补充到 `aiDoc/modules/plugin-development.md`。
