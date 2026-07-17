# 8 模块 model 重建（删除后端后的重做）

- 日期：2026-07-17
- 状态：Model 层已落地（`go build ./...` / `go vet` 通过）。仅 model 层，未含 request DTO / AutoMigrate 注册 / 种子 / API-Service-Router。
- 关联前端：`web/src/typings/api/system.api.d.ts`（`Api.System.{User,Role,Menu,Dept,Post,DictType,DictData,Notice,Setting}`）

## 背景：历史 model 曾落地但已丢失

User/Role（[[system-user-role-models]]）、Menu（[[menu-management]]）、Dept/Post（[[dept-post-management]]）、Dict（[[dict-management]]）的 model 在 2026-07-12～15 曾落地。但 git `5234503 删除后端，让AI搞的乱套了` + `d8e0306 重新初始化后端` —— 那批 model 在删除后端时被清掉，重新初始化时只恢复了 SysUser/SysRole/SysDepartment/SysPosition 基础架构，Menu/Dict 等未恢复。本次基于「当前现状 + 前端契约」重做。

## 本次重建范围（8 模块 + 系统设置）

**新建 model**：SysMenu(sys_menu)、SysDictType(sys_dict_type)、SysDictData(sys_dict_data)、SysNotice(sys_notice)、SysGeneralConfig(sys_general_config 单行 id=1)、SysPost(sys_posts，替代已删的 SysPosition/sys_position.go)

**重构 model**：
- SysRole：删角色树（ParentId/Children/DefaultRouter），改平表对齐前端 Role；保留 DataScope 作后端数据权限字段
- SysDepartment：业务主键 DeptId(int64) + 独立 phone/email（放弃从 leader 关联带出）；Leader 存 userId
- SysSecurityConfig：json-tag 对齐前端 SecuritySettingConfig（passwordMinLength/loginFailLock\*/ipValidation\*），保留 Captcha\*/Limit\*/PwdExpire\* 供登录链路；新增 IP 校验字段
- SysUser：DeptId/RoleId/GetUserId/GetRoleId 全 int64；关联改 Posts(many2many:sys_user_post)
- SysDataAccessLog：UserID/AuthorityID int64

**关联表（5 张，外键 int64）**：sys_user_role / sys_user_post(新建) / sys_role_menu(新建) / sys_role_departments / sys_user_departments

## 用户拍板的关键决策

1. **主键统一 int64 + 连带修复**：业务主键/外键全 int64（雪花），连带把 `service/system/data_scope.go` + `utils/datascope.Identity` + middleware 全链路 uint→int64。jwt/claims 也一并 int64（BaseClaims.ID/AuthorityId、GetUserID/GetUserAuthorityId）——彻底消除 uint 特例（用户中途打断「用户里是 RoleId，那所有的都改一致」促成）。
2. **部门命名沿用现状 SysDepartment/sys_departments**（非历史 SysDept/sys_dept），data_scope 已依赖，代码自洽。
3. **SysUser 保留只读 many2many 导航**（Roles/Departments/Posts），符合 backend-layer-rules.md 第38行「读取侧可挂只读 many2many」；写入侧仍走显式关联表。
4. **ID json `,string`**：所有 ID 字段对齐前端 IdType(string)（Menu.ParentId/Post.DeptId/Dept.ParentId/User.DeptId 等补齐）；Dept.Leader 保持 number（前端 Dept.leader: number）。
5. **status 统一 string**（'0'正常/'1'停用），改掉 SysDepartment/SysPosition 的 *bool。

## 与历史决策的差异（已知）

- 部门命名：现状 SysDepartment vs 历史 SysDept → 用户选沿用现状
- many2many：保留只读 vs 历史 user-role 记忆「不挂」→ 用户选保留只读（规则第38行允许）
- 历史还做了 request DTO + AutoMigrate 注册，本次范围仅 model 层未做

## 待办（后续最小闭环）

- [ ] gorm_biz.go / gorm.go 的 AutoMigrate 注册新 model（当前 RegisterTables 仅 SysUser/JwtBlacklist/SysRole/SysError）
- [ ] request DTO（SysMenuSearch/Req 等，ID 字段 string/[]string）
- [ ] source 种子数据（初始 admin/角色/菜单树/字典）
- [ ] API/Service/Router 业务链路

## 相关文件

- 新建：`server/model/system/{sys_menu,sys_dict_type,sys_dict_data,sys_notice,sys_general_config,sys_post,sys_user_post,sys_role_menu}.go`
- 重构：`server/model/system/{sys_role,sys_department,sys_security_config,sys_user,sys_data_access_log}.go` + 关联表 3 张（sys_user_role/sys_role_department/sys_user_department 外键 int64）
- 连带 int64：`model/system/request/jwt.go`、`utils/claims.go`、`utils/datascope/datascope.go`、`service/system/data_scope.go`、`middleware/{data_scope,access_log}.go`
- 删除：`server/model/system/sys_position.go`
- 关联：[[system-user-role-models]] [[menu-management]] [[dept-post-management]] [[dict-management]] [[snowflake-id-generator]] [[system-init-flow]]
