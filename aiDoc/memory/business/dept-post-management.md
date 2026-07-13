# 部门与岗位管理（Dept & Post Management）

> 类型：业务模块需求 · 状态：进行中（前端页面+后端模型+i18n 已就绪，后端 Service/API/Router 待补）

## 需求

为系统增加「部门管理」「岗位管理」能力，支撑组织架构与岗位维护：
- 部门：树形结构（parentId + ancestors 祖级链），支持新增/编辑/删除、负责人选择（部门下用户）、状态切换；列表返回扁平数组前端构树。
- 岗位：归属部门（deptId），分页列表 + 左侧部门树下钻过滤（belongDeptId）；支持新增/编辑/批量删除/导出。

前端由用户先行接入：新增路由页面 `web/src/views/_admin/system/{dept,post}/`，补齐 service API（`dept.ts`/`post.ts`）、
typings（`Api.System.Dept`/`Post` 全套）、`app.d.ts` 的 i18n Schema 声明。

## 已实现（本次）

### 后端模型（`server/model/system/`）
- `sys_dept.go`：`SysDept` 结构，对齐前端 `Api.System.Dept`。
  - 雪花主键 `dept_id`（`json:"deptId,string"`），树形 `parent_id`（根=0，`json:"parentId,string"`），`ancestors` 祖级链（如 `0,100,101`，由 Service 依据父链推导，不暴露给前端）。
  - 字段：`dept_name / dept_category / order_num / leader / phone / email / status`。
  - `leader` 为负责人 userId，按主键契约以字符串传输（`json:"leader,string"`，0=未指定）。**注意**：前端 `Dept.leader` 残留为 `number` 类型，与全栈「ID 一律字符串」契约有轻微出入，运行时无碍，后续可调整为 `CommonType.IdType`。
  - 瞬态 `Children`（`gorm:"-"`）供树构建使用；审计基座 `global.OPS_AUDIT_MODEL`。
  - 表名 `sys_dept`，已在 `RegisterTables` 注册。
- `sys_post.go`：`SysPost` 结构，对齐前端 `Api.System.Post`。
  - 雪花主键 `post_id`（`json:"postId,string"`），归属 `dept_id`（`json:"deptId,string"`）。
  - 字段：`post_code / post_category / post_name / post_sort / status / remark`。
  - 审计基座 `global.OPS_AUDIT_MODEL`；表名 `sys_post`，已注册。
- **多租户残留**：前端 `Dept`/`Post` 类型含 `tenantId`，多租户已于 2026-07-12 清理（见 `remove-multi-tenant.md`），后端不建模 `tenantId`，与 `SysUser`/`SysRole` 一致。
- `request/sys_dept.go`：`SysDeptSearch`（树形→不内嵌 PageInfo，同 `SysMenuSearch`）+ `SysDeptReq`（`deptId/parentId/leader` 用 `*string` 指针区分新增/修改）。
- `request/sys_post.go`：`SysPostSearch`（分页→内嵌 `request.PageInfo`；含 `belongDeptId` 树下钻过滤）+ `SysPostReq`（`postId/deptId` 用 `*string` 指针）。

### 前端国际化
- `app.d.ts`：新增 `page.system.post` Schema（title/listTitle/deptTreeTitle/emptyDept/exportFileName + 字段标签 + `form.*` FormMsg）。
- `zh-cn.ts`/`en-us.ts`：
  - 填全 `page.system.dept` 全量 key（此前仅有 `title`/`empty`，dept 页面其余文案为裸 key）。
  - 新增 `page.system.post` 全量中/英文案。
  - 新增路由文案 `route.system_dept`/`system_post`（顺带补 `system_menu`/`system_dict`/`system_notice`，见下）。
- **post 页面去硬编码**：`post/index.vue`、`post-operate-drawer.vue`、`post-search.vue` 原为硬编码中文，统一改 `$t('page.system.post.*')`，沿用 dept 约定（`form.X.required`=占位提示、`form.X.invalid`=必填校验文案）。

### Elegant 路由
- `views/_admin/system/{dept,post}` 经 `_admin` 根挂载约定生成 `system_dept`/`system_post` 路由（`/system/dept`、`/system/post`）。
- 通过启动 vite 触发 Elegant 插件重生成 `src/router/elegant/{routes,imports,transform}.ts` 与 `typings/elegant-router.d.ts`（此前路由表陈旧，仅含 `system_role`/`system_user`，未含 menu/dict/dept/post/notice）。
- 重生成一并产出 `system_menu`/`system_dict`/`system_notice`，故补齐对应 5 个路由文案以保证 `Record<I18nRouteKey, string>` 类型闭合。

## 契约要点（对齐 `boundary.md`）

- 部门列表 `GET /system/dept/list` 返回扁平 `Dept[]`（非分页），前端 `handleTree` 构树。
- 岗位列表 `GET /system/post/list` 分页，返回 `{ pageNum, pageSize, total, rows }`。
- ID 一律字符串传输（`json:"...,string"`），前端禁当 number 运算。
- 删除均为路径拼接批量：`DELETE /system/dept/{ids}`、`DELETE /system/post/{ids}`（逗号分隔）。

## 待办（后续，同 menu 模块节奏）

- [ ] 后端 `service/system/sys_dept.go`/`sys_post.go`：列表/树/增删改/排除树/deptTree/optionselect。
- [ ] 后端 `api/v1/system/sys_dept.go`/`sys_post.go` + `router/system/*.go`：对齐前端接口（dept 6 个、post 6 个）。
- [ ] `sys_user_post` 关联表（用户-岗位多对多）建表与授权。
- [ ] `dept.go`/`post.go` 初始化种子：默认部门/岗位 + 菜单与权限（`system:dept:list/add/edit/remove`、`system:post:list/add/edit/remove/export`）。
- [ ] Swagger 注释与统一响应 `{ code, data, msg }`。

## 相关文件

- 前端：`web/src/views/_admin/system/{dept,post}/`、`web/src/service/api/system/{dept,post}.ts`、`web/src/typings/api/system.api.d.ts`、`web/src/locales/langs/{zh-cn,en-us}.ts`、`web/src/typings/app.d.ts`、`web/src/router/elegant/*`（重生成）
- 后端：`server/model/system/sys_dept.go`、`server/model/system/sys_post.go`、`server/model/system/request/sys_{dept,post}.go`、`server/initialize/gorm.go`
- 关联：`server/model/system/sys_menu.go`（同批建模范式）、`aiDoc/memory/business/menu-management.md`、`aiDoc/memory/business/remove-multi-tenant.md`
