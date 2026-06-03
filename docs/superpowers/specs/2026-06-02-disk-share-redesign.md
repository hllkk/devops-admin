# 网盘共享系统重新设计

> 日期: 2026-06-02
> 状态: 已确认，待实施
> 方案: A — 虚拟挂载点

## 1. 需求决策

| 决策项 | 选择 |
|--------|------|
| 保存到我的网盘 | 硬链接 + COW（写时复制） |
| 权限模型 | 4 个细粒度权限位（下载/上传/编辑/删除） |
| 共享目标 | 用户 + 部门 |
| 通知机制 | 页面徽章 + SSE 实时推送 |
| 文件夹共享 | 动态同步，新增文件自动可见 |
| 接受流程 | 自动接受，无需确认 |

## 2. 数据库模型

### 2.1 disk_files 扩展

新增字段:

| 字段 | 类型 | 说明 |
|------|------|------|
| ref_count | INT | 引用计数，默认 1 |
| storage_hash | VARCHAR(64) | 文件内容 SHA-256 哈希 |
| is_cow | BOOL | 是否已触发过写时复制 |

引用计数规则:
- 原始创建者 ref_count = 1
- 每多一次"保存到我的网盘" → ref_count += 1
- 用户删除文件 → ref_count -= 1
- ref_count = 0 时异步清理物理文件

### 2.2 disk_file_shares 调整

保留字段: file_share_id, user_id, file_id, share_type, permissions, expire_date, created_at, updated_at

移除字段: status（不再需要接受/拒绝）

新增字段:

| 字段 | 类型 | 说明 |
|------|------|------|
| notify_sent | BOOL | 是否已发送 SSE 通知 |

### 2.3 disk_file_share_targets 调整

保留字段: id, file_share_id, target_type, target_id, permissions

移除字段: status

新增字段:

| 字段 | 类型 | 说明 |
|------|------|------|
| mount_name | VARCHAR(255) | 挂载点显示名称 |

### 2.4 disk_file_mounts（新建）

```sql
CREATE TABLE disk_file_mounts (
  mount_id       BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id        BIGINT NOT NULL,
  source_file_id BIGINT NOT NULL,
  parent_id      BIGINT,
  mount_name     VARCHAR(255),
  mount_type     ENUM('share','save'),
  share_id       BIGINT,
  created_at     DATETIME,

  INDEX idx_user_parent (user_id, parent_id),
  INDEX idx_source_file (source_file_id)
);
```

- mount_type='share': 共享自动挂载，删除共享时自动清除
- mount_type='save': 用户主动保存，独立于共享生命周期

### 2.5 关系总览

```
disk_files (物理文件)
  ├── ref_count: 引用计数
  ├── storage_hash: 去重
  │
  ├── disk_file_mounts (虚拟挂载)
  │     source_file_id → disk_files
  │
  └── disk_file_shares (共享关系)
        ├── disk_file_share_targets (目标)
        ├── disk_file_share_access_logs (访问日志)
        └── disk_file_share_operations (操作记录)
```

## 3. 核心流程

### 3.1 共享对话框交互

对话框使用 TAB 切换"共享给用户"和"共享给部门"。

**用户 Tab 流程:**
1. 搜索用户，多选
2. 勾选权限（下载/上传/编辑/删除）
3. 点击"确认授权" → API 调用
4. Tab 内展示已授权用户列表
5. 列表支持 [修改] 权限和 [移除] 授权
6. 可多次添加不同用户

**部门 Tab 流程:** 同构，搜索源改为部门。

### 3.2 批量授权后端处理

```
POST /api/v1/share/internal/create
1. 校验文件归属（共享者必须是 owner）
2. 创建 disk_file_shares 记录
3. 批量创建 disk_file_share_targets（去重跳过已授权）
4. 为每个目标创建 disk_file_mounts (mount_type='share')
5. dept 类型 → 查部门所有成员，每人创建 mount
6. disk_files.ref_count += N
7. SSE 推送给所有在线目标用户
```

### 3.3 共享给我的展示

GET /api/v1/share/internal/received 合并用户共享和部门共享，去重返回。

返回数据:
```json
{
  "shareId": 1,
  "fileId": 100,
  "fileName": "项目资料",
  "fileType": "folder",
  "sharedBy": { "userId": 1, "userName": "张三", "avatar": "..." },
  "permissions": ["download", "upload", "edit"],
  "sharedAt": "2026-06-01T10:00:00Z",
  "expireDate": null,
  "isMounted": true,
  "mountId": null
}
```

页面功能:
- 筛选栏: 类型筛选 + 搜索 + 排序
- 列表/网格视图切换
- 状态标记: 未保存 / 已保存 / 共享已失效 / 已过期
- 批量操作: 批量保存到网盘 / 批量下载

### 3.4 查看共享文件夹

点击"查看"复用文件浏览器组件:
- 后端通过 parent_id 递归查原文件树（动态同步零延迟）
- 根据权限显示操作按钮
- 子文件夹可继续点击进入

### 3.5 保存到我的网盘（COW）

保存流程:
```
1. 用户选择目标目录
2. 创建 disk_file_mounts (mount_type='save')
3. disk_files.ref_count += 1
4. 文件夹类型: 只在文件夹级别创建 mount，子文件通过 parent_id 递归查询
```

COW 触发（用户编辑已保存文件时）:
```
1. 检查 ref_count
   - ref_count == 1 → 直接修改
   - ref_count > 1 → 触发 COW
2. COW: 复制物理文件 → 创建新 disk_file 记录
3. 原文件 ref_count -= 1
4. 用户 mount 指向新文件
5. 原文件 ref_count == 0 → 异步清理
```

懒复制策略: 大文件夹共享时只在文件夹级别挂载，子文件按需 COW。

### 3.6 取消共享

```
1. 软删除 disk_file_shares
2. 删除所有 disk_file_share_targets
3. 删除 mount_type='share' 的挂载（save 类型不受影响）
4. disk_files.ref_count -= N
5. SSE 通知受影响用户
6. ref_count 归零 → 异步清理物理文件
```

## 4. API 设计

```
POST   /api/v1/share/internal/create              批量授权
GET    /api/v1/share/internal/:fileId/targets      获取已授权列表
PUT    /api/v1/share/internal/target/:targetId/permissions  修改权限
DELETE /api/v1/share/internal/target/:targetId     移除单个授权
DELETE /api/v1/share/internal/:shareId             取消整个共享
GET    /api/v1/share/internal/received             共享给我的列表
GET    /api/v1/share/internal/:shareId/files       查看共享文件夹内容
POST   /api/v1/share/internal/save-to-drive        保存到我的网盘
GET    /api/v1/share/internal/my-shares            我发起的共享列表
```

## 5. 前端组件架构

### 5.1 共享对话框

```
disk/modules/share-dialog/
├── index.vue                    ← 对话框容器，Tab 切换 (~80行)
├── share-to-user.vue           ← 用户 Tab (~200行)
├── share-to-dept.vue           ← 部门 Tab (~200行)
├── share-target-item.vue       ← 已授权列表单行 (~60行)
├── share-permission-checker.vue ← 权限复选框组 (~40行)
└── share-user-search.vue       ← 用户搜索 (~50行)
```

### 5.2 共享给我的页面

```
shared-with-me/
├── index.vue                    ← 页面容器 (~100行)
└── components/
    ├── share-filter-bar.vue     ← 筛选栏
    ├── share-file-card.vue      ← 文件卡片/列表项
    ├── share-file-grid.vue      ← 网格视图
    ├── share-file-list.vue      ← 列表视图
    ├── share-batch-action-bar.vue ← 批量操作栏
    └── share-status-tag.vue     ← 状态标签
```

### 5.3 保存到网盘

```
disk/modules/save-to-drive-dialog.vue  ← 目录选择器 (~150行)
```

## 6. 后端架构

```
api/v1/disk/
└── disk_internal_share.go       ← Handler

service/disk/
├── disk_internal_share.go       ← 业务逻辑编排
├── disk_share_permission.go     ← 权限校验（独立模块）
├── disk_share_cow.go            ← COW 写时复制
└── disk_share_mount.go          ← 挂载点管理

model/disk/
├── disk_file_share.go           ← 共享 CRUD
├── disk_file_share_target.go    ← 共享目标 CRUD
├── disk_file_mount.go           ← 挂载点 CRUD
└── disk_file.go                 ← 扩展 ref_count/storage_hash
```

## 7. 安全设计

### 7.1 权限矩阵

| 操作 | 下载权限 | 上传权限 | 编辑权限 | 删除权限 |
|------|---------|---------|---------|---------|
| 预览 | 不需要 | - | - | - |
| 下载 | 需要 | - | - | - |
| 上传 | - | 需要 | - | - |
| 重命名/移动 | - | - | 需要 | - |
| 修改内容 | - | - | 需要 | - |
| 删除 | - | - | - | 需要 |
| 保存到网盘 | 需要 | - | - | - |

### 7.2 安全规则

- 共享者只能共享自己拥有的文件
- 被共享者不能二次共享原始文件（无 share 权限）
- 保存到网盘后拥有新文件完整权限，可独立共享
- 被共享者删除 = 只删 mount 记录，不影响原始文件
- 共享者删除原始文件 → share 挂载失效，save 挂载不受影响
- 过期共享由定时任务每小时扫描清理

### 7.3 操作审计

所有共享操作写入 disk_file_share_access_logs: view, download, upload, edit, delete, save_to_drive。

## 8. 性能设计

### 8.1 索引

```sql
ALTER TABLE disk_file_share_targets ADD INDEX idx_target_user (target_type, target_id);
ALTER TABLE disk_file_mounts ADD INDEX idx_user_parent (user_id, parent_id);
ALTER TABLE disk_file_mounts ADD INDEX idx_source_file (source_file_id);
ALTER TABLE disk_files ADD INDEX idx_ref_count (ref_count);
ALTER TABLE disk_file_shares ADD INDEX idx_expire (expire_date);
```

### 8.2 Redis 缓存

| Key | 内容 | TTL | 失效条件 |
|-----|------|-----|---------|
| share_perm:{userID}:{fileID} | 权限列表 | 5min | 授权变更时 DEL |
| share_received:{userID}:page:{n} | 共享列表分页 | 2min | 新共享/取消时 DEL |
| share_targets:{fileID}:type:{t} | 已授权目标 | 5min | 授权/移除时 DEL |
| file_ref:{fileID} | ref_count | 10min | mount 变更时 INCR/DECR |

### 8.3 大文件夹优化

- 文件夹级别挂载，不逐文件创建 mount
- 子文件通过 parent_id 递归查询原文件树
- COW 懒复制: 只在实际修改时触发，非全量复制

### 8.4 并发安全

- ref_count 使用 `UPDATE SET ref_count = ref_count + 1` 原子操作
- ref_count = 0 清理加分布式锁
- COW 竞态: 各自创建副本，不会丢失数据

## 9. SSE 通知事件

| 事件 | 触发时机 | 数据 |
|------|---------|------|
| new_share | 有人共享文件给我 | shareId, fileName, sharedBy, perms |
| share_cancelled | 共享被取消 | shareId, fileName, cancelledBy |
| share_expired | 共享过期 | shareId, fileName |
| permission_change | 权限被修改 | shareId, fileName, newPerms |
| shared_file_update | 共享文件夹内有变更 | shareId, fileName, action, operator |

## 10. 与公共链接共享的关系

两者独立运行，互不影响。同一文件可同时内部共享 + 公共链接共享。

内部共享走 disk_file_shares + disk_file_mounts，公共链接走 disk_shares，存储和权限体系各自独立。

## 11. 实施范围

### 数据库
- disk_files 新增 ref_count, storage_hash, is_cow
- disk_file_shares 移除 status，新增 notify_sent
- disk_file_share_targets 移除 status，新增 mount_name
- 新建 disk_file_mounts 表

### 后端
- 新增 disk_share_permission.go, disk_share_cow.go, disk_share_mount.go
- 重构 disk_internal_share.go handler 和 service
- 新增 9 个 API 端点

### 前端
- 拆分 share-dialog.vue (1100行 → 6 个组件)
- 拆分 shared-with-me (1125行 → 7 个组件)
- 新增 save-to-drive-dialog.vue
- SSE 新增 5 个事件类型订阅
