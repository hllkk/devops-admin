# 前端自动化升级脚本使用指南

## 概述

`sync_auto.py` 是一个非交互式的前端升级脚本，用于自动同步 SoybeanAdmin 上游更新到本地仓库。

## 使用方式

```bash
# 进入 scripts 目录
cd /home/devops-admin/scripts

# 同步 upstream 到 main 并合并到 dev（默认）
python3 sync_auto.py --dir /home/devops-admin/frontend

# 仅同步到 main 分支
python3 sync_auto.py --main-only --dir /home/devops-admin/frontend

# 预览变更（不执行任何操作）
python3 sync_auto.py --preview --dir /home/devops-admin/frontend

# 模拟运行（显示将执行的操作但不实际执行）
python3 sync_auto.py --dry-run --dir /home/devops-admin/frontend

# 合并后恢复本地生成的文件（如路由配置）
python3 sync_auto.py --restore-local --dir /home/devops-admin/frontend
```

## 参数说明

| 参数 | 说明 |
|------|------|
| `--main-only` | 仅同步 upstream 到 main 分支，不合并到 dev |
| `--preview` | 预览上游变更，显示版本差异、变更统计、提交列表 |
| `--dry-run` | 模拟运行，显示将执行的操作但不实际执行 |
| `--restore-local` | 合并后恢复本地生成的文件（如 elegant-router 生成的文件） |
| `--dir <path>` | 指定前端目录路径（默认使用配置文件中的值） |

## 配置文件

配置文件位于 `sync_config.json`：

```json
{
  "frontend_dir": "/home/devops-admin/frontend",
  "upstream_remote": "upstream",
  "upstream_branch": "main",
  "main_branch": "main",
  "dev_branch": "dev",
  "auto_accept_patterns": [
    "pnpm-lock.yaml",
    "package-lock.json",
    "yarn.lock",
    "*.md",
    "package.json"
  ],
  "keep_local_patterns": [
    "src/router/elegant/*.ts",
    "src/typings/*.d.ts",
    ".env*",
    "vite.config.ts"
  ],
  "commit_message": "chore: 同步 upstream"
}
```

### 配置项说明

| 配置项 | 说明 |
|--------|------|
| `frontend_dir` | 前端项目目录路径 |
| `upstream_remote` | 上游远程仓库名称（默认 upstream） |
| `upstream_branch` | 上游分支名称（默认 main） |
| `main_branch` | 本地主分支名称（默认 main） |
| `dev_branch` | 本地开发分支名称（默认 dev） |
| `auto_accept_patterns` | 自动接受上游版本的文件模式 |
| `keep_local_patterns` | 合并后需要恢复的本地文件模式 |
| `commit_message` | 同步提交消息前缀 |

## 冲突处理策略

脚本自动处理冲突文件：

| 文件类型 | 处理策略 |
|----------|----------|
| `package.json` | 接受上游版本（框架依赖更新） |
| `pnpm-lock.yaml` | 接受上游版本 |
| `*.md` 文件 | 接受上游版本 |
| `src/router/elegant/*.ts` | 保留本地版本（自动生成） |
| `src/typings/*.d.ts` | 保留本地版本（自动生成） |
| `.env*` | 保留本地版本（环境配置） |
| `vite.config.ts` | 保留本地版本（项目配置） |
| 其他文件 | 默认接受上游版本 |

## 工作流程

### 默认流程（同步到 main 并合并到 dev）

1. 检查当前分支和未提交更改
2. 如有未提交更改，自动暂存（stash）
3. 切换到 main 分支
4. Fetch upstream 更新
5. 检查是否有新版本
6. Merge upstream/main（自动处理冲突）
7. Push main 分支
8. 切换到 dev 分支
9. Merge main（自动处理冲突）
10. Push dev 分支
11. 切换回原始分支
12. 恢复暂存的更改

### --restore-local 流程

额外步骤：
- 合并完成后，恢复 `keep_local_patterns` 中匹配的文件

## 与原脚本对比

| 特性 | `sync_frontend.py`（原脚本） | `sync_auto.py`（新脚本） |
|------|------------------------------|--------------------------|
| 运行模式 | 交互式 | 非交互式 |
| CI/CD 适用 | 不适用 | 适用 |
| 冲突处理 | 手动选择 | 自动处理 |
| 参数支持 | 无 | 多种参数 |
| 模拟运行 | 无 | 支持（--dry-run） |
| 预览功能 | 交互式预览 | 命令行预览 |

## 最佳实践

### 升级前检查

```bash
# 1. 预览变更
python3 sync_auto.py --preview --dir /home/devops-admin/frontend

# 2. 确认无重大破坏性变更后执行
python3 sync_auto.py --restore-local --dir /home/devops-admin/frontend

# 3. 更新依赖
cd /home/devops-admin/frontend && pnpm install

# 4. 运行验证
pnpm typecheck && pnpm lint
```

### 定期升级

建议每周或每月检查上游更新：

```bash
# CI/CD 定时任务示例
0 9 * * 1 python3 /home/devops-admin/scripts/sync_auto.py --preview --dir /home/devops-admin/frontend > /home/devops-admin/logs/sync_preview.log
```

## 故障排查

### upstream 远程不存在

```bash
cd /home/devops-admin/frontend
git remote add upstream https://github.com/soybeanjs/soybean-admin.git
git fetch upstream
```

### 合并冲突无法自动解决

手动处理后继续：

```bash
cd /home/devops-admin/frontend
# 查看冲突文件
git diff --name-only --diff-filter=U
# 手动解决冲突后
git add -A
git commit -m "chore: 解决合并冲突"
git push origin main
```

### 需要回滚

使用交互式脚本的回滚功能：

```bash
python3 sync_frontend.py
# 选择 [4] 回滚最近同步
```