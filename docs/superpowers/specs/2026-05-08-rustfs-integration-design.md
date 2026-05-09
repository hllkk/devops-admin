# RustFS 对接设计文档

> 日期: 2026-05-08
> 状态: 待实现
> 范围: RustFS S3 存储对接 + 本地存储迁移工具

## 背景

当前 DevOps Admin 网盘功能默认使用本地磁盘存储，通过 `StorageProvider` 接口抽象存储层。RustFSProvider 已有 stub 框架（枚举、配置、空方法体），但缺少实际实现。

RustFS 是 Rust 编写的高性能分布式对象存储，100% S3 API 兼容，Apache 2.0 许可证，4KB 小对象比 MinIO 快 2.3 倍，读取最高 323 GB/s。适合作为网盘的大文件存储后端。

## 方案选择

**选定：方案 A — RustFS 专用 Provider**

使用 `minio-go` SDK 在 `RustFSProvider` 中实现全部 `StorageProvider` 接口方法。不与 AWS S3 Provider 合并，保持独立演进能力。

理由：
- 项目已有 RustFS 枚举和配置，改动最小
- RustFS 大文件分片可用 `ComposeObject` 服务端合并
- 独立实现避免 S3 兼容差异导致的边界问题

## 架构

```
API Layer (disk_file.go / disk_download.go)
    │
    │  provider := GetStorageProvider("rustfs")
    │
    ▼  StorageProvider 接口
RustFSProvider
    ├── client *minio.Client        ← RustFS S3 连接
    ├── chunkTempDir string          ← 分片暂存目录
    ├── bucketName string            ← 配置的桶名
    ├── basePath string              ← 桶内路径前缀
    │
    │  方法:
    │    BuildPath        → 构造 S3 object key
    │    ReadDirectory    → ListObjectsV2 + delimiter="/"
    │    GetFileInfo      → StatObject
    │    DownloadFile     → GetObject (stream)
    │    GetFileURL       → PresignedGetObject (有效期1小时)
    │    MergeChunks      → ComposeObject (S3 服务端合并)
    │    MoveToTrash      → CopyObject + RemoveObject
    │    RestoreFromTrash → CopyObject + RemoveObject
    │    DeleteFromTrash  → RemoveObject
    │
    ▼  minio-go SDK
RustFS Server (Docker, 同服务器)
    Endpoint: localhost:9000
    Bucket: devops-admin
```

## 数据流

### 大文件分片上传

```
前端分片上传 → 后端暂存到本地 chunk 临时目录
             → 所有分片上传完毕
             → RustFSProvider.MergeChunks:
                 用 minio-go ComposeObject 在 S3 服务端合并分片
                 → 合并成功后删除本地暂存分片
```

保留本地暂存原因：分片逐个上传，可能跨时间，不适合直接产生大量 incomplete multipart。ComposeObject 一次性在 S3 端完成，无需下载到本地再上传。

### 文件下载

```
前端请求下载 → 后端验证权限 → 返回预签名 URL（1小时有效）
                              → 前端直接从 RustFS 下载

内部调用场景 → DownloadFile 返回 io.ReadCloser（后端中转）
```

### 回收站

S3 无原生回收站，通过路径模拟：

```
正常文件: bucket/basePath/{userID}/files/xxx.pdf
回收站:  bucket/basePath/{userID}/trash/xxx.pdf

MoveToTrash:       CopyObject(files→trash) + RemoveObject(files)
RestoreFromTrash:  CopyObject(trash→files) + RemoveObject(trash)
DeleteFromTrash:   RemoveObject(trash)
```

## 迁移工具

### 架构

```
disk_migration.go — MigrationService

sourceProvider: LocalStorageProvider
targetProvider: RustFSProvider
db: *gorm.DB

方法:
  EstimateMigration(userID)        → 文件数量+总大小
  MigrateUserFiles(userID, onProgress) → 逐文件迁移
  GetMigrationStatus(userID)       → 查询进度
  RetryFailed(userID)              → 重试失败项
```

### 迁移流程

1. 遍历 `disk_files` 表中该用户所有文件记录
2. 从本地 `StoragePath` 读取文件流
3. 上传到 RustFS 对应的 object key
4. 更新 `disk_files` 表的 `storage_path` 和 `storage_type` 字段
5. 失败的文件记录到 `disk_migrations` 表，支持重试
6. 迁移完成后可切换默认存储类型为 rustfs

### 数据模型

```sql
CREATE TABLE disk_migrations (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    file_id BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending/running/success/failed
    source_path VARCHAR(500),
    target_path VARCHAR(500),
    error_message TEXT,
    started_at DATETIME,
    completed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

## 错误处理

| 场景 | 处理方式 |
|------|---------|
| RustFS 连接失败 | 启动时健康检查 + 运行时重试（3次，指数退避） |
| Bucket 不存在 | 初始化时自动创建 |
| 文件上传失败 | 返回明确错误 + 记录日志 |
| 迁移中断 | `disk_migrations` 表记录进度，支持断点续迁 |
| 预签名 URL 过期 | 前端重新请求 |
| 分片合并失败 | 清理已上传的部分对象 + 保留本地暂存 + 返回错误 |

## 文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `service/disk/disk_provider.go` | 修改 | 用 minio-go 重写 RustFSProvider 全部方法 |
| `service/disk/disk_migration.go` | 新增 | 本地→RustFS 迁移服务 |
| `api/v1/disk/disk_migration.go` | 新增 | 迁移 API handler（触发/进度/重试） |
| `router/disk/disk_migration.go` | 新增 | 迁移路由注册 |
| `model/disk/disk_migration.go` | 新增 | 迁移记录模型 + 自动迁移 |
| `go.mod` / `go.sum` | 修改 | 添加 `github.com/minio/minio-go/v7` 依赖 |
| `docs/rustfs-integration.md` | 新增 | 使用文档（部署配置、API 说明、迁移步骤） |

## 配置示例

```yaml
# config.yaml
rustfs:
  endpoint: "localhost:9000"
  access-key-id: "devops-admin"
  secret-access-key: "your-secret-key"
  bucket-name: "devops-admin"
  use-ssl: false
  base-path: "/disk"

system:
  oss-enable: true
  oss-type: "rustfs"    # 切换为 RustFS 存储
```

## API 设计

### 迁移相关 API

```
POST   /api/v1/disk/migration/start     → 开始迁移（可指定 userID 或全部）
GET    /api/v1/disk/migration/status     → 查询迁移进度
POST   /api/v1/disk/migration/retry      → 重试失败项
```

### 迁移请求/响应

```go
// StartMigrationReq 开始迁移请求
type StartMigrationReq struct {
    UserID int64 `json:"userId"` // 空值=迁移所有用户
}

// MigrationStatusResp 迁移状态响应
type MigrationStatusResp struct {
    Total     int64 `json:"total"`
    Success   int64 `json:"success"`
    Failed    int64 `json:"failed"`
    Pending   int64 `json:"pending"`
    Running   int64 `json:"running"`
}

// RetryMigrationReq 重试迁移请求
type RetryMigrationReq struct {
    UserID int64 `json:"userId"` // 空值=重试所有失败项
}
```

## 安全考量

- AccessKey/SecretKey 通过配置文件管理，不硬编码
- 预签名 URL 有效期限制为 1 小时
- 文件路径构造包含用户隔离（basePath/{userID}/...）
- RustFS 服务仅监听内网端口，不暴露公网
- 迁移 API 限制仅管理员可调用

## 测试策略

- 单元测试：使用 mock S3 server（如 minio-go 的 mock 接口）
- 集成测试：本地 Docker 启动 RustFS 实例，测试完整上传/下载/删除流程
- 迁移测试：构造测试文件 → 迁移 → 验证 RustFS 中文件完整性（MD5 校验）

## 后续扩展

- Seafile WebDAV 对接：后续通过 WebDAV 协议读取 Seafile 文件，写入 RustFS
- CDN 集成：RustFS S3 协议天然支持 CDN 回源
- 多租户：按用户分配不同 bucket 或 path prefix
