# 角色授权菜单树按业务模块分组

> 日期：2026-07-24 ｜ 状态：已实现

## 需求
角色管理 → 菜单权限授权树此前把所有模块的菜单**平铺在一个虚拟根下**（admin 的系统管理/定时任务/日志 与 server/gateway 占位首页混在同一层），不清晰。改为按 admin/server/gateway 三个业务模块分组展示，与主框架侧边栏的模块隔离理念贯通。

## 改动

### 后端
- `MenuTreeSelectNode` 加 `Module` 字段（`json:"module,omitempty"`）—— 角色授权树 VO 此前为精简字段把 module 丢了，前端拿不到归属无法分组；菜单管理的完整 `Menu` 类型本就有 module
- `buildMenuTreeSelect`（`service/system/sys_menu.go`）映射补 `Module: n.Module`
- 落库无需改：`saveRoleMenus` 早有 `mid <= 0` 过滤防线，分组虚拟节点不入库

### 前端（`components/custom/menu-tree.vue`）
- **分组放渲染层 `treeData` computed**（不是 `getMenuList` 里）：`options`(平铺顶层菜单) 经 `buildGroupedOptions` 按 module 归类，在根(id=0)下包三个**虚拟分组节点**（负 id -1..-3，复用 `MODULE_CONFIG[mod].icon` + `$t('module.${mod}')`，menuType='M'）
- **为何放渲染层**：`role-operate-drawer` 编辑角色时 `MenuTree :immediate="operateType==='add'"` 为 false → `getMenuList` 不执行 → `options` 由父组件直塞 `data.menus`。分组若只放 `getMenuList`，**编辑场景会漏分组（平铺）**，只有新增角色才分组。首次实现即踩此坑，后改到 computed 统一两入口
- `getAllMenuIds`/`getLeafMenuIds`/`NTree :data` 全部用 `treeData`（含根0+分组负id+菜单）；`getMenuList` 只设 `options=data`（平铺）
- `expandedKeys` 折叠态 `[0,-1,-2,-3]`（根+三分组层），全展态 `getAllMenuIds(treeData)`
- `getCheckedMenuIds` 提交时 `filter(id => Number(id) > 0)`，过滤分组负 id 与根 0，与后端 `mid<=0` 双保险防污染 sys_role_menu
- 分组节点 menuType='M' 不渲染小房子（renderHouse 仅认 C 菜单），模块图标走 renderPrefix
- i18n `module.*` 三处（zh-cn/en-us/app.d.ts）早已存在，零新增 key

## 关键约束
- **分组必须在渲染层 `treeData` computed**：编辑角色 `immediate=false` 绕过 `getMenuList`、`options` 由父组件直塞 `data.menus`；分组放 `getMenuList` 会导致**编辑场景平铺无分组**（首次实现踩坑点）
- **分组节点 = 纯 UI 归类**，负 id（-1..-3），不写 sys_role_menu（后端 `saveRoleMenus` `mid<=0` + 前端 `getCheckedMenuIds` `>0` 双过滤）
- **角色是跨模块的**：用「一棵树 + 模块分组文件夹」而非 Tab 切换，一次操作可勾多模块；勾选 / cascade / 默认路由小房子逻辑全部不变
- module 缺失（老库未回填）的菜单兜底归 admin
- 分组节点 menuType='M'，故不渲染默认路由小房子（renderHouse 仅认 C 菜单）

## 相关
- [[module-isolation-backend-driven]] 模块隔离（module 字段来源）
- [[role-default-router]] 角色默认路由（MenuTreeSelectNode 此前加 Path，本次加 Module，同一 VO 的两次扩字段）
