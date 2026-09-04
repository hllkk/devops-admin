# 超级管理员用户禁止删除（前后端双拦截）

> 2026-09-03。BUG：超管登录后在用户管理可勾选并删除超管自己，删掉后系统失去最高权限账号且无法从界面自愈。规则定为：**挂 SuperAdmin=true 角色的用户永不可删、不可被管理员侧改写**。同日二轮补齐全部管理员侧写接口（Delete/Update/UpdateStatus/ResetPwd 四处），堵住摘角色/重置密码/停用三类绕过删除保护的旁路。

## 后端拦截（`service/system/sys_user_manage.go`）

- 共享辅助 `protectedSuperAdminNames(ctx, ids)`：`sys_users JOIN sys_user_role JOIN sys_roles` 查挂 `super_admin = true` 角色的用户名（`Distinct().Pluck`，join 写法复用同文件 `GetList` 先例），四处写接口事务前统一调用，命中任一则**整体拒绝**
- `Delete`：禁删，提示 `不允许删除超级管理员(xxx),超级管理员是系统最高权限账号,请先调整其角色`
- `Update`：管理侧整体禁改——**关键旁路**：全量替换 roleIds 可把超管角色从超管身上摘掉，摘掉后保护查询不再命中、删除即成，故必须拦（超管本人改资料/密码走自助 profile 接口不受限）
- `UpdateStatus`：禁改状态（'0'正常/'1'停用，停用等效废号）
- `ResetPwd`：禁重置密码（有 resetPwd 权限的普通管理员重置超管密码=垂直提权接管；超管密码由本人 profile/updatePwd 自助改）
- 提示语统一带出超管用户名，API 层 `FailWithMessage(err.Error())` 透传给前端 message
- 保护哲学对齐 `sys_role.go` 超管角色保护（isProtectedRole）：既防误操作，更防垂直提权

## 前端禁勾（`views/_admin/system/user/index.vue`）

- selection 列加 `disabled: row => row.superAdmin`（NaiveUI `DataTableSelectionColumn` 原生回调，行数据 `superAdmin` 字段后端 `GetList` 早已回填 `GetSuperAdmin()`，类型注释「列表超管保护用」即为此准备）
- 该页超管的操作列本就 `return null`（无编辑/改密/删除单按钮）、状态开关本就 disabled，本次补齐的正是「批量勾选删除」这最后一个洞

## 已确认边界

- 唯一删除入口是 `DELETE /system/user/{userIds}`（BatchDeleteUser），旧 `deleteUser` 路由已注释，无旁路
- 超管用户集合的其它写入面已各自有保护：角色侧 `isProtectedRole` 禁改/禁删/禁授权进/禁取消授权超管角色（`service/system/sys_role.go`）
- **遗留观察项（未修，独立问题）**：`Create` 建用户时 `saveUserRoles` 直插 sys_user_role，若传超管角色 roleId 可造出新超管（垂直提权旁路，不击穿「不可删」但击穿角色侧「禁授权进超管角色」的意图）——建议后续在 saveUserRoles/Create 加超管角色校验

## 关联

[[user-management]]（用户管理接口层）、[[role-management]]（超管角色保护同源模式）；同日另修 [[user-center-route-merge]]。

## 验证

`go build ./...`/`go vet` 通过；join 查询在 dev 库（devops-pgsql-dev）实测命中超管用户 `super`；前端 `pnpm typecheck` 通过。二轮（Update/UpdateStatus/ResetPwd 补拦+辅助函数抽取）复验 build/vet 通过。

## 同日附带：版本号 v0.2.0 → v1.0.1

`server/global/version.go` 的 `Version` 为唯一版本源（前端「关于」弹窗经 `GET /system/upgrade/version` 读它；生产镜像经 Dockerfile ldflags 以 APP_VERSION 注入覆盖，`deploy/docker-prod/.env.example` 的 `APP_VERSION=prod` 只是 compose tag 示例默认值不必跟版本）。`web/package.json` 的 2.2.0 是 soybean 上游版本，不随项目版本。
