# 前端类型定义修复实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 MenuTreeSelectItem 类型定义，添加缺失字段使前后端类型一致。

**Architecture:** 修改前端类型定义文件，补充 menuType、perms、status、hiddenInMenu、icon、module 字段，与后端 Go 结构体字段对应。

**Tech Stack:** TypeScript + Vue3

---

## File Structure

| 文件 | 改动类型 | 职责 |
|------|----------|------|
| `frontend/src/typings/api/system.api.d.ts:87-91` | 修改 | 补充 MenuTreeSelectItem 类型缺失字段 |

---

## Task 1: 修改 MenuTreeSelectItem 类型定义

**Files:**
- Modify: `frontend/src/typings/api/system.api.d.ts:87-91`

- [ ] **Step 1: 替换 MenuTreeSelectItem 类型定义**

修改 `frontend/src/typings/api/system.api.d.ts` 第 87-91 行：

将：
```typescript
    /** menu tree select item */
    type MenuTreeSelectItem = {
      id: CommonType.IdType;
      label: string;
      children?: MenuTreeSelectItem[];
    };
```

改为：
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

- [ ] **Step 2: 运行 TypeScript 类型检查**

```bash
cd frontend && pnpm typecheck
```

Expected: 无类型错误

- [ ] **Step 3: 运行 lint 检查**

```bash
cd frontend && pnpm lint
```

Expected: 无 lint 错误

- [ ] **Step 4: Commit 类型定义修复**

```bash
git add frontend/src/typings/api/system.api.d.ts
git commit -m "fix: complete MenuTreeSelectItem type definition with backend fields"
```

---

## Task 2: 验证修复效果

- [ ] **Step 1: 运行完整前端检查**

```bash
cd frontend && pnpm typecheck && pnpm lint
```

Expected: 全部通过

- [ ] **Step 2: 查看所有改动**

```bash
git status
```

Expected: 所有改动已提交

- [ ] **Step 3: 查看提交历史**

```bash
git log --oneline -5
```

Expected: 包含本次 commit

---

## Self-Review Checklist

**Spec coverage:**
- ✅ menuType 字段: Task 1
- ✅ icon 字段: Task 1
- ✅ status 字段: Task 1
- ✅ hiddenInMenu 字段: Task 1
- ✅ module 字段: Task 1
- ✅ perms 字段: Task 1
- ✅ TypeScript 验证: Task 1, Task 2
- ✅ lint 验证: Task 1, Task 2

**Placeholder scan:**
- 无 TBD、TODO、未完成部分
- 所有步骤有具体代码或命令

**Type consistency:**
- menuType 使用 `'M' | 'C' | 'F'` 与 Api.System.MenuType 保持一致
- status 使用 `Common.EnableStatus` 与现有类型保持一致
- 其他字段类型与后端 Go 结构体字段对应