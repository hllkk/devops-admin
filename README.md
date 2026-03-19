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
