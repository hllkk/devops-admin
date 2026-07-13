# 菜单管理（Menu Management）

> 类型：业务模块需求 · 状态：进行中（前端页面+后端模型已就绪，后端 Service/API/Router 待补）

## 需求

为系统增加「菜单管理」能力，支撑后台菜单/权限的可视化维护：树形展示目录-菜单-按钮三级结构，
支持新增/编辑/删除、级联删除、按钮权限（F 类型）维护，以及角色分配菜单时的菜单树选择。

前端由用户先行接入：新增路由页面 `web/src/views/_admin/system/menu/`，并补齐 service API、
typings（`Api.System.Menu` 全套）、业务常量（menuType / isFrame / iconType / layout 字典）、
i18n key 类型声明（`app.d.ts`）。

## 已实现（本次）

### 后端模型（`server/model/system/`）
- `sys_menu.go`：`SysMenu` 结构，对齐前端 `Api.System.Menu`。
  - 雪花主键 `menu_id`（`json:"menuId,string"`），树形 `parent_id`（根=0，`json:"parentId,string"`）。
  - 字段：`menu_name / order_num / path / component / query_param / is_frame / is_cache / menu_type / visible / status / perms / icon / remark`。
  - 状态字典：`menu_type` M目录/C菜单/F按钮；`is_frame` 0外链/1内部/2iframe；`is_cache` 0缓存/1不缓存；`visible` 0显示/1隐藏；`status` 0正常/1停用。
  - 审计基座 `global.OPS_AUDIT_MODEL`（createBy/updateBy + 时间戳）。
  - 瞬态字段 `ParentName`、`Children`（`gorm:"-"`），供树构建 / treeselect 使用。
  - 表名 `sys_menu`，已在 `server/initialize/gorm.go` 的 `RegisterTables` 注册（实体先于关联表：User/Role/Menu → UserRole/RoleMenu）。
- `request/sys_menu.go`：
  - `SysMenuSearch`：列表查询（`menuName/status/menuType/parentId` + 排序），**不内嵌 PageInfo**——菜单为树形、列表接口返回扁平数组、前端用 `handleTree` 构树。
  - `SysMenuReq`：新增/修改（`menuId/parentId` 用 `*string` 指针区分新增与修改，对齐 `RecordNullable`）。

### 前端国际化
- `web/src/locales/langs/{zh-cn,en-us}.ts`：在 `page.system` 下、`role` 与 `dept` 之间补齐完整 `menu` 文案块，
  覆盖 `app.d.ts` 声明的全部 53 个 key（含 `placeholder.*` 5 项、`form.*` 7 项 FormMsg）。
  中/英 key 集合与声明完全一致（结构化校验通过）。

## 契约要点（对齐 `boundary.md`）

- 菜单列表接口 `GET /system/menu/list` 返回扁平 `Menu[]`（非分页），前端自行构树。
- ID 一律字符串传输（`json:"...,string"`），前端禁当 number 运算。
- 关联表 `SysRoleMenu`（角色-菜单）已存在，给角色分配菜单走「删后批量插」。
- 树/树选择相关响应（`treeselect`、`roleMenuTreeselect`、`cascade` 删除）待 Service/API 实现时定义。

## 待办（后续）

- [ ] 后端 `service/system/sys_menu.go`：列表/树/增删改/级联删除/treeselect/roleMenuTreeselect。
- [ ] 后端 `api/v1/system/sys_menu.go` + `router/system/sys_menu.go`：对齐前端 7 个接口。
- [ ] `menu.go` 初始化种子：默认菜单与权限（`system:menu:list/add/edit/remove`）。
- [ ] Swagger 注释与统一响应 `{ code, data, msg }`。

## 相关文件

- 前端：`web/src/views/_admin/system/menu/`、`web/src/service/api/system/menu.ts`、`web/src/typings/api/system.api.d.ts`、`web/src/locales/langs/{zh-cn,en-us}.ts`、`web/src/constants/business.ts`
- 后端：`server/model/system/sys_menu.go`、`server/model/system/request/sys_menu.go`、`server/initialize/gorm.go`
- 关联：`server/model/system/sys_role_menu.go`
