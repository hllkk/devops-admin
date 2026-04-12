# 前端类型定义修复设计

**Created:** 2026-04-12
**Status:** Draft

---

## 问题分析

### 问题现象

用户报告两个问题：
1. **角色授权页面按钮缺失** - 菜单树中无法显示按钮节点，无法授权按钮权限
2. **菜单类型字段消失** - 菜单管理页面的菜单类型字段不再显示

### 根因分析

**根本原因：前端类型定义不完整**

| 位置 | 问题 |
|------|------|
| `frontend/src/typings/api/system.api.d.ts:87-91` | `MenuTreeSelectItem` 类型缺少 `menuType`, `perms`, `status`, `hiddenInMenu`, `icon`, `module` 字段 |

---

## 前后端数据流分析

### 后端响应结构

**文件:** `backend/model/system/response/sys_menu.go:56-67`

```go
type MenuTreeSelectItem struct {
    ID           int64                `json:"id"`
    Label        string               `json:"label"`
    Icon         string               `json:"icon,omitempty"`
    MenuType     string               `json:"menuType,omitempty"`    // 关键：按钮节点为 'F'
    Status       string               `json:"status,omitempty"`
    HiddenInMenu bool                 `json:"hiddenInMenu,omitempty"`
    Module       string               `json:"module,omitempty"`
    Perms        string               `json:"perms,omitempty"`       // 关键：按钮权限编码
    Children     []MenuTreeSelectItem `json:"children,omitempty"`
}
```

**按钮节点构建逻辑:** `backend/service/system/sys_role.go:945-958`

```go
// 菜单类型(C)且有按钮时，添加按钮作为子节点
if menu.MenuType == "C" && len(menu.Buttons) > 0 {
    buttonChildren := make([]SysResponse.MenuTreeSelectItem, 0, len(menu.Buttons))
    for _, button := range menu.Buttons {
        buttonChildren = append(buttonChildren, SysResponse.MenuTreeSelectItem{
            ID:       button.ID,
            Label:    button.Label,
            MenuType: "F",    // 按钮节点 menuType='F'
            Perms:    button.Code,
            Icon:     "material-symbols:smart-button-outline",
        })
    }
    children = append(buttonChildren, children...)
}
```

### 前端类型定义（不完整）

**文件:** `frontend/src/typings/api/system.api.d.ts:87-91`

```typescript
type MenuTreeSelectItem = {
  id: CommonType.IdType;
  label: string;
  children?: MenuTreeSelectItem[];
  // 缺少: menuType, status, hiddenInMenu, module, perms, icon
};
```

### 前端组件使用

**文件:** `frontend/src/components/custom/menu-tree.vue`

| 行号 | 代码 | 作用 |
|------|------|------|
| 150 | `if (option.menuType === 'F')` | 检测按钮节点，显示权限码 tooltip |
| 157 | `option.perms || ''` | 显示按钮权限编码 |
| 186 | `if (option.menuType === 'F')` | 检测按钮节点，显示按钮图标 |
| 165 | `if (option.status === '0')` | 检测停用菜单，显示禁用状态 |
| 173 | `if (option.hiddenInMenu === true)` | 检测隐藏菜单，显示隐藏状态 |

**影响分析：**
- TypeScript 不识别 `menuType` 属性，但 JavaScript 运行时可以访问
- 组件逻辑正确，但类型不匹配会导致 TypeScript 编译警告/错误
- Vite 开发模式下可能不报错，但 `pnpm typecheck` 会检测到问题

---

## 角色授权数据流

### API 路径

1. 前端调用: `fetchGetRoleAuthTree(roleId)` → `/system/role/auth-tree/:id`
2. 后端处理: `RoleAuthTreeView` → `GetRoleAuthTree(roleId)`
3. 返回数据: `RoleAuthTreeResponse`

### 后端响应结构

**文件:** `backend/model/system/response/sys_role.go:64-74`

```go
type RoleAuthTreeResponse struct {
    Trees       map[string][]MenuTreeSelectItem `json:"trees"`       // module -> 菜单树
    CheckedKeys RoleAuthCheckedKeys             `json:"checkedKeys"` // 已选中的ID
}

type RoleAuthCheckedKeys struct {
    Menus   []int64 `json:"menus"`   // 已选菜单ID
    Buttons []int64 `json:"buttons"` // 已选按钮ID
}
```

### 前端响应类型

**文件:** `frontend/src/typings/api/system.api.d.ts:93-100`

```typescript
type RoleAuthTreeResponse = {
  trees: Record<string, MenuTreeSelectItem[]>;  // 使用不完整的 MenuTreeSelectItem
  checkedKeys: {
    menus: CommonType.IdType[];
    buttons: CommonType.IdType[];
  };
};
```

---

## 修复方案

### 修改文件

`frontend/src/typings/api/system.api.d.ts`

### 修改内容

将第 87-91 行的类型定义修改为：

```typescript
/** menu tree select item */
type MenuTreeSelectItem = {
  id: CommonType.IdType;
  label: string;
  /** 菜单类型（M目录 C菜单 F按钮） */
  menuType?: 'M' | 'C' | 'F';
  /** 菜单图标 */
  icon?: string;
  /** 菜单状态（0正常 1停用） */
  status?: Common.EnableStatus;
  /** 是否隐藏 */
  hiddenInMenu?: boolean;
  /** 所属模块 */
  module?: string;
  /** 权限标识（按钮节点使用） */
  perms?: string;
  children?: MenuTreeSelectItem[];
};
```

### 字段说明

| 字段 | 类型 | 来源 | 用途 |
|------|------|------|------|
| `menuType` | `'M' \| 'C' \| 'F'` | 后端 MenuType | 区分目录/菜单/按钮节点 |
| `icon` | `string` | 后端 Icon | 显示节点图标 |
| `status` | `Common.EnableStatus` | 后端 Status | 显示节点状态 |
| `hiddenInMenu` | `boolean` | 后端 HiddenInMenu | 显示隐藏状态 |
| `module` | `string` | 后端 Module | 模块归属 |
| `perms` | `string` | 后端 Perms | 按钮权限编码（仅按钮节点） |

---

## 验证方案

### 1. TypeScript 类型检查

```bash
cd frontend && pnpm typecheck
```

预期：无类型错误

### 2. 功能验证

**角色授权页面:**
1. 进入角色管理 → 编辑角色
2. 查看菜单权限树
3. 确认：
   - 菜单节点下显示按钮节点（带按钮图标）
   - 按钮节点 hover 显示权限编码 tooltip
   - 可以勾选按钮节点进行授权

**菜单管理页面:**
1. 进入菜单管理
2. 选择一个菜单节点
3. 查看菜单详情区域
4. 确认菜单类型字段正常显示

---

## 影响范围

- 仅修改类型定义文件，无运行时逻辑变更
- 影响所有使用 `MenuTreeSelectItem` 类型的组件：
  - `menu-tree.vue` - 菜单树组件
  - `role-operate-drawer.vue` - 角色授权抽屉
- 类型修复后，组件逻辑无需修改（已正确实现）

---

## 关联问题

此修复与之前已修复的问题关联：
- **问题1:** 菜单管理按钮列表 API 错误（已修复）
- **问题2:** 类型定义不完整（本次修复）

两者独立但相互影响：
- 问题1修复后，按钮数据可以正确获取
- 问题2修复后，按钮数据可以在授权树中正确显示