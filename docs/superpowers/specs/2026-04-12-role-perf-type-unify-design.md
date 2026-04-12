# 角色权限性能优化与类型统一设计文档

## 问题背景

前端在打开编辑角色的窗口时，会导致浏览器卡住。经分析发现：

1. **性能问题**：编辑角色时 API 请求量过大（`2N + M` 次请求）
2. **类型不一致**：前后端菜单类型表示方式不统一，存在兼容判断逻辑

## 问题根因分析

### 性能问题

`role-operate-drawer.vue` 编辑角色时的请求流程：
- 遍历每个模块调用 `fetchGetRoleMenuTreeSelect`（N次）
- 遍历每个模块调用 `fetchGetRoleButtonTreeSelect`（N次）
- `watchEffect` 遍历每个菜单调用 `fetchGetMenuButtons`（M次）

总请求量 = `2N + M`，当模块和菜单数量较多时导致浏览器卡住。

### 类型不一致问题

| 位置 | 存储值/使用值 | 标准值 |
|------|---------------|--------|
| 后端数据库 | "1"=目录, "2"=菜单 | - |
| 后端响应 | 通过 `convertMenuType` 转换 | "M", "C", "F" |
| 后端硬编码 | `menu.MenuType == "2"` | 应为 "C" |
| 前端兼容判断 | `'F' || '3'`, `'C' || '2'` | "M", "C", "F" |

## 解决方案

### 方案选择：统一树结构 API + 后端标准化

核心思路：后端返回完整统一的菜单-按钮树，前端只需调用一个 API。

**改进效果**：
- API 请求量从 `2N + M` 减少到 `1`
- 类型完全统一为 "M"/"C"/"F"
- 消除所有兼容判断逻辑

## 设计详情

### 第一部分：后端类型统一

**统一类型定义**：

| 类型 | 含义 | 数据库存储 | API响应 |
|------|------|------------|---------|
| M | 目录 | "M" | "M" |
| C | 菜单 | "C" | "C" |
| F | 按钮 | "F" | "F" |

**数据库迁移 SQL**：

```sql
-- 更新菜单类型字段值
UPDATE sys_menus SET menu_type = 'M' WHERE menu_type = '1';
UPDATE sys_menus SET menu_type = 'C' WHERE menu_type = '2';

-- 确保字段长度足够
ALTER TABLE sys_menus MODIFY COLUMN menu_type VARCHAR(1) NOT NULL COMMENT '菜单类型(M目录 C菜单 F按钮)';
```

**后端改动**：
1. 移除 `convertMenuType` 函数（不再需要转换）
2. 修改硬编码判断：`menu.MenuType == "2"` → `menu.MenuType == "C"`

### 第二部分：API 精简与合并

**新增统一 API**：`/system/role/auth-tree/{roleId}`

**请求参数**：
- `roleId`（路径参数）

**响应结构**：

```typescript
interface RoleAuthTreeResponse {
  trees: Record<string, MenuTreeSelectItem[]>;  // module -> 菜单树（含按钮节点）
  checkedKeys: {
    menus: CommonType.IdType[];   // 已选菜单ID
    buttons: CommonType.IdType[]; // 已选按钮ID
  };
}

interface MenuTreeSelectItem {
  id: number;
  label: string;
  menuType: 'M' | 'C' | 'F';  // 统一类型
  icon?: string;
  status?: string;
  perms?: string;            // 按钮节点使用
  children?: MenuTreeSelectItem[];
}
```

**移除的 API**：
- `fetchGetRoleButtonTreeSelect` - 冗余，按钮已在树中
- `fetchGetMenuButtons` - 冗余，后端已预加载

**请求量对比**：

| 场景 | 改前 | 改后 |
|------|------|------|
| 编辑角色（N模块） | 2N + M | 1 |

### 第三部分：前端组件改动

#### role-operate-drawer.vue

**移除内容**：
- `fetchGetRoleButtonTreeSelect` 导入和调用
- 循环中单独获取按钮权限的逻辑
- `startMenuLoading/stopMenuLoading` 的手动控制（简化）

**简化后的流程**：

```typescript
async function handleUpdateModelWhenEdit() {
  model.value = createDefaultModel();

  if (props.operateType === 'add') {
    return;
  }

  if (props.operateType === 'edit' && props.rowData) {
    startMenuLoading();
    Object.assign(model.value, jsonClone(props.rowData));

    await nextTick();
    await menuTreeRef.value?.refresh();

    // 单次调用获取完整权限树
    const authTree = await menuTreeRef.value?.loadRoleAuthTree(model.value.roleId!);

    if (authTree) {
      // 设置选中状态（合并菜单和按钮ID）
      const allKeys = [...authTree.menus, ...authTree.buttons];
      for (const module of menuTreeRef.value?.appList || []) {
        menuTreeRef.value?.setCheckedKeysByModule(module.appCode, allKeys);
      }
    }

    stopMenuLoading();
  }
}
```

#### menu-tree.vue

**移除内容**：
- `buttonsMap` 响应式状态
- `buttonsLoading` 状态
- `loadMenuButtons` 方法
- `buildTreeWithButtons` 方法
- `checkTreeHasButtons` 方法
- `getMenuIdsOfTypeC` 方法
- 两个 `watchEffect`（按钮加载逻辑）
- `clearButtonsCache` 方法

**新增方法**：

```typescript
// 加载角色完整权限树
async function loadRoleAuthTree(roleId: CommonType.IdType) {
  const { data, error } = await fetchGetRoleAuthTree(roleId);
  if (error) return null;

  // 设置各模块的菜单树数据
  for (const [module, tree] of Object.entries(data.trees)) {
    moduleMenuOptions[module] = [
      { id: 0, label: 'menu.root', icon: 'material-symbols:home-outline-rounded', children: tree }
    ];
  }

  return data.checkedKeys;
}

// 简化后的按钮ID获取
function getCheckedButtonIds(): CommonType.IdType[] {
  const buttonIds: CommonType.IdType[] = [];

  for (const module of appList.value) {
    const keys = moduleCheckedKeys[module.appCode] || [];
    for (const id of keys) {
      const node = findNodeById(id, moduleMenuOptions[module.appCode] || []);
      if (node && node.menuType === 'F') {
        buttonIds.push(id);
      }
    }
  }

  return [...new Set(buttonIds)];
}
```

#### 类型定义更新

新增 `frontend/src/typings/api/system.api.d.ts`：

```typescript
/** role auth tree response */
type RoleAuthTreeResponse = {
  trees: Record<string, MenuTreeSelectItem[]>;
  checkedKeys: {
    menus: CommonType.IdType[];
    buttons: CommonType.IdType[];
  };
};
```

### 第四部分：后端改动汇总

| 文件 | 改动内容 |
|------|----------|
| `backend/service/system/sys_menu.go` | 1. 移除 `convertMenuType` 函数<br>2. `menu.MenuType == "2"` → `menu.MenuType == "C"` |
| `backend/api/v1/system/sys_menu.go` | 新增 `GetRoleAuthTreeView` 接口 |
| `backend/service/system/sys_role.go` | 新增 `GetRoleAuthTree` 方法 |
| `backend/router/system/sys_menu.go` | 注册新路由 `/role/auth-tree/:id` |
| `backend/model/system/response/sys_role.go` | 新增 `RoleAuthTreeResponse` 结构体 |

**新增后端方法核心逻辑**：

```go
func (r *RoleService) GetRoleAuthTree(roleId int64) (SysResponse.RoleAuthTreeResponse, error) {
    // 1. 获取所有模块的应用列表
    // 2. 获取角色已关联的菜单ID和按钮ID
    // 3. 为每个模块构建完整菜单树（含按钮节点，类型统一为 M/C/F）
    // 4. 返回 trees 和 checkedKeys
}
```

## 验证计划

| 验证项 | 方法 |
|--------|------|
| 数据库迁移成功 | 查询 `sys_menus.menu_type` 全为 'M'/'C'/'F' |
| 菜单列表正常显示 | 进入菜单管理页面，检查树形结构 |
| 编辑角色不卡顿 | 点击编辑按钮，响应时间 < 500ms |
| 权限保存正确 | 编辑后保存，重新打开验证选中状态一致 |
| 类型统一 | 前端无兼容判断代码 |

## 注意事项

1. **部署顺序**：先执行 SQL 迁移，再部署后端代码，最后部署前端
2. **数据备份**：迁移前备份 `sys_menus` 表
3. **兼容性**：评估外部系统是否依赖旧 API

## 预期效果

- 性能：编辑角色响应时间从数秒降至 < 500ms
- 代码质量：移除约 150 行冗余代码
- 可维护性：类型统一，无兼容判断