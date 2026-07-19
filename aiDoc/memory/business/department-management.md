# 部门管理（Department Management）

> 类型：业务模块需求 · 状态：后端接口全套已落地（list/exclude/增/改/批量删/optionselect），build/vet 通过；菜单+按钮权限种子早已存在；前端页面早就绪

承接 [[dept-post-management]] 的 SysDepartment model 层（07-13 model + i18n + Elegant 路由）与 [[system-model-rebuild]]（07-17 重建 SysDepartment/SysRoleDepartment/SysUserDepartment），本文件记录**部门 CRUD 接口层**（07-18 落地）。实现模式对齐 [[dict-management]] / [[post-management]]（同期落地的四层范式）。同时把上一轮临时放在岗位 service 的部门树构建迁回部门模块（见 [[post-management]] 的 deptTree 迁移）。

## 需求

系统级「部门管理」：树形部门表，支持部门的增删改查、按名称/状态过滤、选父级（排除自身及子树防环）、下拉选择。对齐前端 `web/src/service/api/system/dept.ts` 5 个接口契约，前端页面（`views/_admin/system/dept/`）早就绪。

## 前端契约（反推后端）

`web/src/service/api/system/dept.ts`（5 接口，挂在 `/system/dept`）：

- `GET /system/dept/list`：部门列表。`DeptSearchParams = { deptName, status }`（+ CommonSearchParams）。**不分页**，返回平表 `Dept[]`，前端 `useNaiveTreeTable` + `treeTransform(response,{idField:'deptId'})` 组装树。
- `GET /system/dept/list/exclude/{deptId}`：排除 deptId 及其子部门的平表（编辑时选父级，前端 `handleTree` 组装）。
- `POST /system/dept`：新增（`DeptOperateParams`，deptId 空）。
- `PUT /system/dept`：修改（`DeptOperateParams`，含 deptId）。
- `DELETE /system/dept/{ids}`：批量删（逗号分隔）。
- `GET /system/dept/optionselect`：部门下拉（平表 `Dept[]`）。

`Api.System.Dept = CommonRecord<{ deptId, parentId, ancestors, deptName, deptCategory, orderNum, leader, phone, email, status, children }>`（`system.api.d.ts:368`）；`DeptOperateParams` 不含 ancestors（后端维护）；`status: EnableStatus = '0'|'1'`。drawer 选负责人另调 `fetchGetDeptUserList`（`/system/user/list/dept/{deptId}`，归用户模块，本轮不做）。

## 后端（接口全套已落地，07-18）

四层文件 + 三个 enter.go 注册 + PrivateGroup 挂载，`go build ./...` + `go vet ./...` 通过：

- **model**：`SysDepartment`（`sys_departments`）早已落地，嵌入 `OPS_AUDIT_MODEL` + 雪花主键 `DeptId`，含 `ParentId/Ancestors/OrderNum/Status/Leader/Phone/Email`，`Children/NamePath` 为 `gorm:"-"` 内存组装。本次在 `model/system/sys_department.go` 末尾落 `DeptTreeNode`（**从 `sys_post.go` 迁来**，对齐前端 CommonTreeRecord；id/parentId/weight 数字序列化对齐前端数字 key）。
- **request**（`model/system/request/sys_dept.go`，新建）：
  - `Int64String`（自定义 int64 + `UnmarshalJSON`，容忍 `""`/`null`/数字/`"数字串"` → 解决前端顶层新增部门 `parentId=''` 绑定失败）。
  - `DeptSearch`（内嵌 `PageInfo` 仅兼容前端分页参数、不用；`deptName`/`status`）。
  - `DeptOperateParams`（`deptId`/`parentId` 用 `Int64String`；`orderNum`/`leader`(int64,null→0)/`phone`/`email`/`status` 等；无 ancestors）。
- **service**（`service/system/sys_department.go`，新建）：`DepartmentService`
  - `GetDeptList`：不分页平表，deptName 模糊/status 精确，`order_num ASC, dept_id ASC`。
  - `GetExcludeDeptList(deptId)`：`fullChain = ancestors+","+deptId`，排除 `dept_id=deptId` 与 `ancestors=fullChain OR LIKE fullChain+",%"`（自身+全部子孙）；deptId 无效返回全部。
  - `CreateDept`：ancestors = 父.ancestors+","+父.deptId（顶层 parentId<=0 → "0"）；`checkDeptNameUnique` 同父级名称唯一；审计字段赋值。
  - `UpdateDept`：parentId 必填、不可为自身/子孙（`isDescendant` 防环兜底）；`checkDeptNameUnique` 排除自身；parentId 变更时**同步子孙 ancestors**（旧完整链前缀→新完整链前缀，`c.Ancestors[len(oldFullChain):]` 拼接，对齐 RuoYi updateDept）。
  - `DeleteDept`：**四重引用校验**——子部门 / 用户主部门(SysUser.dept_id) / 用户多部门(sys_user_departments) / 岗位(SysPost.dept_id) / 角色数据权限(sys_role_departments)，任一占用即禁删；`assertNoRef` 通用计数。
  - `GetDeptOptionList`：全量平表。
  - `GetDeptTree`（**迁自岗位**）：查启用部门 `buildDeptTree` 递归组装 `[]DeptTreeNode`。
- **api**（`api/v1/system/sys_department.go`，新建）：`DeptApi` 6 handler + swag 注释；`@Success` 落 `[]system.SysDepartment`/`data=bool`；审计字段走 `utils.GetUserID(c)`；批量删 ID 解析用 `strings.SplitSeq`。
- **router**（`router/system/sys_department.go`，新建）：`DeptRouter.InitDeptRouter` 挂 `system/dept` group，注册 list / list/exclude/:deptId / optionselect / POST / PUT / DELETE`:ids`。
- **注册**：`service/system/enter.go` 加 `DepartmentService`；`api/v1/system/enter.go` 加 `DeptApi` + `departmentService` 变量；`router/system/enter.go` 加 `DeptRouter` + `deptApi`；`initialize/router.go` PrivateGroup 块加 `systemRouter.InitDeptRouter(PrivateGroup)`。
- **岗位 deptTree 迁移**：`service/system/sys_post.go` 删除 `GetDeptTree`/`buildDeptTree`，`api/v1/system/sys_post.go` 的 `GetPostDeptTree` 改调 `departmentService.GetDeptTree`（`DeptTreeNode` 随之归 `sys_department.go`）。岗位 `/system/post/deptTree` 行为不变，逻辑归部门模块。

## 设计决策

- **ancestors 维护**：树形部门的核心。新增按父级拼接；修改 parentId 变更时遍历子孙做前缀替换（直接子 ancestors=旧完整链，更深子以其+",..."开头，统一 `newFullChain + tail`）。`isDescendant` 用 `ancestors` 链含 deptId 段判子孙，做防环兜底（前端 exclude 已过滤）。
- **Int64String**：仅 dept 的 `parentId`/`deptId` 需要（顶层新增传空串）；leader 用普通 int64（前端 null→0 不报错）。
- **list 不分页**：部门是树，前端组装，后端返全量平表（含 status 过滤）。
- **菜单+按钮权限种子早已存在**：`source/system/sys_menu.go` 07-15 已 seed `route.system_dept` C 菜单 + 4 个 F 按钮（`system:dept:query/add/edit/remove`），本次无需动菜单。接口仅过 JWT 鉴权（casbin 未落地，见 [[menu-seed-routes-alignment]]）。
- **fetchGetDeptUserList 不在本轮**：drawer 选负责人依赖它，但路径 `/system/user/list/dept/{deptId}` 归用户模块、返回 User 列表，且 leader 非必填，不阻塞部门主流程，留待用户模块。

## 相关文件

- 前端：`web/src/service/api/system/dept.ts`、`web/src/views/_admin/system/dept/`、`web/src/typings/api/system.api.d.ts`
- 后端 model：`server/model/system/sys_department.go`（SysDepartment + DeptTreeNode）、`sys_user_department.go`/`sys_role_department.go`（删除引用校验依赖）
- 后端接口四层：`server/model/system/request/sys_dept.go`、`server/service/system/sys_department.go`、`server/api/v1/system/sys_department.go`、`server/router/system/sys_department.go`（及三个 `enter.go` 注册、`initialize/router.go` 挂载）
- 迁移影响：`server/service/system/sys_post.go`、`server/api/v1/system/sys_post.go`（deptTree 改调部门 service）
- 关联：[[dept-post-management]]（model 层）、[[post-management]]（deptTree 迁出方）、[[dict-management]]（同模式参照）、[[menu-seed-routes-alignment]]（菜单/权限 seed）
