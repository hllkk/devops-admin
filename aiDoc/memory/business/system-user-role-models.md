# 后端 User / Role 模型建模

- 日期：2026-07-12
- 状态：Model 层已落地（commit `7b3f318`→`e4ea5b5`：模型/DTO/关联表/AutoMigrate/迁移测试均已合入）。service/api/router 三层仍未实现。
- **后续演进（2026-07-12）**：审计基座从 `model/system/base.go`（`AuditModel`）上移至 `server/global/common.go`，命名 `global.OPS_AUDIT_MODEL`（内嵌 `global.OPS_MODEL`）；`OPS_MODEL` 同步去掉 `ID`、时间字段改 `CreateTime/UpdateTime`。`base.go` 仅留状态常量。详见「文档同步」末条。
- 关联前端：`web/src/typings/api/system.api.d.ts`（`Api.System.User` / `Api.System.Role`，commit 5199562 仅前端）

## 背景

前端已落地 RuoYi 风格用户/角色页面与类型（`/system/user`、`/system/role` 全套调用），后端无任何 user/role 模型（`SysUser` 注释桩，`SysRole` 不存在）。需定义后端 Model 层对齐前端契约。

## 契约决策（用户拍板）

1. **范围**：仅 Model 层（持久化模型 + 关联表 + request DTO + AutoMigrate 注册），不含 service/api/router。
2. **相关实体**：仅 User/Role + 关联表（`sys_user_role`、`sys_role_menu`）；`deptId` 存普通字段，`postIds` 留空数组；不建模 Dept/Post。
3. **契约 reconcile**：方案 A——User/Role 不依赖 `OPS_MODEL` 的主键，自定义 int64 业务主键（`UserId`/`RoleId`，`json:",string"`），内嵌审计基座对齐前端 `CommonRecord`。雪花回调不依赖基座，照常填主键。
4. **基座位置（演进后）**：审计基座 `global.OPS_AUDIT_MODEL` 位于 `server/global/common.go`（内嵌 `global.OPS_MODEL` 时间戳 + `CreateBy`/`UpdateBy`）；生命周期基座 `global.OPS_MODEL` 不含主键。早期方案曾把审计 mixin 放 `model/system/base.go`（`AuditModel`），后上移至 global 以便跨模块复用。

## 设计要点

- `global.OPS_AUDIT_MODEL`（`global/common.go`）：内嵌 `OPS_MODEL`（`CreateTime` autoCreateTime / `UpdateTime` autoUpdateTime / `DeletedAt`）+ `CreateBy`/`UpdateBy`，全 camelCase json，对齐 `CommonRecord`。状态常量 `StatusEnable="0"`/`StatusDisable="1"` 仍在 `model/system/base.go`。
- `SysUser`：`Password json:"-"`、`LoginDate *time.Time` 可空、`UserName uniqueIndex`；保留 `Login` 接口（鉴权链路 uint 暂不动）。
- `SysRole`：`Flag gorm:"-"` 瞬态；超管由初始化流程以显式 `RoleId=1` 种子。
- 关联表复合主键：插入必须显式传两个 ID（雪花回调仅填 0 值主键，不覆盖显式值）。
- **关联建模取舍（不挂 `many2many`）**：多对多用显式 `SysUserRole`/`SysRoleMenu`，理由与读写约定见 `aiDoc/modules/backend-layer-rules.md` →「关联建模」。核心矛盾：雪花显式主键策略 vs `many2many` 隐式写入接管；且授权=批量删插、鉴权=定向 join，都不需要对象图导航。
- request DTO：ID 字段全 `string`/`[]string`（线上 ID 全字符串防精度丢失），service 层转 int64；search 内嵌 `request.PageInfo`。
- AutoMigrate：`SysUser/SysRole/SysUserRole/SysRoleMenu` 进 `RegisterTables()`。

## 暂不动（下一阶段 / 已知技术债）

- service / api / router 三层
- Dept（树形）/ Post 建模
- 超管 `RoleId=1` 种子（初始化流程）
- 鉴权链路 int64/string 改造（`Login` 接口、JWT claims，见 [[snowflake-id-generator]] 同款 TODO）
- `UserName` uniqueIndex + 软删除冲突（后续可加含 `deleted_at` 联合唯一索引）

## 文档同步

- 修订 `aiDoc/modules/backend-layer-rules.md`：`GVA_MODEL` → `OPS_MODEL`，补 `AuditModel` 说明，补「关联建模」规则。 ✅
- 设计文档：`docs/specs/2026-07-12-system-user-role-models-design.md`。
- **基座上移（2026-07-12）**：`OPS_MODEL` 去 `ID` + 时间字段改 `CreateTime/UpdateTime`；新增 `global.OPS_AUDIT_MODEL`；`model/system/base.go` 删 `AuditModel` 仅留状态常量；同步 `backend-layer-rules.md` / `model-example.md` / `boundary.md`。 ✅
