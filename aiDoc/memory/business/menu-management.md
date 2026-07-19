# 菜单管理（Menu Management）

> 类型：业务模块需求 · 状态：后端全套已落地（model + i18n + 七接口），build/vet/路由注册测试通过；菜单+按钮权限种子早已存在；前端页面早就绪

## 需求

系统级「菜单管理」：树形维护目录(M)-菜单(C)-按钮(F) 三级结构，支持增删改、级联删除、按钮权限维护，以及角色分配菜单时的菜单树选择。对齐前端 `web/src/service/api/system/menu.ts` 7 个接口契约。

## 后端模型（07-12 建模 / 07-17 重建，已落地）

- `SysMenu`（`sys_menu`）：嵌入 `OPS_AUDIT_MODEL` + 雪花主键 `MenuId`；字段 `ParentId/MenuType/MenuName/OrderNum/Path/Component/QueryParam/IsFrame/IsCache/Visible/Status/Perms/Icon/Remark`；`ParentName/Children` 为 `gorm:"-"` 内存组装。状态字典：menuType M/C/F、isFrame 0外链/1内部/2iframe、isCache 0缓存/1不缓存、visible 0显示/1隐藏、status 0正常/1停用。
- 关联表 `SysRoleMenu`（`sys_role_menu`，复合主键 sys_role_id+sys_menu_id）。
- 已注册 `RegisterTables`；菜单+按钮权限种子见 `source/system/sys_menu.go`（07-15 seed `route.system_*` C 菜单 + 各 F 按钮）。

## 后端接口（07-18 落地，build/vet/路由注册测试通过）

四层文件 + 三个 enter.go 注册 + PrivateGroup 挂载，对齐前端 `menu.ts` 7 接口：

- **request**（`model/system/request/sys_menu.go`，新建）：
  - `MenuSearch`（内嵌 `PageInfo` 兼容前端分页参数、不用；`menuName/status/menuType/parentId`；parentId 用 `int64` 走 query 绑定）。
  - `MenuOperateParams`（`menuId`/`parentId` 用 `Int64String` 兼容前端 IdType[string|number]；其余 menuType/orderNum/path/component/queryParam/isFrame/isCache/visible/status/perms/icon/remark）。
  - > 注：07-12 model 记忆里设想的 `SysMenuSearch`(不内嵌 PageInfo)/`SysMenuReq`(*string 指针) 已被本次实际实现替代——统一为 `MenuSearch`/`MenuOperateParams`，与 dict/post/dept 同款范式。
- **service**（`service/system/sys_menu.go`，新建）：`MenuService`
  - `GetMenuList`：不分页平表（前端 `handleTree` 组装树），menuName 模糊 / status/menuType/parentId 精确，`order_num ASC, menu_id ASC`。
  - `CreateMenu`/`UpdateMenu`：menuName 必填；审计字段从 claims 注入。
  - `DeleteMenu`：**子菜单 + 角色引用(sys_role_menu)双重校验**，任一占用即禁删（对齐 RuoYi）。
  - `GetMenuTreeSelect`：全量平表（选父级/树选择）。
  - `GetRoleMenuTreeSelect(roleId)`：`{menus: 全量平表, checkedKeys: 角色已分配菜单的叶子 ID}`；`leafCheckedKeys` 取角色菜单中"没有子菜单也属角色菜单"的节点（对齐 RuoYi，供 NTree cascade 回显）。
  - `CascadeDeleteMenu`：`collectWithDescendants` 按 parent_id 递归收集选中+全部子孙（菜单无 ancestors 字段），删菜单 + 清理 sys_role_menu。
- **api**（`api/v1/system/sys_menu.go`，新建）：`MenuApi` 7 handler + swag 注释；审计字段走 `utils.GetUserID(c)`；批量 ID 解析用 `strings.SplitSeq`。
- **router**（`router/system/sys_menu.go`，新建）：`MenuRouter.InitMenuRouter` 挂 `system/menu` group，注册 list / treeselect / roleMenuTreeselect/:roleId / POST / PUT / DELETE`:menuId` / DELETE`cascade/:menuIds`。
- **注册**：三个 `enter.go` 加 `MenuService` / `MenuApi`+`menuService` / `MenuRouter`+`menuApi`；`initialize/router.go` PrivateGroup 加 `InitMenuRouter`。

## 设计决策

- **菜单不分页、后端返平表**：前端 `handleTree`/`treeTransform` 组装树，后端无需建树。treeselect / roleMenuTreeselect 的 menus 同为平表。
- **DELETE :menuId 与 cascade/:menuIds 同层共存**：gin 允许 static(cascade)+param(:menuId) 同层（static 优先匹配），经 `router/system/sys_menu_test.go` 的 `TestMenuRouterRegistration` 验证不 panic（测试保留作回归防护）。
- **roleMenuTreeselect 返回叶子 checkedKeys**：对齐 RuoYi——NTree cascade 模式只需回显最深层节点，父级自动半选/全选；`leafCheckedKeys` 用 `parent_id IN roleIds AND menu_id IN roleIds` 找"有子也属角色"的父，取补集得叶子。
- **菜单无 ancestors，cascade 靠递归**：`collectWithDescendants` 逐层 Pluck parent_id，避免一次拉全表。
- **菜单+按钮权限种子早已存在**：`source/system/sys_menu.go` 已 seed `route.system_menu` C 菜单 + 4 个 F 按钮（`system:menu:query/add/edit/remove`），本次无需动菜单 seed。

## 前端契约要点

- `GET /system/menu/list`：`MenuSearchParams={menuName,status,menuType,parentId}`，返回平表 `Menu[]`。`{parentId,menuType:'F'}` 用于查某菜单的按钮列表（页面右侧按钮权限表）。
- `DELETE /system/menu/{menuId}`：删**单个**（非批量）；`DELETE /system/menu/cascade/{menuIds}`：级联删（逗号分隔）。
- `GET /system/menu/roleMenuTreeselect/{roleId}`：返回 `RoleMenuTreeSelect={checkedKeys,menus}`（角色分配菜单回显）。
- ID 字符串传输（`json:",string"`），前端 IdType=string|number。

## 前端 i18n（07-13 已落地）

`page.system.menu` 全量 53 key（zh/en + `app.d.ts` 声明）齐全。

## 相关文件

- 前端：`web/src/views/_admin/system/menu/`、`web/src/service/api/system/menu.ts`、`web/src/typings/api/system.api.d.ts`
- 后端 model：`server/model/system/sys_menu.go`（SysMenu + RoleMenuTreeSelect）、`server/model/system/request/sys_menu.go`、`sys_role_menu.go`（删除/级联/角色引用依赖）
- 后端接口四层：`server/service/system/sys_menu.go`、`server/api/v1/system/sys_menu.go`、`server/router/system/sys_menu.go`、`server/router/system/sys_menu_test.go`（及三个 `enter.go`、`initialize/router.go`）
- 关联：[[dict-management]] / [[post-management]] / [[department-management]]（同模式参照）、[[menu-seed-routes-alignment]]（菜单/权限 seed）
