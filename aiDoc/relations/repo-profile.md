# 仓库画像 (repo-profile)

## 项目定位

**devops-admin** 是基于 gin-vue-admin 后端范式 + SoybeanAdmin 前端的管理平台，业务按模块组织。各模块职责与边界见 `aiDoc/modules/business-modules.md`。

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go + Gin + GORM + Casbin + Viper + Zap + JWT；module `devops-admin` |
| 前端 | SoybeanAdmin 2.x：Vue3 + Vite + TypeScript + Pinia + NaiveUI + UnoCSS + Elegant Router + `@sa/axios` + vue-i18n；pnpm monorepo |
| 部署 | `deploy/` |

## 目录地图

```
/AGENT.MD                 # AI 协作规则唯一真源
/aiDoc/                   # 结构化 AI 文档层（见 aiDoc/README.md）
/server/                  # Go + Gin 后端（GVA 分层：api/router/service/model/initialize/plugin/global/middleware/utils/config/core/docs/source）
  └─ cmd/ mcp/ task/ resource/   # 额外目录（见 relations/system-map.md）
/web/                     # SoybeanAdmin 前端（待 scaffold）
/deploy/                  # 部署资产
```

## 当前实况（会变化，按需校准）

- `server/`：目录骨架已建，**尚无业务代码**（仅 `main.go` + `config.yaml`）
- `web/`：**尚未 scaffold**；`frontend-rules.md` 为 SoybeanAdmin 2.x 前瞻性规范
- go module = `devops-admin`
