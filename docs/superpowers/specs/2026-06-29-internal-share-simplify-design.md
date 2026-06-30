# 内部共享子系统简化重写设计

- **日期**：2026-06-29
- **状态**：待实现（brainstorming 已确认，待 writing-plans）
- **范围**：网盘「共享给用户 / 共享给部门（群组）」内部共享子系统（InternalShare）
- **前置 spec**：`2026-05-07-file-share-to-user-dept-design.md`、`2026-06-02-disk-share-redesign.md`（本次为其简化重写，部分推翻）
- **关联**：`docs/disk-security-performance-audit.md`、网盘安全审计记忆

---

## 1. 背景与动机

当前网盘存在**两套并行的共享系统**：

| 系统 | 对象 | 数据模型 |
|------|------|----------|
| **内部共享 InternalShare** | 站内用户/部门 | `FileShare` + `FileShareTarget` + `FileMount`(挂载点) + `File.ref_count` + COW + 双层权限 + 操作日志 + SSE |
| **公开分享 Share** | 外网任何人 | `Share`（短链+提取码+二维码+过期，`model/disk/disk_file.go:124`） |

本文档仅针对**内部共享子系统**。公开分享 `Share`、OnlyOffice、跨用户秒传**不在本次范围，保持不动**。

内部共享当前搬了企业网盘（Seafile/NextCloud 级）的几乎所有重型机制，叠加了若干**过度灵活**的设计，导致：

1. **语义二象性**：OnlyOffice 在线编辑共享文件=改同一份+版本历史（`FileHistory`），而「重传覆盖/重命名」却触发 COW 复制成副本——同一系统两种冲突语义。
2. **COW + ref_count 是 bug 集中区**：网盘安全审计中 P0/P1 大半（COW 长事务、ref_count 归零竞态、COW 副本不扣配额、配额双轨、asyncCleanup 竞态）都与这套副本机制相关。
3. **双层权限冗余**：`FileShare.Permissions`(默认) + `FileShareTarget.Permissions`(逐人覆盖) + 回退逻辑，企业网盘主流（Seafile/NextCloud/SharePoint）均为一个共享一套权限。
4. **部门物化快照不一致**：共享给部门时遍历成员给每人建 `FileMount`，之后成员加入/调离不联动。

**动机**：在定位为「企业协作网盘」的前提下，保留企业网盘体验骨架，一次性删除 COW / FileMount(物化) / ref_count / 双层权限 四套机制，将复杂度降低约 40%，并根除上述安全审计项。

---

## 2. 设计目标与非目标

### 目标
- 协作编辑统一为「改同一份」，冲突由 OnlyOffice + `FileHistory` 兜底
- 权限收敛为三档角色单层模型
- 「共享与我」独立列表 + 主文件树根运行时虚拟挂载，部门成员变动自动生效
- 「保存到我的网盘」改为物理独立副本
- 彻底移除 `FileMount`、`ref_count`、COW、双层权限

### 非目标
- 不做实时协同编辑（CRDT/OT），依赖 OnlyOffice
- 不引入文件级乐观锁 CAS（列入未来扩展）
- 不改造公开分享 `Share`、OnlyOffice、跨用户秒传、上传下载主链路
- 不做存量数据迁移（生产无存量，可推倒重建表结构）

---

## 3. 核心决策（已与产品方确认）

| # | 决策 | 理由 |
|---|------|------|
| 1 | 协作走 OnlyOffice，所有编辑=改同一份 | OnlyOffice 已是改同一份+版本历史，消除二象性，COW 整套移除 |
| 2 | 三档角色权限（viewer/editor/owner） | 对标主流企业网盘，用户选角色而非逐个勾权限；砍 Target 级权限 |
| 3 | 独立区 + 主树运行时虚拟挂载 | 消除 FileMount 物化与部门快照不一致，成员变动自动生效 |
| 4 | 「保存到网盘」= 物理复制 | 独立副本语义，砍 save FileMount 与整个 ref_count |
| 5 | 无存量数据，直接重建表结构 | 无需迁移脚本/兼容层 |

---

## 4. 后端设计

### 4.1 数据模型变更

| 动作 | 对象 | 位置 |
|------|------|------|
| **删整张表** | `disk_file_mounts`（`FileMount` 模型） | `model/disk/disk_file_mount.go` |
| **删字段** | `File.RefCount` / `File.IsCow` / `File.MountFileID` | `model/disk/disk_file.go`（File struct） |
| **删字段** | `FileShareTarget.Permissions` / `.Version` / `.MountName` | `model/disk/disk_file_share.go:28` |
| **改字段** | `FileShare.Permissions`（list）→ `FileShare.Role`（单值） | `model/disk/disk_file_share.go:11` |
| **精简保留** | `FileShare`：UserID + FileID + ShareType + **Role** + ExpireDate + NotifySent | 同上 |
| **精简保留** | `FileShareTarget`：仅 FileShareID + TargetType + TargetID | 同上 |
| **保留** | `FileShareAccessLog`（访问审计） | `model/disk/disk_file_share.go:43` |
| **保留简化** | `FileShareOperation`（协作写入审计，保留字段不扩） | `model/disk/disk_file_share.go:57` |
| **保留不动** | `FileHistory`（版本历史） | `model/disk/disk_file.go:107` |
| **新增** | `ShareRole` 枚举：`viewer` / `editor` / `owner` | `model/common/enumarate.go`（与 OperationPermission 同文件） |

### 4.2 权限角色映射（单层，存于 `FileShare.Role`，全体 target 共享）

```
viewer  → [DOWNLOAD]                              // 预览/下载
editor  → [DOWNLOAD, UPLOAD, PUT, DELETE]         // 协作增删改
owner   → editor 全部 + 管理共享(加人/删人/改角色/删共享)
```

- 发起人 `FileShare.UserID` 天然 = owner
- `checkShareFolderPermission`（`service/disk/disk_internal_share.go:1196`）：**保留其运行时 dept 匹配逻辑**，内部判断从「遍历 permissions 列表」改为「role → 权限集映射表」查表
- 砍掉 `getTargetPermissions`（:1626）/ `resolveTargetPermissions`（:1649）回退逻辑
- 砍掉 `UpdateTargetPermissions`（API + service + route）及 Target 级乐观锁；改为 `UpdateShareRole`（owner 改整个共享的 role）

**权限合并语义（消除歧义）**：
- 一个 `FileShare` 内所有 target 共享同一 `Role`，且 `ShareType` 决定 target 类型（全 user 或全 dept）。
- 同一文件可创建多条 `FileShare`（不同对象 / 不同 role）实现差异化授权，例如给 A 直接共享 editor、给部门 X 共享 viewer。
- 当一个用户同时命中多条共享（如既是 user-target 的 editor，又是某 dept-target 的 viewer），权限取**最高 role**（owner > editor > viewer），由 `checkShareFolderPermission` 在多命中时按 role 优先级裁决，而非「取第一条」。

### 4.3 协作写入（统一「改同一份」）

| 函数 | 处理 |
|------|------|
| `UploadToShareFolder`（:1281）/ `CreateShareSubFolder`（:1430） | **保留**，写源文件、扣分享者配额、记 `FileShareOperation`；权限校验改 role(editor) |
| `UpdateFileContent`（:148） | **删 TriggerCOW 分支**（:178），共享文件直接改源（需 editor） |
| `RenameFile`（:3935） | **删 TriggerCOW 分支**（:3975），共享文件直接改名（需 editor） |
| `service/disk/disk_share_cow.go` | **整文件删除**（TriggerCOW / copyPhysicalFile / asyncCleanupIfZeroRef） |

**冲突兜底**：OnlyOffice 在线编辑=改同一份 + `FileHistory` 版本；非 Office 文件的 PUT 重传=覆盖同一份，**最后写入胜出**（不加 CAS）。详见第 9 节已知取舍。

### 4.4 运行时虚拟挂载

- **「共享与我」列表**（`GetSharedWithMeList` :615）：数据源**不变**（已运行时 union 查 `FileShareTarget`：`target_type=user AND target_id=我` OR `target_type=dept AND target_id IN 我的deptIDs`），仅把返回字段 `permissions` 换成 `role`
- **主文件树根目录**（`getFileListInRealDir`，`service/disk/disk_file_list.go`）：当前合并 `FileMount` 挂载项 → 改为运行时 union 查「共享给我的文件夹」，虚拟挂到根展示；点进去走已有 `GetSharedFolderContents`（:900，保留）
- 砍掉 `service/disk/disk_share_mount.go` 的 `ShareMountService` 整个、`getDeptMemberIDs` 的**物化挂载用法**
- **保留** `getDeptMemberIDs`（:409）：SSE 通知（`collectTargetUserIDs` :299）、权限缓存失效（`InvalidateSharePermissionCache`）仍需把部门展开成具体人

### 4.5 「保存到我的网盘」改物理复制

- `SaveToMyDrive`（:318）/ `BatchSaveToDrive`：当前 `CreateSaveMount`（软引用）→ 改调现有 `CopyFile` / `copyFolderRecursive`，生成**独立 File 记录**，完全脱离共享
- 副本独立：不随源更新、删共享不影响它
- 权限门槛：需 `DOWNLOAD`（任意 role 均可，viewer 亦然）
- 砍掉 `resolveNameConflict` 中针对 `FileMount` 的分支，复用普通文件名冲突逻辑

---

## 5. 前端设计（Vue3 + TS + Naive UI）

| 组件/文件 | 改造 |
|-----------|------|
| `views/disk/modules/share-to-user.vue` / `share-to-dept.vue` | 权限多选勾(DOWNLOAD/UPLOAD/PUT/DELETE) → **角色单选**(查看/编辑/所有者) |
| `views/disk/modules/share-permission-checker.vue` | 改造为 role 选择器，或并入上方后删除 |
| `views/disk/modules/share-target-item.vue` | 显示 permissions 列表 → 显示 role 标签 |
| `views/disk/modules/share-dialog.vue` | 提交字段 `permissions[]` → `role` |
| `views/shared-with-me/modules/share-list-page.vue`(1280行) | my-shared 权限管理改 role；shared-with-me 展示改 role；**删 isMounted/mountId 分支** |
| `views/disk/modules/save-to-drive-dialog.vue` | 文案改「保存为独立副本」 |
| 类型定义(`typings/`) | `OperationPermission[]` → `ShareRole`；`SharedWithMeItem` 删 `isMounted/MountID/permissions` 加 `role`；前后端类型一一对应（项目红线，禁 any） |
| API(`fetch*`) | `fetchCreateInternalShare` 传 `role`；删 `fetchUpdateTargetPermissions` → 改 `fetchUpdateShareRole`；`fetchRemoveShareTarget` 保留 |

`group-share` 与 `shared-with-me` 继续共用 `share-list-page.vue`，仅 `share-type` 参数区别（user/dept），结构不变。

---

## 6. 完整删除清单

### 后端
- `service/disk/disk_share_cow.go` — **整文件**
- `service/disk/disk_share_mount.go` — **整文件**（`ShareMountService` 全部方法）
- `model/disk/disk_file_mount.go` — `FileMount` 模型 + 表 `disk_file_mounts`
- `File` 字段：`RefCount` / `IsCow` / `MountFileID`
- `FileShareTarget` 字段：`Permissions` / `Version` / `MountName`
- `getTargetPermissions` / `resolveTargetPermissions`
- `UpdateTargetPermissions`（API `api/v1/disk/disk_internal_share.go:29` + service :413 + route `router/disk/disk_internal_share.go`）
- `UpdateFileContent:178` / `RenameFile:3975` 的 `TriggerCOW` 调用分支
- `CreateInternalShare`（:160）内的 `CreateShareMounts` 调用（:280）
- `RemoveShareTarget`（:453）内 `RemoveSingleMount` / `BatchRemoveMountsByUserIDs` 的 FileMount 维护分支

### 前端
- `isMounted` / `mountId` 全部逻辑与字段
- 权限多选 UI（`share-permission-checker.vue` 旧实现）
- 冗余的 `OperationPermission[]` permissions 类型与传输字段

---

## 7. 影响面（同步改造的关联点）

| 点 | 处理 |
|----|------|
| `ResolveFileOwner`（:36）/ `checkFileAccessByInternalShareWithAction` | `requiredPermission` 判断改 role 映射 |
| `checkInternalSharePermissionWithAction`（preview/download 包，`service/disk/disk_preview.go`） | 改 role |
| `CleanExpiredShares`（:1027） | 简化（无 FileMount 可清） |
| `getFileListInRealDir`（`service/disk/disk_file_list.go`） | 挂载项来源改运行时 union |
| 公开分享 `Share` / OnlyOffice / 跨用户秒传 / 上传下载主链路 | **不动** |

---

## 8. 测试策略（TDD，覆盖率 ≥80%）

### 后端单测/集成
- **权限**：viewer 拒编辑、editor 拒管理（加人/删人/改 role）、owner 全能；部门成员加入→主树可见、调离→不可见
- **协作写入**：editor 上传/改名/更新共享文件=改源并扣分享者配额、记 Operation；viewer 调用被拒
- **save 物理复制**：副本独立、删共享后副本仍在、源更新不影响副本
- **运行时挂载**：`GetSharedWithMeList` 与主树根虚拟挂载均不依赖 `FileMount`
- **删除清理**：删共享/删 target 后无残留（已无 FileMount）
- **回归**：公开分享、OnlyOffice 编辑、普通上传下载不受影响

### 前端
- role 选择器交互（单选、回显）
- shared-with-me / my-shared 列表 role 展示
- `pnpm typecheck` + `pnpm lint` 必过（零 any、前后端类型一一对应）

---

## 9. 错误处理

- 权限不足：统一返回明确错误（「无 X 权限」），不泄漏共享是否存在
- 协作写入：保留现有「物理文件 + DB + 配额」事务一致性回滚（落地失败删物理文件、建库失败退配额）
- save 物理复制：失败回滚（已复制物理文件清理 + 不建 File 记录）
- 路径边界：保留 `validatePathInShareRange`（:1258）

---

## 10. 已知取舍与未来扩展

### 当前取舍（有意为之，非缺陷）
- **非 Office 文件并发重传覆盖无冲突检测**，最后写入胜出。靠 `FileHistory` 版本可回溯。符合「OnlyOffice 为主力协作」决策。

### 未来扩展（不在本次范围）
- **文件级乐观锁 CAS**：若日后需更强并发冲突保护，按审计建议在 `File` 加 `Version` 字段 + CAS 更新（后写冲突报错让用户选择）。本次预留，不实现。
- **跨用户去重**（共享 storage_path + RefCount 模型）：本次反而移除 ref_count，存储成本换简化；未来如需再评估。

---

## 11. 实施范围

采用**方案 A：一次性重写内部共享子系统**。理由：无存量可推倒，避免「方案 B 渐进」中 FileMount/ref_count 在阶段间半保留造成的中间态混乱。

实施顺序（供 writing-plans 细化）：
1. 后端模型层（删表/字段 + 新增 ShareRole + 角色映射）
2. 后端权限校验层（role 替换 permissions 判断）
3. 后端协作写入（删 COW 分支 + 虚拟挂载 + save 物理复制）
4. 删除 COW / FileMount / share_mount 整文件与残留引用
5. 前端类型 + API + 组件改造
6. 测试 + typecheck/lint/build 验证
7. 用户测试通过后提交（遵循 no-auto-commit）
