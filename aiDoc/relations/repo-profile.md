# 仓库画像 (repo-profile)

## 项目定位

**devops-admin** 是基于 gin-vue-admin 后端范式 + SoybeanAdmin 前端的管理平台，业务按模块组织。各模块职责与边界见 `aiDoc/modules/business-modules.md`。

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go + Gin + GORM + Casbin + Viper + Zap + JWT；module `github.com/hllkk/devops-admin/server`（简称 `devops-admin`） |
| 前端 | SoybeanAdmin 2.x：Vue3 + Vite + TypeScript + Pinia + NaiveUI + UnoCSS + Elegant Router + `@sa/axios` + vue-i18n；pnpm monorepo |
| 部署 | `deploy/` |

## 目录地图

```
/AGENT.MD                 # AI 协作规则唯一真源
/aiDoc/                   # 结构化 AI 文档层（见 aiDoc/README.md）
/server/                  # Go + Gin 后端（GVA 分层：api/router/service/model/initialize/plugin/global/middleware/utils/config/core/docs/source）
  └─ cmd/ mcp/ task/ resource/   # 额外目录（见 relations/system-map.md）
/web/                     # SoybeanAdmin 前端（已 scaffold）
/deploy/                  # 部署资产
```

## 当前实况（会变化，按需校准）

- `server/`：分层骨架完整（api/router/service/model/initialize/global/middleware/utils）。业务模型已建 `SysUser/SysRole/SysMenu/SysDept/SysPost` + `SysDictType/SysDictData` + 关联表 `SysUserRole/SysRoleMenu` + 对应 request DTO（`model/system/`、`model/system/request/`）；统一响应 `model/common/response`、雪花主键 `utils/snowflake`。**业务 Service/API/Router 正在逐步补齐**：`init`/`setting`/`loginlog`/`captcha` 等已建 service+api+router；`user`/`role`/`menu`/`dept`/`post`/`dict` 多数仍为 **Model-only**，对应 Service/API/Router 待补（进度见 `memory/demand-index.md`）。
- `web/`：**已 scaffold**（SoybeanAdmin 2.x + RuoYi 约定混合体）。系统模块页面齐备：`src/views/_admin/system/{user,role,menu,dept,post,dict,notice}`，接口封装 `src/service/api/system/*`，类型 `src/typings/api/system.api.d.ts`。前端先行、后端按 `Api.System.*` 反推实现是当前常态。
- go module = `github.com/hllkk/devops-admin/server`（仓库内简称 `devops-admin`）
