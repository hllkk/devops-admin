# VibeCoding — DevOps Admin (Root)

## 项目总览
运维管理系统 - 前后端分离架构

### 子项目
| 子项目 | 目录 | 技术栈 | .claude 位置 |
|--------|------|--------|--------------|
| 前端 | frontend/ | Vue3 + TS 6.0 + Naive UI | frontend/.claude/ |
| 后端 | backend/ | Go 1.26 + Gin + GORM | backend/.claude/ |

## 使用指引
- **前端开发**: 进入 frontend/ 目录，使用前端 VibeCoding 配置
- **后端开发**: 进入 backend/ 目录，使用后端 VibeCoding 配置
- **跨模块任务**: 在根目录协调，可使用 superpowers skills

## 子模块管理
- frontend/ 是 git submodule (SoybeanAdmin 上游)
- 同步上游: `git fetch upstream && git merge upstream/main`

## 注意事项
- 前端使用 pnpm，后端使用 go mod
- 修改代码后重启相应服务