# DevOps Admin

运维管理系统 - 前后端分离架构

## 项目结构

```
devops-admin/
├── frontend/     # 前端 (Vue3 + SoybeanAdmin)
├── backend/      # 后端 (Gin + GORM)
└── deploy/       # 部署配置
```

## 快速开始

### 克隆项目（包含 submodule）

```bash
git clone --recursive git@github.com:hllkk/devops-admin.git
```

### 前端

```bash
cd frontend
pnpm install
pnpm dev
```

### 后端

```bash
cd backend
go mod tidy
go run main.go
```

## 同步上游更新

### 前端（同步 SoybeanAdmin）

```bash
cd frontend
git fetch upstream
git merge upstream/main
```

### 后端

直接在 backend 目录提交即可。

## 技术栈

- **前端**: Vue3 + TypeScript + Naive UI + SoybeanAdmin
- **后端**: Go + Gin + GORM

## 分支策略

### 前端分支

| 分支 | 用途 | 说明 |
|------|------|------|
| `main` | 上游同步 | 保持与 SoybeanAdmin 同步，不直接修改 |
| `dev` | 项目开发 | 日常开发分支，功能完成后可合并到 main |
| `feature/*` | 功能开发 | 新功能开发 |

### 后端分支

| 分支 | 用途 |
|------|------|
| `main` | 生产版本 |
| `dev` | 开发分支 |

### 同步流程

```
upstream/main → main → dev
```

上游更新会自动同步到 `main`，然后合并到 `dev` 分支。

## 分支策略

### 前端分支

| 分支 | 用途 | 说明 |
|------|------|------|
| `main` | 上游同步 | 保持与 SoybeanAdmin 同步，不直接修改 |
| `dev` | 项目开发 | 日常开发分支，功能完成后可合并到 main |
| `feature/*` | 功能开发 | 新功能开发 |

### 后端分支

| 分支 | 用途 |
|------|------|
| `main` | 生产版本 |
| `dev` | 开发分支 |

### 同步流程

```
upstream/main → main → dev
```

上游更新会自动同步到 `main`，然后合并到 `dev` 分支。

### BUG收集
```
1. 项目初始化之后，无法正确跳转到登录页面，需要刷新一次才可以
```
