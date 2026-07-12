# 后端 User / Role 模型建模

- 日期：2026-07-12
- 状态：设计中（设计文档已落 `docs/specs/2026-07-12-system-user-role-models-design.md`，待评审 → 实施）
- 关联前端：`web/src/typings/api/system.api.d.ts`（`Api.System.User` / `Api.System.Role`，commit 5199562 仅前端）

## 背景

前端已落地 RuoYi 风格用户/角色页面与类型（`/system/user`、`/system/role` 全套调用），后端无任何 user/role 模型（`SysUser` 注释桩，`SysRole` 不存在）。需定义后端 Model 层对齐前端契约。

## 契约决策（用户拍板）

1. **范围**：仅 Model 层（持久化模型 + 关联表 + request DTO + AutoMigrate 注册），不含 service/api/router。
2. **相关实体**：仅 User/Role + 关联表（`sys_user_role`、`sys_role_menu`）；`deptId` 存普通字段，`postIds` 留空数组；不建模 Dept/Post。
3. **契约 reconcile**：方案 A——User/Role 不内嵌 `OPS_MODEL`，自定义 int64 业务主键（`UserId`/`RoleId`，`json:",string"`），内嵌 system 模块自己的 `AuditModel` 对齐前端 `CommonRecord`。雪花回调不依赖基座，照常填主键。
4. **mixin 位置**：审计 mixin 放 `server/model/system/base.go`（命名 `AuditModel`）；其它模型继续用 `global.OPS_MODEL`。

## 设计要点

- `AuditModel`（`model/system/base.go`）：`CreateBy/CreateTime(autoCreateTime)/UpdateBy/UpdateTime(autoUpdateTime)/DeletedAt`，全 camelCase json，对齐 `CommonRecord`；附 `StatusEnable="0"`/`StatusDisable="1"` 常量。
- `SysUser`：`Password json:"-"`、`LoginDate *time.Time` 可空、`UserName uniqueIndex`；保留 `Login` 接口（鉴权链路 uint 暂不动）。
- `SysRole`：`Flag gorm:"-"` 瞬态；超管由初始化流程以显式 `RoleId=1` 种子。
- 关联表复合主键：插入必须显式传两个 ID（雪花回调仅填 0 值主键，不覆盖显式值）。
- request DTO：ID 字段全 `string`/`[]string`（线上 ID 全字符串防精度丢失），service 层转 int64；search 内嵌 `request.PageInfo`。
- AutoMigrate：`SysUser/SysRole/SysUserRole/SysRoleMenu` 进 `RegisterTables()`。

## 暂不动（下一阶段 / 已知技术债）

- service / api / router 三层
- Dept（树形）/ Post 建模
- 超管 `RoleId=1` 种子（初始化流程）
- 鉴权链路 int64/string 改造（`Login` 接口、JWT claims，见 [[snowflake-id-generator]] 同款 TODO）
- `UserName` uniqueIndex + 软删除冲突（后续可加含 `deleted_at` 联合唯一索引）

## 文档同步

- 修订 `aiDoc/modules/backend-layer-rules.md`：`GVA_MODEL` → `OPS_MODEL`，补 `AuditModel` 说明。
- 设计文档：`docs/specs/2026-07-12-system-user-role-models-design.md`。
