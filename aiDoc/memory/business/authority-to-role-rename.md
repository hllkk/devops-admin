# 全局命名统一 Authority → Role

- 日期：2026-07-17
- 状态：已落地（`go build ./...` / `go vet` 通过，`go fmt` 对齐修正，authority 零残留）
- 动机：项目角色概念统一用 Role 更直观，清除 GVA 残留的 Authority 命名

## 范围

**前端 `web/src` 已全部是 Role，无任何 authority，本次零改动。**
**后端 11 文件、约 38 处**（紧随 [[system-model-rebuild]] 同会话完成）。

## 替换映射

| 原值 | 新值 | 说明 |
|---|---|---|
| `AuthorityId` | `RoleId` | `BaseClaims.AuthorityId`、`GetUserAuthorityId`→`GetUserRoleId`、casbin `waitUse.AuthorityId` |
| `AuthorityID` | `RoleID` | `datascope.Identity/AuditEvent`、`SysDataAccessLog` |
| `authorityID`/`authorityId`/`authority_id` | `roleID`/`roleId`/`role_id` | 局部变量、json tag、zap 字段名 |
| `SysAuthority` / `InitAuthority*` | `SysRole` / `InitRole*` | 注释（datascope/router/gorm） |

## 副作用（重建期均无影响）

- **casbin sub 语义**：`middleware/casbin_rbac.go` 的 `sub` 从 authorityId 字符串 → roleId 字符串。当前空库无 casbin 策略数据，将来 seed 时按 roleId 生成即可。
- **`SysDataAccessLog` 列/JSON**：DB 列 `authority_id`→`role_id`、json `authorityId`→`roleId`。该表为后端内部数据权限审计表，前端 log 模块不直接消费，安全。

## 关键文件

- `model/system/request/jwt.go`（BaseClaims.RoleId）
- `utils/claims.go`（GetUserRoleId / cl.RoleId / LoginToken 填充）
- `utils/datascope/datascope.go`（Identity.RoleID / AuditEvent.RoleID）
- `service/system/data_scope.go`（BuildIdentity roleID 参数）
- `middleware/{data_scope,access_log,casbin_rbac}.go`
- `model/system/sys_data_access_log.go` + `service/system/sys_data_access_log.go`
- `initialize/{router,gorm}.go`（注释）

## 方法

`find server -name '*.go' -exec sed -i -e ... {} +` 批量替换（7 条规则互不嵌套），随后 `go fmt ./...` 修正 sed 改变字段名长度导致的 struct 对齐错位。

## 关联

[[system-model-rebuild]]（同会话先完成的 model 重建，本次在其基础上继续命名清理）
