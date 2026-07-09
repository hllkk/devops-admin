# 开发流程 (development-workflow)

## 分支策略
- `main`：生产
- `develop`：开发
- `feature/*`：功能
- `hotfix/*`：紧急修复

## 提交规范
- `type(scope): desc`，type ∈ feat|fix|docs|style|refactor|test|chore
- 前端优先 `pnpm commit`（`sa git-commit`，`-l=zh-cn` 中文）
- 后端按语义化提交

## 后端开发工作流（自下而上）
1. **模型设计**：先定 `model` + `model/request`（奠定基础）
2. **Service** → **API**（带 Swagger）→ **Router** 逐层实现
3. **Initialize**：注册路由 + `AutoMigrate`
4. 交付：完整代码 + 相对路径说明（如 `server/service/file/file_service.go`）
- 新增某层文件前，先读 `aiDoc/examples/backend/`（待补）对应示例

## 前端开发工作流
1. 接口封装：`src/service/api/` + 类型 `src/typings/api/`
2. 页面 / 组件：`src/views/`、`src/components/`
3. 路由：新增 `src/views/*` 后跑 `pnpm gen-route` 重生成 Elegant 路由
4. 状态：按需在 `src/store/modules/` 加 Pinia store
5. i18n：用户可见文案写入 `src/locales/langs/{zh-cn,en-us}.ts`
- 提交前必须过 `pnpm typecheck && pnpm lint && pnpm fmt`（`simple-git-hooks` 强制）

## 联调
- 后端 Swagger 为准；前端 mock 对齐统一响应 `{ code, data, msg }`（成功码见 `aiDoc/frontend-backend/boundary.md`）
- 接口 / 字段 / 枚举变更同步更新 `aiDoc/frontend-backend/boundary.md` 与 `aiDoc/modules/business-modules.md`
