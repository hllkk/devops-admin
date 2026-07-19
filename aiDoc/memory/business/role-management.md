# 角色管理（Role Management）

> 类型：业务模块需求 · 状态：后端接口全套已落地（八接口），build/vet/路由注册测试通过；菜单+按钮权限种子早已存在；前端页面早就绪

承接 [[system-user-role-models]] 的 SysRole model 层（07-12）与 [[system-model-rebuild]]（07-17 重建 SysRole/SysUserRole/SysRoleMenu/SysRoleDepartment），本文件记录**角色 CRUD 接口层**（07-19 落地）。实现模式对齐 [[dict-management]] / [[post-management]] / [[department-management]] / [[menu-management]]。

## 需求

系统级「角色管理」：角色的增删改、状态切换、**分配菜单**（含回显）、**用户授权**（分配/取消用户）。对齐前端 `web/src/service/api/system/role.ts` 8 个接口契约。

## 角色分配菜单回显链路（用户重点关注）

完整链路两端对称，回显已由菜单模块支持、保存本轮落地：

- **回显（读）**：编辑角色 → `GET /system/menu/roleMenuTreeselect/{roleId}`（[[menu-management]] 已实现）→ `{menus: 全量平表, checkedKeys: 角色已分配菜单的叶子 ID}` → drawer `model.menuIds = checkedKeys`，NTree cascade 自动推导父级半选。
- **保存（写）**：drawer 提交 → `POST/PUT /system/role`（`RoleOperateParams.menuIds`）→ 后端事务全量替换 `sys_role_menu`。前端 `MenuTree.getCheckedMenuIds` 在 cascade 模式返回「叶子 + 半选父级」= 应保存全集，与回显的叶子集对称。

## 前端契约（反推后端）

`web/src/service/api/system/role.ts`（8 接口，挂在 `/system/role`）：

- `GET /system/role/list`：分页角色（`roleName/roleKey/status`），返回 `RoleList`（分页）。
- `POST /system/role`：新增（`RoleOperateParams` 含 `menuIds`）。
- `PUT /system/role`：修改（含 `menuIds`，全量替换）。
- `PUT /system/role/changeStatus`：改状态（`roleId + status`）。
- `DELETE /system/role/{ids}`：批量删。
- `GET /system/role/authUser/allocatedList`：角色已分配用户分页（query `roleId/userName/phonenumber`）。
- `PUT /system/role/authUser/selectAll`：批量授权用户（query `roleId + userIds逗号`）。
- `PUT /system/role/authUser/cancelAll`：批量取消授权（query `roleId + userIds逗号`）。

`RoleOperateParams` 含 `roleId/roleName/roleKey/roleSort/menuCheckStrictly/status/remark/menuIds`，**不含** `superAdmin/dataScope`（前端不消费，走保留值/默认）。

## 后端（接口全套已落地，07-19）

四层文件 + 三个 enter.go 注册 + PrivateGroup 挂载，`go build` + `go vet` + 路由注册测试通过：

- **request**（`model/system/request/sys_role.go`，新建）：
  - `RoleSearch`（分页 roleName/roleKey/status）。
  - `RoleOperateParams`（roleId/roleName/roleKey/roleSort/menuCheckStrictly/status/remark + `menuIds []Int64String`——兼容前端 IdType[] 混合）。
  - `RoleUserSearch`（roleId + userName/phonenumber + PageInfo）。
  - `RoleAuthUserParams`（roleId + userIds 逗号分隔，query 传参）。
- **service**（`service/system/sys_role.go`，新建）：`RoleService`
  - `GetRoleList`：roleName/roleKey 模糊、status 精确，`role_sort ASC, role_id ASC`，`LimitOffset` 分页。
  - `CreateRole`：**事务**（roleKey 唯一校验 → 建角色 → `saveRoleMenus` 批量插 sys_role_menu）。
  - `UpdateRole`：**事务**（roleKey 唯一排除自身 → 更新角色 → 全量替换 sys_role_menu 删后插）；不更新 superAdmin/dataScope。
  - `UpdateRoleStatus`：只更 status + update_by。
  - `DeleteRole`：`sys_user_role` 引用校验（有用户禁删）→ 事务清理 sys_role_menu/sys_role_departments → 删角色。
  - `GetAllocatedUserList`：`JOIN sys_user_role ON sys_user_id=sys_users.id WHERE sys_role_id=?`，userName/phonenumber 模糊，分页。
  - `AuthUserSelectAll`：去重已有后批量插 sys_user_role。
  - `AuthUserCancelAll`：删 sys_user_role where role+user。
  - `saveRoleMenus`/`toInt64Slice` 辅助。
- **api**（`api/v1/system/sys_role.go`，新建）：`RoleApi` 8 handler + swag 注释；审计字段走 `utils.GetUserID(c)`；`parseUserIds` 解析逗号分隔。
- **router**（`router/system/sys_role.go`，新建）：`RoleRouter.InitRoleRouter` 挂 `system/role` group，注册 list / authUser/allocatedList / POST / PUT / PUT changeStatus / PUT authUser/selectAll / PUT authUser/cancelAll / DELETE`:ids`。
- **注册**：三个 `enter.go` 加 `RoleService`/`RoleApi`+`roleService`/`RoleRouter`+`roleApi`；`initialize/router.go` PrivateGroup 加 `InitRoleRouter`。

## 设计决策

- **分配菜单 = 全量替换**：create/update 在事务里 `DELETE sys_role_menu WHERE role_id` + 批量 INSERT（对齐 RuoYi，删后插，便于对 SQL 日志/调优）；与 menu 的 `roleMenuTreeselect`（叶子回显）+ 前端 `getCheckedMenuIds`（叶子+半选父级保存）对称。
- **事务**：CreateRole/UpdateRole/DeleteRole 多表操作用 `Transaction`，保证角色与关联表一致。
- **menuIds 用 []Int64String**：前端 `getCheckedMenuIds` 返回 `string[]`，普通 `[]int64` 绑定 `["100"]` 会失败，复用 [[department-management]] 的 `Int64String` 兼容。
- **roleKey 唯一**：对齐 RuoYi（角色权限字符唯一），create 校验、update 排除自身。
- **不更新 superAdmin/dataScope**：前端 `RoleOperateParams` 未含；superAdmin 走默认 false，dataScope 走默认 1（数据权限档位由数据权限模块另行管理，不在角色 drawer）。
- **路由注册**：PUT `""`(root) 与 `changeStatus`/`authUser/*`(static) 同节点共存、DELETE `:ids` 与 GET/PUT static 同 group 共存，经 `router/system/sys_role_test.go` 的 `TestRoleRouterRegistration` 验证不 panic（保留作回归防护；同文件 `TestMenuRouterRegistration` 覆盖菜单）。
- **菜单+按钮权限种子早已存在**：`source/system/sys_menu.go` 已 seed `route.system_role` C 菜单 + 5 个 F 按钮（`system:role:query/add/edit/remove/export`），无需动菜单。

## 相关文件

- 前端：`web/src/service/api/system/role.ts`、`web/src/views/_admin/system/role/`（operate-drawer 含分配菜单、auth-user-drawer）、`web/src/components/custom/menu-tree.vue`（`getCheckedMenuIds` 保存语义）、`web/src/typings/api/system.api.d.ts`
- 后端 model：`server/model/system/sys_role.go`、`sys_role_menu.go`（分配菜单）、`sys_user_role.go`（用户授权/删除引用）、`sys_role_department.go`（删除清理）
- 后端接口四层：`server/model/system/request/sys_role.go`、`server/service/system/sys_role.go`、`server/api/v1/system/sys_role.go`、`server/router/system/sys_role.go` + `sys_role_test.go`（及三个 `enter.go`、`initialize/router.go`）
- 关联：[[system-user-role-models]]（model 层）、[[menu-management]]（roleMenuTreeselect 回显 + 菜单路由测试同文件）、[[department-management]]（Int64String 复用）、[[dict-management]]/[[post-management]]（同模式参照）
