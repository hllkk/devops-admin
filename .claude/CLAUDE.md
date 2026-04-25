# VibeCoding — DevOps Admin (Root)

## 项目总览
运维管理系统 - 前后端分离架构

### 子项目
| 子项目 | 目录 | 技术栈 | .claude 位置 |
|--------|------|--------|--------------|
| 前端 | frontend/ | Vue3 + TS 6.0 + Naive UI + UnoCSS | frontend/.claude/ |
| 后端 | backend/ | Go 1.26 + Gin + GORM | backend/.claude/ |

## 使用指引
- 前端开发: 进入 frontend/ 目录，使用前端 VibeCoding 配置
- 后端开发: 进入 backend/ 目录，使用后端 VibeCoding 配置
- 跨模块任务: 在根目录协调，可使用 superpowers skills
- 代码规范: 请参考前端和后端的开发规范
- 前后端对接: 开发完成后，需要确认前端的请求和后端的响应是否一致，包括请求参数、数据类型、响应数据、请求方法、请求地址等。
- 代码提交: 先调用 code-simplifier agent 对代码进行简化，如果code-simplifier agent不可用，就使用simplify skill，优化完成需要用户测试通过确认后提交。

## 子模块管理
- frontend/ 是 git submodule (SoybeanAdmin 上游)
- 同步上游: `git fetch upstream && git merge upstream/main`

## 注意事项
- 前端使用 pnpm，后端使用 go mod
- 修改代码后重启相应服务