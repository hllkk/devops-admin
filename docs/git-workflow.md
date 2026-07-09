# Git 工作流规范

## 仓库结构

Monorepo 单仓模式，所有代码在一个仓库中：

```
devops-admin/
├── server/     # Go 后端 (Gin + GORM)
├── web/        # Vue3 前端 (SoybeanAdmin)
├── deploy/     # 部署配置 (Docker Compose)
├── docs/       # 项目文档
└── aiDoc/      # AI 协作规则
```

## 分支策略

| 分支 | 用途 | 保护 |
|------|------|------|
| `master` | 生产就绪代码 | 禁止直接 push，只接受 MR/PR |
| `dev` | 日常开发集成分支 | 从 master 拉出 |
| `feature/<name>` | 新功能开发 | 从 dev 拉出，合并回 dev |
| `fix/<name>` | Bug 修复 | 从 dev 或 master 拉出 |
| `release/<version>` | 发布准备 | 从 dev 拉出，合并回 master + dev |

```
master ←── release/v1.0 ──→ dev ←── feature/xxx
  ↑                            ↑        ↑
  └── fix/critical-bug         └── fix/minor-bug
```

## 提交规范 (Conventional Commits)

```
<type>(<scope>): <subject>

feat(server): 新增用户管理 CRUD 接口
fix(web): 修复登录页密码框无法粘贴的问题
chore(deploy): 更新 Docker Compose 端口配置
docs: 补充 API 文档中的分页参数说明
refactor(server): 提取公共校验逻辑到 middleware
```

| Type | 说明 |
|------|------|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `docs` | 文档变更 |
| `style` | 代码格式（不影响逻辑） |
| `refactor` | 重构（不改变功能） |
| `perf` | 性能优化 |
| `test` | 测试相关 |
| `chore` | 构建/工具/依赖变更 |
| `revert` | 回滚提交 |

## 标准提交流程

```bash
# 1. 从 dev 拉取最新
git checkout dev
git pull origin dev

# 2. 创建功能分支
git checkout -b feature/my-feature

# 3. 开发并提交（频繁小步提交）
git add server/api/user.go server/service/user.go
git commit -m "feat(server): 新增用户列表查询接口"

# 4. 推送功能分支
git push origin feature/my-feature

# 5. 在 Gitee 上创建 Pull Request → dev

# 6. 合并后删除本地功能分支
git checkout dev
git pull origin dev
git branch -d feature/my-feature
```

## SSH 密钥配置（可选，推荐替代 HTTPS）

```bash
# 生成 SSH 密钥（如果没有）
ssh-keygen -t ed25519 -C "your-email@example.com"

# 查看公钥，添加到 Gitee 设置中
cat ~/.ssh/id_ed25519.pub

# 切换远程地址为 SSH
git remote set-url origin git@gitee.com:huang_lei521/devops-admin.git
```

## 同步 SoybeanAdmin 上游框架更新

前端 `web/` 基于 SoybeanAdmin，通过 `git subtree` 机制追踪上游框架更新。

### 预览上游变更

```bash
bash scripts/sync-upstream.sh
```

### 执行同步

```bash
bash scripts/sync-upstream.sh --run
```

### 冲突策略

Subtree `--squash` 合并时，冲突按文件所属层级自动解决：

```
web/packages/**          → 接受上游 (框架核心包)
web/build/**             → 接受上游 (构建配置)
web/pnpm-lock.yaml       → 接受上游 (依赖锁文件)
web/src/views/**         → 保留本地 (业务页面)
web/src/router/routes/** → 保留本地 (路由配置)
web/src/service/**       → 保留本地 (API 定义)
web/src/layouts/**       → 保留本地 (布局定制)
web/src/store/**         → 保留本地 (状态管理)
web/src/locales/**       → 保留本地 (国际化文本)
web/.env*                → 保留本地 (环境变量)
web/vite.config.ts       → 保留本地 (构建配置)
web/package.json         → ⚠ 人工合并 (新依赖 vs 本地脚本)
```

### 同步后验证

```bash
cd web
pnpm install         # 安装可能新增的依赖
pnpm dev             # 验证前端启动正常
```

## 从远程项目迁移代码时的注意事项

远程项目 (172.21.10.40) 的代码可以直接复制使用，但注意：

1. **手动复制文件**，不要用 submodule 或 subtree
2. **不要复制 `.git` 目录**
3. **检查硬编码的配置**（IP、端口、密钥等）
4. **适配包名/路径** — 远程用 `backend/`，本地用 `server/`
5. **检查 LICENSE** — 远程项目代码属于原作者 (hllkk)
