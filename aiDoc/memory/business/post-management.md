# 岗位管理（Post Management）

> 类型：业务模块需求 · 状态：后端接口全套已落地（list/增/改/批量删/optionselect/deptTree），build/vet 通过；菜单+按钮权限种子早已存在；前端页面早就绪

承接 [[dept-post-management]] 的 SysPost model 层（07-13 model + i18n + Elegant 路由）与 [[system-model-rebuild]]（07-17 重建 SysPost/SysUserPost），本文件记录**岗位 CRUD 接口层**（07-18 落地）。实现模式严格对齐 [[dict-management]]（同期落地的四层范式）。

## 需求

系统级「岗位管理」：左侧部门树 + 右侧岗位表，支持岗位的增删改查、下拉选择、按部门过滤。对齐前端 `web/src/service/api/system/post.ts` 6 个接口契约，前端页面（`views/_admin/system/post/`）早就绪。

## 前端契约（反推后端）

`web/src/service/api/system/post.ts`（6 接口，均挂在 `/system/post`）：

- `GET /system/post/list`：分页查岗位。`PostSearchParams = { pageNum, pageSize, postCode, postName, status, belongDeptId }`（+ CommonSearchParams 的 orderByColumn/isAsc/params，后端忽略多余项）。返回 `PostList = PaginatingQueryRecord<Post>`（rows/total/pageNum/pageSize）。
- `POST /system/post`：新增（`PostOperateParams`，postId 空）。
- `PUT /system/post`：修改（`PostOperateParams`，含 postId）。
- `DELETE /system/post/{ids}`：批量删（逗号分隔 ID）。
- `GET /system/post/optionselect`：岗位下拉（params `{ postIds, deptId }`），返回 `Post[]`。供用户管理抽屉分配岗位用。
- `GET /system/post/deptTree`：左侧部门树 + 新增抽屉部门选择，返回 `Api.Common.CommonTreeRecord`（`{ id, parentId, label, weight, children }[]`）。

`Api.System.Post = CommonRecord<{ postId, deptId, postCode, postCategory, postName, postSort, status, remark }>`（`system.api.d.ts:410`）；`status: EnableStatus = '0'|'1'`；`IdType = string | number`。

## 后端（接口全套已落地，07-18）

四层文件 + 三个 enter.go 注册 + PrivateGroup 挂载，`go build ./...` + `go vet ./...` 通过：

- **model**：`SysPost`（`sys_posts`）07-13 已落地，嵌入 `OPS_AUDIT_MODEL` + 雪花主键 `PostId`（json `,string`）。本次在 `model/system/sys_post.go` 末尾新增 `DeptTreeNode`（对齐前端 CommonTreeRecord；**id/parentId/weight 用数字序列化**，对齐前端 `expandedKeys=[100]` 等数字 key——区别于实体主键的 `,string`）。
- **request**（`model/system/request/sys_post.go`，新建）：
  - `PostSearch`（内嵌 `PageInfo` + `postCode/postName/status/deptId/belongDeptId`，`form` tag 适配 GET query；`deptId`/`belongDeptId` 用 `json:",string"` + `form`，query 绑定按 int64 解析）。
  - `PostOperateParams`（`postId/deptId/postCode/postCategory/postName/postSort/status/remark`，create 时 postId 空，deptId 必填）。
- **service**（`service/system/sys_post.go`，新建）：`PostService`
  - `GetPostList`：postCode/postName 模糊、status 精确、`belongDeptId`/`deptId` 任一>0 则按 `dept_id` 精确过滤（belongDeptId 优先；**语义=该部门直接挂载的岗位，不含子部门**——SysPost 为平表无子树概念，符合现有种子数据，子部门联动后续按需扩展）；排序 `post_sort ASC, post_id DESC`；分页走 `LimitOffset`。
  - `CreatePost`：postCode/postName 必填兜底 + **postCode 唯一性校验**（对齐 RuoYi）；审计字段从 claims 注入（struct literal 不可命名内嵌提升字段，改赋值）。
  - `UpdatePost`：postId 必填 + postCode 唯一性排除自身；`Updates` map 写入。
  - `DeletePost`：**sys_user_post 引用校验**——已被用户引用的岗位禁止删（"岗位已分配给用户,不能删除"，对齐 RuoYi）；否则软删。
  - `GetPostOptionList(deptId)`：返回 `status='0'` 启用岗位；deptId>0 限定该部门。
  - `GetDeptTree`：查全部启用部门（`status='0'`，`order_num ASC, dept_id ASC`），`buildDeptTree` 按 parentId **递归**组装树（map[parentId]→[]dept，避免 Go 切片值拷贝陷阱）。
- **api**（`api/v1/system/sys_post.go`，新建）：`PostApi` 6 个 handler + 完整 swag 注释；`@Success` 列表落 `response.PageResult{rows=[]system.SysPost}`、deptTree 落 `[]system.DeptTreeNode`、写操作落 `data=bool`；审计字段走 `utils.GetUserID(c)`，错误日志走 `logger.WithCtx`；批量删 ID 解析用 `strings.SplitSeq`（对齐 [[dict-management]] 先例）。
- **router**（`router/system/sys_post.go`，新建）：`PostRouter.InitPostRouter` 挂 `system/post` group，注册 list/optionselect/deptTree/POST/PUT/DELETE`:ids`。gin 路由树：GET 静态三路径 + DELETE `:ids` 参数路径 + POST/PUT 空路径，HTTP 方法/路径段互不冲突。
- **注册**：`service/system/enter.go` 加 `PostService`；`api/v1/system/enter.go` 加 `PostApi` + `postService` 变量；`router/system/enter.go` 加 `PostRouter` + `postApi` 变量；`initialize/router.go` 的 PrivateGroup 块加 `systemRouter.InitPostRouter(PrivateGroup)`（鉴权 + 操作日志由 PrivateGroup 全局中间件统一处理，子 group 不重复挂）。

## 设计决策

- **deptTree 放岗位 service 内**：部门模块尚未独立实现（无 department service/router），前端契约固定为 `/system/post/deptTree`，故在 `PostService.GetDeptTree` 内查 `sys_departments` 构建树。**待部门模块独立实现后迁出**至部门 service，`DeptTreeNode` 一并迁移。
- **不实现 export**：`post.ts` 无 export 方法（`index.vue` 的 `/system/post/export` 走通用 `useDownload`，后端导出能力项目级未落地，同字典），暂不实现。
- **postCode 唯一 + 删除引用校验**：对齐 RuoYi 岗位语义，避免编码重复与孤儿用户岗位引用。
- **菜单+按钮权限种子早已存在**：`source/system/sys_menu.go` 07-15 已 seed `route.system_post` C 菜单 + 5 个 F 按钮（`system:post:query/add/edit/remove/export`），本次无需动菜单。当前接口仅过 JWT 鉴权即可访问（casbin 接口权限推导项目级未落地，见 [[menu-seed-routes-alignment]]）。

## 相关文件

- 前端：`web/src/service/api/system/post.ts`、`web/src/views/_admin/system/post/`、`web/src/typings/api/system.api.d.ts`
- 后端 model：`server/model/system/sys_post.go`（SysPost + DeptTreeNode）、`server/model/system/sys_user_post.go`（删除引用校验依赖）
- 后端接口四层：`server/model/system/request/sys_post.go`、`server/service/system/sys_post.go`、`server/api/v1/system/sys_post.go`、`server/router/system/sys_post.go`（及三个 `enter.go` 注册、`initialize/router.go` 挂载）
- 关联：[[dept-post-management]]（model 层）、[[dict-management]]（同模式参照）、[[menu-seed-routes-alignment]]（菜单/权限 seed）、[[menu-management]]（菜单 seed 规划参考）
