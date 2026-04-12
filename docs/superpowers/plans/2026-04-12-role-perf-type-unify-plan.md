# 角色权限性能优化与类型统一实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将编辑角色时的 API 请求量从 `2N + M` 减少到 `1`，并统一前后端菜单类型为 "M"/"C"/"F"。

**Architecture:** 后端新增统一 API `/system/role/auth-tree/{roleId}` 返回完整菜单-按钮树，前端移除冗余请求和兼容逻辑。

**Tech Stack:** Go 1.26 + Gin + GORM (后端), Vue3 + TypeScript + Naive UI (前端)

---

## File Structure

### 后端文件
| 文件 | 改动类型 | 职责 |
|------|----------|------|
| `backend/service/system/sys_menu.go` | 修改 | 移除 convertMenuType，修复硬编码类型判断 |
| `backend/service/system/sys_role.go` | 修改 | 新增 GetRoleAuthTree 方法 |
| `backend/api/v1/system/sys_role.go` | 修改 | 新增 GetRoleAuthTreeView 接口 |
| `backend/router/system/sys_role.go` | 修改 | 注册新路由 |
| `backend/model/system/response/sys_role.go` | 修改 | 新增 RoleAuthTreeResponse 结构体 |

### 前端文件
| 文件 | 改动类型 | 职责 |
|------|----------|------|
| `frontend/src/typings/api/system.api.d.ts` | 修改 | 新增 RoleAuthTreeResponse 类型 |
| `frontend/src/service/api/system/role.ts` | 修改 | 新增 fetchGetRoleAuthTree API |
| `frontend/src/service/api/system/menu.ts` | 修改 | 移除冗余 API |
| `frontend/src/components/custom/menu-tree.vue` | 修改 | 移除按钮加载逻辑，新增 loadRoleAuthTree |
| `frontend/src/views/manage/role/modules/role-operate-drawer.vue` | 修改 | 简化编辑流程，移除冗余调用 |

---

## Phase 1: 数据库迁移

### Task 1: 执行数据库迁移 SQL

**Files:**
- 数据库 `sys_menus` 表

**说明:** 此步骤需要手动执行 SQL，将数据库中旧的类型值更新为统一标准。

- [ ] **Step 1: 备份 sys_menus 表**

```sql
-- 创建备份表
CREATE TABLE sys_menus_backup_20260412 AS SELECT * FROM sys_menus;
```

- [ ] **Step 2: 执行类型迁移**

```sql
-- 更新菜单类型字段值
UPDATE sys_menus SET menu_type = 'M' WHERE menu_type = '1';
UPDATE sys_menus SET menu_type = 'C' WHERE menu_type = '2';
```

- [ ] **Step 3: 验证迁移结果**

```sql
-- 检查是否还有旧类型值
SELECT menu_type, COUNT(*) FROM sys_menus GROUP BY menu_type;
-- 预期结果: 只有 'M', 'C' 两种值（按钮在 sys_buttons 表）
```

- [ ] **Step 4: 记录迁移完成**

在项目文档中记录迁移已完成，以便后续部署参考。

---

## Phase 2: 后端类型统一

### Task 2: 移除 convertMenuType 函数

**Files:**
- Modify: `backend/service/system/sys_menu.go:92-104`

- [ ] **Step 1: 删除 convertMenuType 函数**

找到并删除以下代码块（第92-104行）：

```go
// 删除整个函数
// convertMenuType 转换菜单类型从数据库格式到标准格式
// 数据库存储: "1"=目录, "2"=菜单, "F"=按钮
// 标准格式: "M"=目录, "C"=菜单, "F"=按钮
func convertMenuType(dbType string) common.MenuType {
	switch dbType {
	case "1":
		return common.MenuTypeCatalog // "M"
	case "2":
		return common.MenuTypeMenu // "C"
	default:
		return common.MenuType(dbType)
	}
}
```

删除后，`buildMenuTree` 函数中第177行的调用需要修改。

- [ ] **Step 2: 修改 buildMenuTree 中的类型赋值**

文件 `backend/service/system/sys_menu.go:177`，将：

```go
Type:         convertMenuType(menu.MenuType),
```

改为：

```go
Type:         common.MenuType(menu.MenuType),
```

- [ ] **Step 3: 验证编译**

```bash
cd backend && go build ./...
```

Expected: 编译成功，无错误

- [ ] **Step 4: Commit**

```bash
git add backend/service/system/sys_menu.go
git commit -m "refactor: remove convertMenuType function, use unified type M/C/F directly"
```

### Task 3: 修复 buildMenuTreeSelect 硬编码类型判断

**Files:**
- Modify: `backend/service/system/sys_menu.go:298`

- [ ] **Step 1: 修改硬编码类型判断**

文件 `backend/service/system/sys_menu.go:298`，将：

```go
if menu.MenuType == "2" && len(menu.Buttons) > 0 {
```

改为：

```go
if menu.MenuType == "C" && len(menu.Buttons) > 0 {
```

- [ ] **Step 2: 验证编译**

```bash
cd backend && go build ./...
```

Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add backend/service/system/sys_menu.go
git commit -m "fix: use 'C' instead of '2' for menu type check"
```

---

## Phase 3: 后端新增统一 API

### Task 4: 新增 RoleAuthTreeResponse 结构体

**Files:**
- Modify: `backend/model/system/response/sys_role.go`

- [ ] **Step 1: 在文件末尾添加新结构体**

在 `backend/model/system/response/sys_role.go` 文件末尾添加：

```go
// RoleAuthTreeResponse 角色权限树响应（统一菜单+按钮）
type RoleAuthTreeResponse struct {
	Trees       map[string][]MenuTreeSelectItem `json:"trees"`       // module -> 菜单树
	CheckedKeys RoleAuthCheckedKeys             `json:"checkedKeys"` // 已选中的ID
}

// RoleAuthCheckedKeys 已选中的权限ID
type RoleAuthCheckedKeys struct {
	Menus   []int64 `json:"menus"`   // 已选菜单ID
	Buttons []int64 `json:"buttons"` // 已选按钮ID
}
```

- [ ] **Step 2: 验证编译**

```bash
cd backend && go build ./...
```

Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add backend/model/system/response/sys_role.go
git commit -m "feat: add RoleAuthTreeResponse struct for unified auth tree API"
```

### Task 5: 新增 GetRoleAuthTree 服务方法

**Files:**
- Modify: `backend/service/system/sys_role.go`

- [ ] **Step 1: 在文件末尾添加新方法**

在 `backend/service/system/sys_role.go` 文件末尾添加：

```go
// GetRoleAuthTree 获取角色完整权限树（菜单+按钮统一返回）
func (r *RoleService) GetRoleAuthTree(roleId int64) (SysResponse.RoleAuthTreeResponse, error) {
	var response SysResponse.RoleAuthTreeResponse
	response.Trees = make(map[string][]SysResponse.MenuTreeSelectItem)

	if global.OPS_DB == nil {
		return response, fmt.Errorf("数据库尚未初始化！！！")
	}

	// 1. 获取角色信息
	var role system.SysRole
	if err := global.OPS_DB.Where("id = ?", roleId).First(&role).Error; err != nil {
		return response, fmt.Errorf("角色不存在")
	}

	// 2. 获取角色已关联的菜单ID
	var roleMenus []system.SysMenu
	if err := global.OPS_DB.Where("id IN (?)", global.OPS_DB.Table("sys_role_menus").
		Select("sys_menu_id").Where("sys_role_id = ?", roleId)).
		Find(&roleMenus).Error; err != nil {
		return response, fmt.Errorf("获取角色菜单失败: %w", err)
	}

	// 3. 获取角色已关联的按钮ID
	var roleButtons []system.SysButton
	if err := global.OPS_DB.Where("id IN (?)", global.OPS_DB.Table("sys_role_buttons").
		Select("sys_button_id").Where("sys_role_id = ?", roleId)).
		Find(&roleButtons).Error; err != nil {
		return response, fmt.Errorf("获取角色按钮失败: %w", err)
	}

	// 收集已选中的ID
	for _, menu := range roleMenus {
		response.CheckedKeys.Menus = append(response.CheckedKeys.Menus, menu.ID)
	}
	for _, btn := range roleButtons {
		response.CheckedKeys.Buttons = append(response.CheckedKeys.Buttons, btn.ID)
	}

	// 4. 获取所有应用模块
	var apps []system.SysApp
	if err := global.OPS_DB.Find(&apps).Error; err != nil {
		return response, fmt.Errorf("获取应用列表失败: %w", err)
	}

	// 5. 为每个模块构建菜单树（包含按钮）
	for _, app := range apps {
		tree, err := menuService.GetMenuTreeSelectWithButtons(app.AppCode)
		if err != nil {
			global.OPS_LOG.Error("获取模块菜单树失败", zap.String("module", app.AppCode), zap.Error(err))
			continue
		}
		response.Trees[app.AppCode] = tree
	}

	return response, nil
}
```

- [ ] **Step 2: 新增 GetMenuTreeSelectWithButtons 方法**

在 `backend/service/system/sys_menu.go` 文件末尾添加：

```go
// GetMenuTreeSelectWithButtons 获取菜单树选择（包含按钮节点，用于权限分配）
func (m *MenuService) GetMenuTreeSelectWithButtons(module string) ([]SysResponse.MenuTreeSelectItem, error) {
	if global.OPS_DB == nil {
		return nil, fmt.Errorf("数据库尚未初始化！！！")
	}

	var menus []system.SysMenu
	query := global.OPS_DB.Preload("Buttons", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_by ASC")
	}).Where("status = ? AND constant = ? AND fixed = ?", "1", false, false)

	if module != "" {
		query = query.Joins("JOIN sys_app_menus ON sys_app_menus.sys_menu_id = sys_menus.id").
			Joins("JOIN sys_apps ON sys_apps.id = sys_app_menus.sys_app_id").
			Where("sys_apps.app_code = ?", module)
	}

	err := query.Order("`order_by` ASC").Find(&menus).Error
	if err != nil {
		global.OPS_LOG.Error("获取菜单列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取菜单列表失败: %w", err)
	}

	return m.buildMenuTreeSelectWithButtons(menus, 0, module), nil
}

// buildMenuTreeSelectWithButtons 构建菜单树选择项（包含按钮作为子节点）
func (m *MenuService) buildMenuTreeSelectWithButtons(menus []system.SysMenu, parentId int64, module string) []SysResponse.MenuTreeSelectItem {
	var tree []SysResponse.MenuTreeSelectItem
	for _, menu := range menus {
		if menu.ParentId == parentId {
			menuName := menu.MenuName
			if menu.I18nKey != "" {
				menuName = menu.I18nKey
			}

			node := SysResponse.MenuTreeSelectItem{
				ID:           menu.ID,
				Label:        menuName,
				Icon:         menu.Icon,
				MenuType:     menu.MenuType, // 直接使用数据库值 M/C
				Status:       menu.Status,
				HiddenInMenu: menu.HiddenInMenu,
				Module:       module,
			}

			children := m.buildMenuTreeSelectWithButtons(menus, menu.ID, module)

			// 如果是菜单类型(C)且有按钮，添加按钮作为子节点
			if menu.MenuType == "C" && len(menu.Buttons) > 0 {
				buttonChildren := make([]SysResponse.MenuTreeSelectItem, 0, len(menu.Buttons))
				for _, button := range menu.Buttons {
					buttonChildren = append(buttonChildren, SysResponse.MenuTreeSelectItem{
						ID:       button.ID,
						Label:    button.Label,
						MenuType: "F",
						Perms:    button.Code,
						Icon:     "material-symbols:smart-button-outline",
					})
				}
				children = append(buttonChildren, children...)
			}

			if len(children) > 0 {
				node.Children = children
			}

			tree = append(tree, node)
		}
	}
	return tree
}
```

需要在文件顶部添加 `gorm.DB` 的导入（如果尚未导入）：

```go
import (
	"gorm.io/gorm"
)
```

- [ ] **Step 3: 验证编译**

```bash
cd backend && go build ./...
```

Expected: 编译成功

- [ ] **Step 4: Commit**

```bash
git add backend/service/system/sys_role.go backend/service/system/sys_menu.go
git commit -m "feat: add GetRoleAuthTree service for unified menu+button tree"
```

### Task 6: 新增 API 接口和路由

**Files:**
- Modify: `backend/api/v1/system/sys_role.go`
- Modify: `backend/router/system/sys_role.go`

- [ ] **Step 1: 新增 API 接口方法**

在 `backend/api/v1/system/sys_role.go` 文件末尾添加：

```go
// RoleAuthTreeView 获取角色完整权限树（菜单+按钮）
func (r *RoleApi) RoleAuthTreeView(c *gin.Context) {
	idParam := c.Param("id")
	if idParam == "" {
		response.FailWithMessage("角色ID不能为空", c)
		return
	}

	roleId, err := utils.ParseId(idParam)
	if err != nil {
		response.FailWithMessage("参数格式错误", c)
		return
	}

	authTree, err := roleService.GetRoleAuthTree(roleId)
	if err != nil {
		global.OPS_LOG.Error("获取角色权限树失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(authTree, "获取角色权限树成功", c)
}
```

- [ ] **Step 2: 注册路由**

修改 `backend/router/system/sys_role.go`，在 roleRouter 组中添加新路由：

在第14行附近添加：

```go
roleRouter.GET("auth-tree/:id", RoleApi.RoleAuthTreeView) // 获取角色完整权限树
```

完整修改后的路由注册：

```go
func (r *RoleRouter) InitRoleRouter(Router *gin.RouterGroup) {
	systemRouter := Router.Group("system")
	{
		roleRouter := systemRouter.Group("role")
		{
			roleRouter.GET("select", RoleApi.RoleSelectView)                    // 获取角色选择框列表
			roleRouter.GET("list", RoleApi.RoleListView)                        // 获取角色列表
			roleRouter.GET("auth-tree/:id", RoleApi.RoleAuthTreeView)          // 获取角色完整权限树
			roleRouter.GET("buttons/:id", RoleApi.RoleButtonTreeView)          // 获取角色按钮权限（保留兼容）
			roleRouter.GET("/:id", RoleApi.RoleDetailView)                      // 获取角色详情
			roleRouter.POST("", RoleApi.RoleCreateView)                         // 创建角色
			roleRouter.PUT("", RoleApi.RoleUpdateView)                          // 更新角色
			roleRouter.DELETE("/:ids", RoleApi.RoleDeleteView)                  // 删除角色
			roleRouter.PUT("changeStatus", RoleApi.RoleChangeStatusView)        // 修改角色状态
			roleRouter.PUT("dataScope", RoleApi.RoleUpdateDataScopeView)        // 修改数据权限
			roleRouter.GET("authUser/allocatedList", RoleApi.RoleAuthUserListView) // 已授权用户列表
			roleRouter.PUT("authUser/selectAll", RoleApi.RoleAuthUserView)      // 批量授权用户
			roleRouter.PUT("authUser/cancelAll", RoleApi.RoleCancelAuthUserView) // 批量取消授权
			roleRouter.GET("deptTree/:id", RoleApi.RoleDeptTreeView)            // 获取部门树
			roleRouter.POST("export", RoleApi.RoleExportView)                   // 导出角色
		}
	}
}
```

- [ ] **Step 3: 验证编译**

```bash
cd backend && go build ./...
```

Expected: 编译成功

- [ ] **Step 4: Commit**

```bash
git add backend/api/v1/system/sys_role.go backend/router/system/sys_role.go
git commit -m "feat: add RoleAuthTreeView API endpoint for unified auth tree"
```

---

## Phase 4: 前端类型定义更新

### Task 7: 更新前端类型定义

**Files:**
- Modify: `frontend/src/typings/api/system.api.d.ts`

- [ ] **Step 1: 新增 RoleAuthTreeResponse 类型**

在 `frontend/src/typings/api/system.api.d.ts` 文件中，在 `RoleMenuTreeSelect` 类型定义附近（约第81行），添加：

```typescript
/** role auth tree response (unified menu + button) */
type RoleAuthTreeResponse = {
  trees: Record<string, MenuTreeSelectItem[]>;
  checkedKeys: {
    menus: CommonType.IdType[];
    buttons: CommonType.IdType[];
  };
};
```

- [ ] **Step 2: 运行 TypeScript 检查**

```bash
cd frontend && pnpm typecheck
```

Expected: 无类型错误

- [ ] **Step 3: Commit**

```bash
git add frontend/src/typings/api/system.api.d.ts
git commit -m "feat: add RoleAuthTreeResponse type definition"
```

---

## Phase 5: 前端 API 层更新

### Task 8: 新增 fetchGetRoleAuthTree API

**Files:**
- Modify: `frontend/src/service/api/system/role.ts`

- [ ] **Step 1: 添加新 API 函数**

在 `frontend/src/service/api/system/role.ts` 文件末尾添加：

```typescript
/** 获取角色完整权限树（菜单+按钮统一返回） */
export function fetchGetRoleAuthTree(roleId: CommonType.IdType) {
  return request<Api.System.RoleAuthTreeResponse>({
    url: `/system/role/auth-tree/${roleId}`,
    method: 'get'
  });
}
```

- [ ] **Step 2: 运行 lint 检查**

```bash
cd frontend && pnpm lint
```

Expected: 无 lint 错误

- [ ] **Step 3: Commit**

```bash
git add frontend/src/service/api/system/role.ts
git commit -m "feat: add fetchGetRoleAuthTree API function"
```

---

## Phase 6: 前端组件改动

### Task 9: 更新 menu-tree.vue 移除冗余逻辑

**Files:**
- Modify: `frontend/src/components/custom/menu-tree.vue`

**说明:** 这是一个较大的改动，需要分多个步骤完成。

- [ ] **Step 1: 移除 buttonsMap 相关状态**

删除第45行的 `buttonsMap` 定义：

```typescript
// 删除这行
const buttonsMap = reactive<Record<string, Api.System.ButtonList>>({}); // menuId -> buttons
```

删除第46行的 `buttonsLoading` 定义：

```typescript
// 删除这行
const buttonsLoading = ref<boolean>(false);
```

- [ ] **Step 2: 移除两个 watchEffect**

删除第73-84行的第一个 watchEffect：

```typescript
// 删除整个 watchEffect
watchEffect(async () => {
  if (!showButtons.value || options.value.length === 0 || buttonsLoading.value) return;
  // ... 全部删除
});
```

删除第88-104行的第二个 watchEffect：

```typescript
// 删除整个 watchEffect
watchEffect(async () => {
  if (!showButtons.value || !props.showModuleTabs || buttonsLoading.value) return;
  // ... 全部删除
});
```

- [ ] **Step 3: 移除辅助函数**

删除以下函数：
- `checkTreeHasButtons`（第106-117行）
- `getMenuIdsOfTypeC`（第119-133行）
- `loadMenuButtons`（第177-185行）
- `buildTreeWithButtons`（第187-242行）
- `clearButtonsCache`（第472-478行）

- [ ] **Step 4: 移除 computedOptions 计算属性中的按钮处理**

修改第52-57行的 `computedOptions`：

```typescript
// 改为直接返回 options
const computedOptions = computed(() => {
  return options.value;
});
```

修改第59-70行的 `computedModuleMenuOptions`：

```typescript
// 改为直接返回 moduleMenuOptions
const computedModuleMenuOptions = computed(() => {
  return moduleMenuOptions;
});
```

- [ ] **Step 5: 添加导入 fetchGetRoleAuthTree**

在文件顶部导入区添加：

```typescript
import { fetchGetRoleAuthTree } from '@/service/api/system/role';
```

- [ ] **Step 6: 新增 loadRoleAuthTree 方法**

在文件中（约第417行 refresh 函数之后）添加新方法：

```typescript
/** 加载角色完整权限树 */
async function loadRoleAuthTree(roleId: CommonType.IdType) {
  const { data, error } = await fetchGetRoleAuthTree(roleId);
  if (error) return null;

  // 设置各模块的菜单树数据
  for (const [module, tree] of Object.entries(data.trees)) {
    moduleMenuOptions[module] = [
      {
        id: 0,
        label: 'menu.root',
        icon: 'material-symbols:home-outline-rounded',
        children: tree
      }
    ] as Api.System.MenuList;
  }

  return data.checkedKeys;
}
```

- [ ] **Step 7: 更新 defineExpose**

修改第491-503行的 defineExpose，添加 `loadRoleAuthTree` 并移除 `clearButtonsCache`：

```typescript
defineExpose({
  getCheckedMenuIds,
  getCheckedButtonIds,
  refresh,
  setCheckedKeysByModule,
  clearAllCheckedKeys,
  loadRoleAuthTree,
  getAppList,
  getModuleMenuList,
  get appList() { return appList.value; }
});
```

- [ ] **Step 8: 更新 getCheckedButtonIds 方法**

修改第481-489行的 `getCheckedButtonIds`：

```typescript
/** 获取勾选的按钮ID列表 */
function getCheckedButtonIds(): CommonType.IdType[] {
  const buttonIds: CommonType.IdType[] = [];

  if (props.showModuleTabs) {
    for (const module of appList.value) {
      const keys = moduleCheckedKeys[module.appCode] || [];
      for (const id of keys) {
        const node = findNodeById(id, moduleMenuOptions[module.appCode] || []);
        if (node && node.menuType === 'F') {
          buttonIds.push(id);
        }
      }
    }
  } else {
    for (const id of checkedKeys.value) {
      const node = findNodeById(id, options.value);
      if (node && node.menuType === 'F') {
        buttonIds.push(id);
      }
    }
  }

  return [...new Set(buttonIds)];
}
```

- [ ] **Step 9: 移除模板中的 buttonsLoading**

修改模板第525行和第537行，移除 `|| buttonsLoading`：

```vue
<!-- 第525行改为 -->
<NSpin class="resource h-full w-full py-6px pl-3px" content-class="h-full" :show="moduleLoading[app.appCode]">
```

```vue
<!-- 第537行改为 -->
:loading="moduleLoading[app.appCode]"
```

- [ ] **Step 10: 运行 typecheck**

```bash
cd frontend && pnpm typecheck
```

Expected: 无类型错误

- [ ] **Step 11: 运行 lint**

```bash
cd frontend && pnpm lint
```

Expected: 无 lint 错误

- [ ] **Step 12: Commit**

```bash
git add frontend/src/components/custom/menu-tree.vue
git commit -m "refactor: remove button loading logic, add loadRoleAuthTree method"
```

### Task 10: 更新 role-operate-drawer.vue 简化编辑流程

**Files:**
- Modify: `frontend/src/views/manage/role/modules/role-operate-drawer.vue`

- [ ] **Step 1: 移除冗余导入**

删除第6行的 `fetchGetRoleButtonTreeSelect` 导入：

```typescript
// 修改第6行，移除 fetchGetRoleButtonTreeSelect
import { fetchGetRoleMenuTreeSelect, fetchGetRoleButtonTreeSelect } from '@/service/api/system';
// 改为
import { fetchGetRoleMenuTreeSelect } from '@/service/api/system';
```

由于不再需要 fetchGetRoleMenuTreeSelect（已在 menu-tree.vue 内部处理），可以完全移除这行导入。

- [ ] **Step 2: 简化 handleUpdateModelWhenEdit 方法**

修改第81-128行的 `handleUpdateModelWhenEdit` 方法：

```typescript
async function handleUpdateModelWhenEdit() {
  model.value = createDefaultModel();
  model.value.menuIds = [];

  if (props.operateType === 'add') {
    // 新增时组件会自动加载数据 (immediate=true)
    return;
  }

  if (props.operateType === 'edit' && props.rowData) {
    startMenuLoading();
    Object.assign(model.value, jsonClone(props.rowData));

    // 等待组件挂载
    await nextTick();

    // 加载完整权限树（包含所有模块的菜单和按钮）
    const authTree = await menuTreeRef.value?.loadRoleAuthTree(model.value.roleId!);

    if (authTree) {
      // 合并菜单和按钮的选中ID，设置到各模块
      const allCheckedKeys = [...authTree.menus, ...authTree.buttons];
      const apps = menuTreeRef.value?.appList || [];

      for (const app of apps) {
        menuTreeRef.value?.setCheckedKeysByModule(app.appCode, allCheckedKeys);
      }
    }

    stopMenuLoading();
  }
}
```

- [ ] **Step 3: 运行 typecheck**

```bash
cd frontend && pnpm typecheck
```

Expected: 无类型错误

- [ ] **Step 4: 运行 lint**

```bash
cd frontend && pnpm lint
```

Expected: 无 lint 错误

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/manage/role/modules/role-operate-drawer.vue
git commit -m "refactor: simplify role edit flow, use unified auth-tree API"
```

---

## Phase 7: 清理与验证

### Task 11: 移除前端冗余 API 定义

**Files:**
- Modify: `frontend/src/service/api/system/menu.ts`

- [ ] **Step 1: 移除 fetchGetRoleButtonTreeSelect**

删除第117-126行的 `fetchGetRoleButtonTreeSelect` 函数：

```typescript
// 删除整个函数
export function fetchGetRoleButtonTreeSelect(roleId: CommonType.IdType, module?: string) {
  return request<{
    checkedKeys: CommonType.IdType[];
    buttons: Api.System.Button[];
  }>({
    url: `/system/role/buttons/${roleId}`,
    method: 'get',
    params: module ? { module } : undefined
  });
}
```

- [ ] **Step 2: 移除 fetchGetMenuButtons（可选保留）**

注意：`fetchGetMenuButtons` 在菜单编辑页面可能仍需使用，暂保留。如果确认菜单编辑页面也不需要，可删除第109-114行。

- [ ] **Step 3: 运行 typecheck**

```bash
cd frontend && pnpm typecheck
```

Expected: 无类型错误（可能有未使用导入警告，需清理）

- [ ] **Step 4: Commit**

```bash
git add frontend/src/service/api/system/menu.ts
git commit -m "refactor: remove redundant fetchGetRoleButtonTreeSelect API"
```

### Task 12: 后端编译验证

- [ ] **Step 1: 编译后端**

```bash
cd backend && go build -o devopsAdmin ./main.go
```

Expected: 编译成功，生成 devopsAdmin 可执行文件

- [ ] **Step 2: 删除编译产物**

```bash
cd backend && rm -f devopsAdmin
```

- [ ] **Step 3: 运行后端测试（如有）**

```bash
cd backend && go test ./...
```

Expected: 测试通过（或无测试文件）

### Task 13: 前端完整验证

- [ ] **Step 1: 运行完整检查**

```bash
cd frontend && pnpm typecheck && pnpm lint
```

Expected: 全部通过

- [ ] **Step 2: 启动开发服务器测试**

```bash
cd frontend && pnpm dev
```

手动测试：
1. 进入角色管理页面
2. 点击编辑按钮
3. 观察响应时间（预期 < 1秒）
4. 检查菜单树是否正确显示（包含按钮节点）
5. 修改权限并保存
6. 重新打开编辑验证选中状态

### Task 14: 最终提交

- [ ] **Step 1: 查看所有改动**

```bash
git status
```

- [ ] **Step 2: 确认改动完整**

确保所有文件都已提交。

- [ ] **Step 3: 推送到远程（可选）**

```bash
git push origin main
```

---

## Self-Review Checklist

**Spec coverage:**
- ✅ 数据库迁移：Task 1
- ✅ 后端类型统一：Task 2, 3
- ✅ 后端新 API：Task 4, 5, 6
- ✅ 前端类型定义：Task 7
- ✅ 前端 API 层：Task 8, 11
- ✅ 前端组件改动：Task 9, 10
- ✅ 验证：Task 12, 13

**Placeholder scan:**
- 无 TBD、TODO、未完成部分
- 所有步骤有具体代码或命令

**Type consistency:**
- `RoleAuthTreeResponse` 类型前后端一致
- `MenuType` 使用 "M"/"C"/"F" 统一标准
- `loadRoleAuthTree` 方法名前后端一致