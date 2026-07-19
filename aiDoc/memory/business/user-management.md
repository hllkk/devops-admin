# 用户管理（User Management）

> 类型：业务模块需求 · 状态：管理员侧接口已落地（9 接口），build/vet/路由注册测试通过；个人中心 profile/updatePwd/avatar 待续；前端页面早就绪

承接 [[system-user-role-models]] 的 SysUser model（07-12）与 [[system-model-rebuild]]（07-17 重建 SysUser/SysUserRole/SysUserPost/SysUserDepartment），本文件记录**用户管理接口层**（07-19 落地）。实现模式对齐 [[dict-management]]/[[post-management]]/[[department-management]]/[[menu-management]]/[[role-management]]。

## 需求

系统级「用户管理」：用户的增删改、状态切换、重置密码、**分配角色/岗位**、按部门/角色过滤、部门树筛选。对齐前端 `web/src/service/api/system/user.ts`。

## 范围（本轮）

**管理员侧 9 接口已落地**（`/system/user/*`）：
1. `GET /system/user/list` 分页（deptId/userName/nickName/phonenumber/status/roleId 过滤）
2. `GET /system/user/list/dept/{deptId}` 部门下用户（**部门负责人选择**，补之前欠部门模块的依赖）
3. `GET /system/user/{userId}` 详情（UserInfo: postIds/roleIds/roles）
4. `GET /system/user/deptTree` 部门树（复用 [[department-management]] 的 `DepartmentService.GetDeptTree`）
5. `POST /system/user` 新增（含 roleIds/postIds + bcrypt 密码 + UUID）
6. `PUT /system/user` 修改（全量替换角色/岗位；password 空=不改）
7. `PUT /system/user/changeStatus` 改状态
8. `PUT /system/user/resetPwd` 重置密码
9. `DELETE /system/user/{userIds}` 批量删（清理三关联表）

**个人中心 3 接口待续**（自助，涉及文件上传/加密验证，单独功能点）：`PUT /system/user/profile`（改基本信息）、`PUT /system/user/profile/updatePwd`（改自己密码，验证旧密码）、`POST /system/user/profile/avatar`（头像 FormData 上传）。

## 后端实现（07-19）

四层文件 + 两个 enter.go 注册 + PrivateGroup 挂载，`go build`/`go vet`/路由注册测试通过：

- **request**（`model/system/request/sys_user.go`，追加）：`UserSearch`（分页 + deptId/userName/nickName/phonenumber/status/roleId）、`UserOperateParams`（userId/deptId/userName/nickName/email/phonenumber/sex/password/status/remark + `roleIds/postIds []Int64String`）、`ResetUserPwdParams`（userId + password）。
- **model DTO**（`model/system/sys_user.go`，追加）：`UserInfo{PostIds []string, RoleIds []string, Roles []SysRole}`——postIds/roleIds 用 **[]string** 对齐前端 `string[]`（NSelect/PostSelect 回显需与 `Role.roleId` 字符串匹配）。
- **service**（`service/system/sys_user_manage.go`，新建；auth 链路 Login/Register/GetUserInfo 仍在 `sys_user.go`）：`UserService` 扩展
  - `GetList`：多字段过滤，roleId>0 时 `JOIN sys_user_role`；`user_id ASC`。
  - `GetDeptUserList(deptId)`：部门启用用户（负责人选择）。
  - `Create`：**事务**（用户名唯一校验 → 建用户[bcrypt + `uuid.New()` + PasswordUpdatedAt] → `saveUserRoles`/`saveUserPosts`）。
  - `Update`：**事务**（Updates map + password 空=不改 + 全量替换角色/岗位；PasswordUpdatedAt 指针字段单独 Update）。
  - `UpdateStatus`/`Delete`（清理 sys_user_role/sys_user_post/sys_user_departments）/`ResetPwd`（bcrypt）。
  - `GetDetail`：Preload Roles + Pluck roleIds/postIds → `int64ToStrSlice` 转 string[]。
  - `saveUserRoles`/`saveUserPosts`/`int64ToStrSlice` 辅助（`toInt64Slice` 复用 [[role-management]]）。
- **api**（`api/v1/system/sys_user_manage.go`，新建）：`UserApi` 9 handler + swag 注释；`GetDeptTree` 调 `departmentService`；审计字段走 `utils.GetUserID(c)`。`BaseApi`（Register/GetUserInfo，auth/initdb 用）仍在 `sys_user.go`。
- **router**（`router/system/sys_user.go`，追加 group）：`InitUserRouter` 末尾加 `system/user` group（9 路由，GET `:userId` 放 static 之后兜底）；旧 `/user/*` 块保留（baseApi，前端不用，冗余无害）。
- **注册**：`api/v1/system/enter.go` 的 `ApiGroup` 加 `UserApi`；`router/system/enter.go` 加 `userApi` 变量；`initialize/router.go` 启用 `systemRouter.InitUserRouter(PrivateGroup)`（旧块此前被注释，本轮启用连带挂 `/system/user/*`）。

## 设计决策

- **分配角色/岗位 = 全量替换**：create/update 在事务里删后批量插 `sys_user_role`/`sys_user_post`（对齐 [[role-management]] 的菜单分配模式）；前端 operate params 只含主部门 deptId（无 deptIds 多部门），多部门归属走数据权限另配。
- **密码无加密中间件**：探测确认项目无 RSA/SM2 解密（前端 `isEncrypt` header 是标记，后端无对应解密）→ `resetPwd`/未来 `updatePwd` 直接收明文 bcrypt 存储（传输层由网关/TLS 保护）。
- **UUID + PasswordUpdatedAt**：create 时 `uuid.New()`（登录链路 claims 用）、设 PasswordUpdatedAt（密码过期判定用）；现有 `Register`（initdb 向导）未设 UUID，正常 create 补齐。
- **UserInfo 字符串数组**：前端 `UserInfo.postIds/roleIds: string[]`，`SysRole.roleId` 因 `json:",string"` 是字符串，故后端 Pluck 后 `int64ToStrSlice` 转字符串，保证 NSelect 回显匹配。
- **deptTree 复用**：用户页左侧部门树与岗位页同源，直接调 `DepartmentService.GetDeptTree`（[]DeptTreeNode），不重复实现。
- **路由注册**：GET `:userId`(param) 与 `list`/`deptTree`(static) 同层、`list`(static handler) 下挂 `dept/:deptId`，经 `router/system/sys_user_test.go` 的 `TestUserRouterRegistration` 验证不 panic（同文件含 menu/role 测试）。
- **菜单+按钮权限种子早已存在**：`source/system/sys_menu.go` 已 seed `route.system_user` C 菜单 + 7 个 F 按钮（`system:user:query/add/edit/remove/export/import/resetPwd`）。

## 相关文件

- 前端：`web/src/service/api/system/user.ts`、`web/src/views/_admin/system/user/`（operate-drawer 分配角色/岗位、password-drawer、import-modal）、`web/src/typings/api/system.api.d.ts`
- 后端 model：`server/model/system/sys_user.go`（SysUser + UserInfo）、`request/sys_user.go`、`sys_user_role.go`/`sys_user_post.go`/`sys_user_department.go`（分配/删除依赖）
- 后端接口四层：`server/service/system/sys_user_manage.go`、`server/api/v1/system/sys_user_manage.go`、`server/router/system/sys_user.go`（+ `sys_user_test.go`）、三个 `enter.go` + `initialize/router.go`
- 关联：[[system-user-role-models]]（model）、[[role-management]]（分配角色/toInt64Slice 复用）、[[department-management]]（deptTree 复用、Int64String 复用）、[[menu-management]]（路由测试同文件）
