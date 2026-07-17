# 仓库画像 (repo-profile)

## 项目定位

**devops-admin** 是基于 gin-vue-admin 后端范式 + SoybeanAdmin 前端的管理平台，业务按模块组织。各模块职责与边界见 `aiDoc/modules/business-modules.md`。

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go + Gin + GORM + Casbin + Viper + Zap + JWT + Redis + MongoDB(qmgo)；module `github.com/hllkk/devops-admin/server`（简称 `devops-admin`） |
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

## 当前实况（2026-07-16 校准，会变化，按需校准）

- `server/`：自 `1d632d9` 起统一 `OPS_` 命名并重建系统基座。分层目录齐备：`router/service/model/initialize/global/middleware/utils/config/core/docs`；`api/` **当前为空**（业务 API 待随各模块重建）；`source/`、`cmd/`、`resource/`、`plugin/` 目录存在但暂无内容，`task/`（clearTable/registry）、`mcp/`（context/http_client）有骨架。
  - 基座三层在 `global/model.go`：`OPS_BASE`（无主键）/ `OPS_MODEL`（含 ID，内部表）/ `OPS_AUDIT_MODEL`（CreateBy/UpdateBy，对外实体）；全局变量统一 `OPS_` 前缀（`global/common.go`：`OPS_DB`/`OPS_DBList`/`OPS_REDIS`/`OPS_MONGO`/`OPS_CONFIG`/`OPS_CACHE` 等），`GVA_` 残留已清零。
  - 已建模型（`model/system/`）：业务实体 `SysUser`（已上 `OPS_AUDIT_MODEL` + 自定义 `UserId`）、`SysRole`（仍旧形态：手写时间戳、主键 `RoleId` json `id`）、`SysDepartment`、`SysPosition`；系统表 `JwtBlacklist`/`SysSecurityConfig`/`SysError`/`SysDataAccessLog`/`SysTimedTask`/`SysTimedTaskLog`；关联表 `SysUserRole`/`SysRoleDepartment`/`SysUserDepartment`。**无** `SysMenu`/`SysDictType`/`SysDictData`（重构后未保留，待重建）。
  - Service 骨架（`service/system/`）：`data_scope`/`jwt_black_list`/`sys_error`/`sys_security_config`/`sys_timed_task`（+http+runner）/`auto_code`；`service/media/` 有上传/附件分类。业务 API/Router（user/role/dept/post/menu/dict 等）**待补**，`router/{system,media}` 当前仅 `enter.go`。
  - 统一响应 `model/common/response`（`Response{code,data,msg}`，`SUCCESS="0000"`/`ERROR="7"`）、分页 `model/common/request.PageInfo`（`PageNum/PageSize/Keyword`，`MaxPageSize=100`，`LimitOffset()`）。雪花主键回调与 `utils/snowflake` **待落地**，当前 DB 自增。
  - 进度轨迹见 `memory/demand-index.md`（多为重构前的实现记录，重构后部分能力需重建）。
- `web/`：**已 scaffold**（SoybeanAdmin 2.x + RuoYi 约定混合体）。系统模块页面齐备：`src/views/_admin/system/{user,role,menu,dept,post,dict,notice}`，接口封装 `src/service/api/system/*`，类型 `src/typings/api/system.api.d.ts`。前端先行、后端按 `Api.System.*` 反推实现是当前常态。
- go module = `github.com/hllkk/devops-admin/server`（仓库内简称 `devops-admin`）
