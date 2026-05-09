# RustFS 存储对接使用文档

## 概述

DevOps Admin 网盘系统已集成 RustFS 作为对象存储后端。RustFS 是 Rust 编写的高性能分布式对象存储，100% S3 API 兼容，Apache 2.0 许可证。

### 优势

| 对比项 | 本地存储 | RustFS |
|--------|---------|--------|
| 可靠性 | 单点故障，磁盘损坏数据丢失 | 多副本/纠删码，自动容灾 |
| 扩展性 | 受单机磁盘限制 | 水平扩展，按需加节点 |
| 大文件 | 本地 IO 瓶颈 | S3 Multipart Upload，服务端合并 |
| 备份 | 需手动定时备份 | 内置数据冗余 |
| 多实例 | 不支持 | 统一存储层，多后端实例 |
| 下载 | 后端中转流量 | 预签名 URL 直出 |

---

## 1. RustFS Docker 部署

### 1.1 创建 Docker Compose 文件

```yaml
# docker-compose.rustfs.yml
version: "3.8"
services:
  rustfs:
    image: rustfs/rustfs:latest
    container_name: rustfs
    ports:
      - "9000:9000"   # S3 API 端口
      - "9001:9001"   # Web 管理控制台
    environment:
      RUSTFS_ROOT_USER: "devops-admin"
      RUSTFS_ROOT_PASSWORD: "your-strong-password"
      RUSTFS_BROWSER: "on"
    volumes:
      - rustfs-data:/data
    restart: unless-stopped

volumes:
  rustfs-data:
```

### 1.2 启动 RustFS

```bash
docker compose -f docker-compose.rustfs.yml up -d
```

### 1.3 验证服务

```bash
# 检查 API 端口
curl http://localhost:9000/minio/health/live

# 访问 Web 控制台
# 浏览器打开 http://localhost:9001
```

### 1.4 创建存储桶

通过 Web 控制台 (http://localhost:9001) 或 mc 命令行创建桶：

```bash
# 安装 mc 客户端
wget https://dl.min.io/client/mc/release/linux-amd64/mc
chmod +x mc && mv mc /usr/local/bin/

# 配置别名
mc alias set rustfs http://localhost:9000 devops-admin your-strong-password

# 创建桶
mc mb rustfs/devops-admin
```

---

## 2. 后端配置

### 2.1 修改配置文件

编辑 `backend/conf/config.yaml`：

```yaml
rustfs:
  endpoint: "localhost:9000"
  access-key-id: "devops-admin"
  secret-access-key: "your-strong-password"
  bucket-name: "devops-admin"
  use-ssl: false
  base-path: "/disk"

system:
  oss-enable: true
  oss-type: "rustfs"    # 切换为 RustFS 存储
```

### 2.2 配置项说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `endpoint` | RustFS 服务地址 | 无 |
| `access-key-id` | 访问密钥 ID | 无 |
| `secret-access-key` | 访问密钥 | 无 |
| `bucket-name` | 存储桶名称 | 无 |
| `use-ssl` | 是否使用 HTTPS | false |
| `base-path` | 桶内路径前缀 | 空 |
| `oss-enable` | 是否启用 OSS | false |
| `oss-type` | 存储类型 | local |

### 2.3 重启后端

```bash
cd backend
go build -o devops-admin .
./devops-admin
```

---

## 3. API 接口

### 3.1 存储迁移接口

#### 估算迁移工作量

```
GET /api/v1/disk/storage-migration/estimate?userId=0
Authorization: Bearer <token>
```

响应：
```json
{
  "code": 0,
  "data": {
    "fileCount": 150,
    "totalSize": 5368709120,
    "totalSizeMB": 5120,
    "totalSizeGB": 5
  },
  "msg": "成功"
}
```

#### 开始迁移

```
POST /api/v1/disk/storage-migration/start
Authorization: Bearer <token>
Content-Type: application/json

{
  "userId": 0    // 0 = 全量迁移，指定用户ID则仅迁移该用户
}
```

响应：
```json
{
  "code": 0,
  "data": {
    "total": 150,
    "success": 148,
    "failed": 2,
    "skipped": 0,
    "duration": "45.2s",
    "errors": [
      {
        "fileId": 123,
        "name": "large-file.zip",
        "path": "/documents/",
        "error": "打开本地文件失败: file does not exist"
      }
    ]
  },
  "msg": "存储迁移完成"
}
```

#### 查询迁移状态

```
GET /api/v1/disk/storage-migration/status?userId=0
Authorization: Bearer <token>
```

响应：
```json
{
  "code": 0,
  "data": {
    "total": 150,
    "success": 148,
    "failed": 2,
    "pending": 0,
    "running": 0
  },
  "msg": "成功"
}
```

#### 重试失败项

```
POST /api/v1/disk/storage-migration/retry
Authorization: Bearer <token>
Content-Type: application/json

{
  "userId": 0    // 0 = 重试所有失败项
}
```

---

## 4. 存储迁移指南

### 4.1 迁移前检查

1. 确保 RustFS 服务已启动且可访问
2. 确认存储桶已创建
3. 确认配置文件中 `oss-type` 仍为 `local`（迁移完成后再切换）
4. 先调用 `/estimate` 接口估算工作量

### 4.2 执行迁移

```bash
# 1. 估算
curl -X GET "http://localhost:8888/api/v1/disk/storage-migration/estimate" \
  -H "Authorization: Bearer <token>"

# 2. 开始迁移（建议先测试单个用户）
curl -X POST "http://localhost:8888/api/v1/disk/storage-migration/start" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"userId": 1}'

# 3. 查看状态
curl -X GET "http://localhost:8888/api/v1/disk/storage-migration/status" \
  -H "Authorization: Bearer <token>"

# 4. 如有失败项，重试
curl -X POST "http://localhost:8888/api/v1/disk/storage-migration/retry" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"userId": 1}'
```

### 4.3 迁移后切换

1. 确认所有文件迁移成功（`/status` 接口 failed=0）
2. 修改 `config.yaml`：`oss-type: "rustfs"`
3. 重启后端服务
4. 验证文件上传、下载、删除功能正常

---

## 5. 存储路径映射

### 本地存储路径

```
/data/disk/{userID}/{path}/{fileName}
```

### RustFS S3 路径

```
Bucket: devops-admin
Object Key: disk/{userID}/{path}/{fileName}
```

### 回收站路径

```
正常文件: disk/{userID}/files/{path}
回收站:  disk/{userID}/trash/{path}
```

---

## 6. 故障排查

### RustFS 连接失败

```
错误: RustFS endpoint 未配置
解决: 检查 config.yaml 中 rustfs.endpoint 是否填写

错误: 创建 RustFS 客户端失败
解决: 检查 endpoint 是否可达，确认端口正确

错误: RustFS 连接检查失败
解决: 确认 RustFS 容器正在运行: docker ps | grep rustfs
```

### 文件上传失败

```
错误: 上传文件到 RustFS 失败
解决:
  1. 检查 access-key-id 和 secret-access-key 是否正确
  2. 检查 bucket 是否存在
  3. 检查 RustFS 磁盘空间
  4. 查看 RustFS 日志: docker logs rustfs
```

### 迁移失败

```
错误: 打开本地文件失败
解决: 检查源文件是否存在于 storage_path 指定的路径

错误: 更新数据库记录失败
解决: 检查数据库连接和 disk_files 表权限
```

### 预签名 URL 过期

```
预签名 URL 有效期为 1 小时
如果下载时提示 URL 过期，前端需要重新请求新的 URL
```

---

## 7. 技术架构

### 分片上传流程

```
前端分片上传 → 分片暂存本地 chunk 目录
            → 所有分片上传完毕
            → MergeChunks:
                检查 S3 上是否有分片 → 没有则从本地上传
                → ComposeObject 服务端合并
                → 删除 S3 和本地分片
```

### 文件下载流程

```
前端请求 → 后端验证权限 → 返回预签名 URL（1小时有效）
                         → 前端直接从 RustFS 下载
```

### S3 API 使用

| 功能 | S3 API | minio-go 方法 |
|------|--------|---------------|
| 上传 | PutObject | `client.PutObject()` |
| 下载 | GetObject | `client.GetObject()` |
| 删除 | DeleteObject | `client.RemoveObject()` |
| 列目录 | ListObjectsV2 | `client.ListObjects()` |
| 文件信息 | HeadObject | `client.StatObject()` |
| 复制 | CopyObject | `client.CopyObject()` |
| 合并分片 | ComposeObject | `client.ComposeObject()` |
| 预签名 | PresignedGetObject | `client.PresignedGetObject()` |
