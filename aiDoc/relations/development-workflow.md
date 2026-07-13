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

## 前端上游同步（soybean-admin）

`web/` 基于 SoybeanAdmin 2.x，上游更新通过 `scripts/sync-upstream.sh` 同步。

### 原理与策略
- `web/` 是**直接复制导入**的（`ff9817e`），不是 `git subtree add`，与上游**没有共同合并基点**。
  因此 `git subtree pull` 会按“不相关历史”合并、引发大量无关冲突，**不能直接用**。
- 脚本 `--run` 自动检测策略：
  - **apply 模式（默认 / 当前）**：`git diff <web/.upstream-track 首行 sha> <上游 HEAD> | git apply --directory=web`
    只把“上游自上次同步以来的真实增量”应用到 `web/`，干净、0 误伤，应用后自动提交并更新跟踪文件。
  - **subtree 模式（兜底）**：仅当检测到合并基点（将来若做过 `git subtree add`）才回退 `git subtree pull --squash`，并按框架层接受上游 / 业务层保留本地的策略解冲突。
- 预览的文件 diff 用 `git subtree split` 把本地 `web/` 映射到根命名空间再对比上游，避免“上游在根、本地在 `web/` 下”造成的虚假全删统计。

### 用法
```bash
bash scripts/sync-upstream.sh            # 预览（不写盘）：版本差异、上游增量文件
bash scripts/sync-upstream.sh --run      # 执行同步：apply 后自动 commit + 更新 .upstream-track
```
- 跟踪文件 `web/.upstream-track` 第一行 = 上次同步到的上游 commit，**勿手动改**。
- `--run` 要求工作区干净（脚本会自带提交）。

### 冲突处理（apply 模式）
当某文件本地已改动、补丁无法干净匹配时：
- 脚本**不改动工作区**，逐文件列出 `[ok]` / `[CONFLICT]`。
- 对冲突文件用 3way 合并：
  ```bash
  git diff <旧 track sha> <上游 HEAD> -- <file> | git apply --directory=web --3way
  ```
- 解决后再 `--run`，干净文件会自动应用。

### 环境
- 上游仓库：`git@github.com:soybeanjs/soybean-admin.git`（SSH 主路径，HTTPS 兜底）。
- 本环境 `github.com:443` 不通，但 SSH(22) 与 `codeload.github.com` 可达；`git-subtree` 需 `dnf install --disablerepo='docker-ce*' git-subtree`（appstream，docker repo 在此网络不可达需跳过）。
