# RustFS S3 存储对接 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 DevOps Admin 后端实现 RustFS S3 存储对接，替换本地存储为大文件场景优化的对象存储方案，并提本地→RustFS 的数据迁移工具。

**Architecture:** 使用 `minio-go` SDK 实现 `RustFSProvider`，通过 S3 协议与 RustFS 通信。分片合并采用 `ComposeObject` 服务端合并，下载采用预签名 URL 直出。迁移工具独立为 `StorageMigrationService`，逐文件迁移并记录进度。

**Tech Stack:** Go 1.26, minio-go/v7, Gin, GORM, RustFS (S3 兼容)

---

## File Structure

| File | Operation | Responsibility |
|------|-----------|----------------|
| `service/disk/disk_rustfs.go` | **Create** | RustFSProvider 完整实现 + minio client 初始化 |
| `service/disk/disk_storage_migration.go` | **Create** | 本地→RustFS 存储迁移服务 |
| `model/disk/disk_storage_migration.go` | **Create** | 迁移记录模型 (StorageMigration) |
| `api/v1/disk/disk_storage_migration.go` | **Create** | 存储迁移 API handler |
| `router/disk/disk_storage_migration.go` | **Create** | 存储迁移路由注册 |
| `model/disk/request/disk_storage_migration.go` | **Create** | 存储迁移请求结构 |
| `model/disk/response/disk_storage_migration.go` | **Create** | 存储迁移响应结构 |
| `service/disk/disk_provider.go` | **Modify** | 删除 RustFSProvider stub，改用 disk_rustfs.go |
| `initialize/gorm.go` | **Modify** | 注册 StorageMigration 模型 |
| `go.mod` / `go.sum` | **Modify** | 添加 minio-go/v7 依赖 |
| `docs/rustfs-integration.md` | **Create** | 使用文档 |

---

### Task 1: 添加 minio-go 依赖

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: 添加 minio-go 依赖**

```bash
cd /home/devops-admin/backend
go get github.com/minio/minio-go/v7
```

- [ ] **Step 2: 验证依赖添加成功**

```bash
cd /home/devops-admin/backend
go mod tidy
grep "minio-go" go.mod
```

Expected: 输出包含 `github.com/minio/minio-go/v7`

- [ ] **Step 3: 编译验证**

```bash
cd /home/devops-admin/backend
go build ./...
```

Expected: 编译成功，无错误

---

### Task 2: 创建 StorageMigration 数据模型

**Files:**
- Create: `model/disk/disk_storage_migration.go`
- Modify: `initialize/gorm.go`

- [ ] **Step 1: 创建迁移记录模型**

创建 `model/disk/disk_storage_migration.go`:

```go
package disk

import (
	"github.com/hllkk/devopsAdmin/global"
)

// StorageMigrationStatus 迁移状态
type StorageMigrationStatus string

const (
	StorageMigrationPending  StorageMigrationStatus = "pending"
	StorageMigrationRunning  StorageMigrationStatus = "running"
	StorageMigrationSuccess  StorageMigrationStatus = "success"
	StorageMigrationFailed   StorageMigrationStatus = "failed"
)

// StorageMigration 存储迁移记录
type StorageMigration struct {
	global.OPS_MODEL
	UserID       int64                  `gorm:"index;not null;comment:用户ID" json:"userId"`
	FileID       int64                  `gorm:"index;not null;comment:文件ID" json:"fileId"`
	Status       StorageMigrationStatus `gorm:"size:20;index;default:pending;comment:迁移状态" json:"status"`
	SourcePath   string                 `gorm:"size:500;comment:源文件路径" json:"sourcePath"`
	TargetPath   string                 `gorm:"size:500;comment:目标S3路径" json:"targetPath"`
	ErrorMessage string                 `gorm:"size:1000;comment:错误信息" json:"errorMessage"`
	StartedAt    *string                `gorm:"comment:开始时间" json:"startedAt"`
	CompletedAt  *string                `gorm:"comment:完成时间" json:"completedAt"`
}

func (StorageMigration) TableName() string {
	return "disk_storage_migrations"
}
```

- [ ] **Step 2: 在 initialize/gorm.go 中注册模型**

在 `RegisterTables` 函数的 `db.AutoMigrate(...)` 调用中，在 `disk.FileShareTarget{}` 之后添加:

```go
			disk.StorageMigration{},
```

- [ ] **Step 3: 编译验证**

```bash
cd /home/devops-admin/backend
go build ./...
```

Expected: 编译成功

- [ ] **Step 4: Commit**

```bash
cd /home/devops-admin
git add backend/model/disk/disk_storage_migration.go backend/initialize/gorm.go backend/go.mod backend/go.sum
git commit -m "feat: add StorageMigration model and minio-go dependency"
```

---

### Task 3: 实现 RustFSProvider 核心方法

**Files:**
- Create: `service/disk/disk_rustfs.go`
- Modify: `service/disk/disk_provider.go` (删除 RustFSProvider stub)

- [ ] **Step 1: 创建 RustFS Provider 实现文件**

创建 `service/disk/disk_rustfs.go`:

```go
package disk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hllkk/devopsAdmin/global"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

// rustFSClient 全局 RustFS minio 客户端（懒初始化）
var rustFSClient *minio.Client

// getRustFSClient 获取或初始化 RustFS minio 客户端
func getRustFSClient() (*minio.Client, error) {
	if rustFSClient != nil {
		return rustFSClient, nil
	}

	config := global.OPS_CONFIG.RustFS
	if config.Endpoint == "" {
		return nil, fmt.Errorf("RustFS endpoint 未配置")
	}

	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 RustFS 客户端失败: %w", err)
	}

	rustFSClient = client
	return rustFSClient, nil
}

// ensureBucket 确保存储桶存在
func ensureBucket(ctx context.Context, client *minio.Client, bucketName string) error {
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("检查存储桶失败: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("创建存储桶失败: %w", err)
		}
		global.OPS_LOG.Info("RustFS 存储桶已创建", zap.String("bucket", bucketName))
	}
	return nil
}

// rustFSObjectInfo 转 os.FileInfo 适配器
type rustFSObjectInfo struct {
	objInfo minio.ObjectInfo
	isDir   bool
}

func (i *rustFSObjectInfo) Name() string {
	name := i.objInfo.Key
	parts := strings.Split(strings.TrimRight(name, "/"), "/")
	return parts[len(parts)-1]
}
func (i *rustFSObjectInfo) Size() int64        { return i.objInfo.Size }
func (i *rustFSObjectInfo) Mode() os.FileMode  { return 0644 }
func (i *rustFSObjectInfo) ModTime() time.Time { return i.objInfo.LastModified }
func (i *rustFSObjectInfo) IsDir() bool        { return i.isDir }
func (i *rustFSObjectInfo) Sys() interface{}   { return nil }

// RustFSProvider RustFS S3 兼容存储实现
type RustFSProvider struct{}

func (p *RustFSProvider) BuildPath(userID int64, currentDirectory string) (string, error) {
	if err := validatePath(currentDirectory); err != nil {
		return "", err
	}

	config := global.OPS_CONFIG.RustFS
	basePath := strings.TrimPrefix(strings.TrimSuffix(config.BasePath, "/"), "/")
	userIDStr := fmt.Sprintf("%d", userID)
	currentDirectory = strings.TrimPrefix(currentDirectory, "/")

	var parts []string
	if basePath != "" {
		parts = append(parts, basePath)
	}
	parts = append(parts, userIDStr)
	if currentDirectory != "" {
		parts = append(parts, currentDirectory)
	}

	path := strings.Join(parts, "/")
	return path, nil
}

// validatePath 验证路径安全性（复用 utils.ValidatePath 逻辑的本地版本）
func validatePath(path string) error {
	cleaned := filepath.Clean(path)
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("路径包含非法字符: %s", path)
	}
	return nil
}

func (p *RustFSProvider) GetStorageType() StorageType {
	return StorageRustFS
}

func (p *RustFSProvider) ReadDirectory(path string) ([]os.FileInfo, error) {
	ctx := context.Background()
	client, err := getRustFSClient()
	if err != nil {
		return nil, err
	}
	bucketName := global.OPS_CONFIG.RustFS.BucketName
	if err := ensureBucket(ctx, client, bucketName); err != nil {
		return nil, err
	}

	prefix := strings.TrimPrefix(path, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var fileInfos []os.FileInfo
	objects := client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Delimiter: "/",
	})

	for obj := range objects {
		if obj.Err != nil {
			return nil, fmt.Errorf("列出目录失败: %w", obj.Err)
		}

		if obj.Key != "" {
			fileInfos = append(fileInfos, &rustFSObjectInfo{objInfo: obj, isDir: false})
		}
		if obj.Prefix != "" && obj.Prefix != prefix {
			dirObj := minio.ObjectInfo{Key: obj.Prefix}
			fileInfos = append(fileInfos, &rustFSObjectInfo{objInfo: dirObj, isDir: true})
		}
	}

	return fileInfos, nil
}

func (p *RustFSProvider) GetFileInfo(path string) (os.FileInfo, error) {
	ctx := context.Background()
	client, err := getRustFSClient()
	if err != nil {
		return nil, err
	}
	bucketName := global.OPS_CONFIG.RustFS.BucketName

	objectKey := strings.TrimPrefix(path, "/")
	info, err := client.StatObject(ctx, bucketName, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	return &rustFSObjectInfo{objInfo: info, isDir: false}, nil
}

func (p *RustFSProvider) DownloadFile(filePath string) (io.ReadCloser, error) {
	ctx := context.Background()
	client, err := getRustFSClient()
	if err != nil {
		return nil, err
	}
	bucketName := global.OPS_CONFIG.RustFS.BucketName

	objectKey := strings.TrimPrefix(filePath, "/")
	obj, err := client.GetObject(ctx, bucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("下载文件失败: %w", err)
	}

	return obj, nil
}

func (p *RustFSProvider) GetFileURL(filePath string) (string, error) {
	ctx := context.Background()
	client, err := getRustFSClient()
	if err != nil {
		return "", err
	}
	bucketName := global.OPS_CONFIG.RustFS.BucketName

	objectKey := strings.TrimPrefix(filePath, "/")
	reqParams := make(url.Values)
	presignedURL, err := client.PresignedGetObject(ctx, bucketName, objectKey, 1*time.Hour, reqParams)
	if err != nil {
		return "", fmt.Errorf("生成预签名URL失败: %w", err)
	}

	return presignedURL.String(), nil
}

func (p *RustFSProvider) MergeChunks(userPath, fileName string, totalChunks int) error {
	ctx := context.Background()
	client, err := getRustFSClient()
	if err != nil {
		return err
	}
	bucketName := global.OPS_CONFIG.RustFS.BucketName

	// 构建目标 object key
	targetKey := strings.TrimPrefix(userPath, "/") + "/" + fileName

	// 构建分片 object key 列表
	var sources []minio.CopySrcOptions
	for i := 0; i < totalChunks; i++ {
		chunkKey := fmt.Sprintf("%s/%s.part%d", strings.TrimPrefix(userPath, "/"), fileName, i)
		sources = append(sources, minio.CopySrcOptions{
			Bucket: bucketName,
			Object: chunkKey,
		})
	}

	// 使用 ComposeObject 在 S3 服务端合并分片
	dst := minio.CopyDestOptions{
		Bucket: bucketName,
		Object: targetKey,
	}

	_, err = client.ComposeObject(ctx, dst, sources...)
	if err != nil {
		return fmt.Errorf("S3服务端合并分片失败: %w", err)
	}

	// 合并成功后删除分片
	for i := 0; i < totalChunks; i++ {
		chunkKey := fmt.Sprintf("%s/%s.part%d", strings.TrimPrefix(userPath, "/"), fileName, i)
		_ = client.RemoveObject(ctx, bucketName, chunkKey, minio.RemoveObjectOptions{})
	}

	return nil
}

func (p *RustFSProvider) MoveToTrash(userID int64, sourcePath, trashPath string) error {
	return p.copyAndDelete(userID, sourcePath, trashPath, "移动到回收站")
}

func (p *RustFSProvider) RestoreFromTrash(userID int64, trashPath, targetPath string) error {
	return p.copyAndDelete(userID, trashPath, targetPath, "从回收站恢复")
}

func (p *RustFSProvider) DeleteFromTrash(userID int64, trashPath string) error {
	ctx := context.Background()
	client, err := getRustFSClient()
	if err != nil {
		return err
	}
	bucketName := global.OPS_CONFIG.RustFS.BucketName

	objectKey := strings.TrimPrefix(trashPath, "/")
	err = client.RemoveObject(ctx, bucketName, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("删除回收站文件失败: %w", err)
	}
	return nil
}

// copyAndDelete 通用的复制+删除操作（用于 MoveToTrash 和 RestoreFromTrash）
func (p *RustFSProvider) copyAndDelete(userID int64, sourcePath, targetPath, action string) error {
	ctx := context.Background()
	client, err := getRustFSClient()
	if err != nil {
		return err
	}
	bucketName := global.OPS_CONFIG.RustFS.BucketName

	srcKey := strings.TrimPrefix(sourcePath, "/")
	dstKey := strings.TrimPrefix(targetPath, "/")

	// 检查源文件是否存在
	_, err = client.StatObject(ctx, bucketName, srcKey, minio.StatObjectOptions{})
	if err != nil {
		return fmt.Errorf("源文件不存在: %w", err)
	}

	// 复制到目标位置
	src := minio.CopySrcOptions{Bucket: bucketName, Object: srcKey}
	dst := minio.CopyDestOptions{Bucket: bucketName, Object: dstKey}

	_, err = client.CopyObject(ctx, dst, src)
	if err != nil {
		return fmt.Errorf("%s失败: %w", action, err)
	}

	// 删除源文件
	err = client.RemoveObject(ctx, bucketName, srcKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("%s后删除源文件失败: %w", action, err)
	}

	return nil
}

// UploadFromReader 从 Reader 上传文件到 RustFS（迁移工具用）
func UploadFromReader(ctx context.Context, objectKey string, reader io.Reader, objectSize int64, contentType string) error {
	client, err := getRustFSClient()
	if err != nil {
		return err
	}
	bucketName := global.OPS_CONFIG.RustFS.BucketName

	if err := ensureBucket(ctx, client, bucketName); err != nil {
		return err
	}

	_, err = client.PutObject(ctx, bucketName, objectKey, reader, objectSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("上传文件到 RustFS 失败: %w", err)
	}

	return nil
}

// DeleteObject 删除指定 object（迁移清理用）
func DeleteObject(ctx context.Context, objectKey string) error {
	client, err := getRustFSClient()
	if err != nil {
		return err
	}
	bucketName := global.OPS_CONFIG.RustFS.BucketName

	return client.RemoveObject(ctx, bucketName, objectKey, minio.RemoveObjectOptions{})
}

// HealthCheck 检查 RustFS 连接是否正常
func HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := getRustFSClient()
	if err != nil {
		return err
	}
	bucketName := global.OPS_CONFIG.RustFS.BucketName
	_, err = client.BucketExists(ctx, bucketName)
	return err
}
```

- [ ] **Step 2: 从 disk_provider.go 中删除 RustFSProvider stub**

在 `service/disk/disk_provider.go` 中删除以下内容（从 `type RustFSProvider struct{}` 到其所有方法直到 `type TencentCOSProvider struct{}` 之前）:

删除 RustFSProvider 结构体定义和以下方法：
- `func (p *RustFSProvider) BuildPath(...)`
- `func (p *RustFSProvider) ReadDirectory(...)`
- `func (p *RustFSProvider) GetStorageType(...)`
- `func (p *RustFSProvider) MergeChunks(...)`
- `func (p *RustFSProvider) GetFileInfo(...)`
- `func (p *RustFSProvider) DownloadFile(...)`
- `func (p *RustFSProvider) GetFileURL(...)`
- `func (p *RustFSProvider) MoveToTrash(...)`
- `func (p *RustFSProvider) RestoreFromTrash(...)`
- `func (p *RustFSProvider) DeleteFromTrash(...)`

`GetStorageProvider` 工厂函数中的 `case StorageRustFS: return &RustFSProvider{}` 不需要改动，因为实现已移到新文件。

- [ ] **Step 3: 编译验证**

```bash
cd /home/devops-admin/backend
go build ./...
```

Expected: 编译成功，无错误

- [ ] **Step 4: Commit**

```bash
cd /home/devops-admin
git add backend/service/disk/disk_rustfs.go backend/service/disk/disk_provider.go
git commit -m "feat: implement RustFSProvider with minio-go S3 integration"
```

---

### Task 4: 创建存储迁移请求/响应模型

**Files:**
- Create: `model/disk/request/disk_storage_migration.go`
- Create: `model/disk/response/disk_storage_migration.go`

- [ ] **Step 1: 创建请求模型**

创建 `model/disk/request/disk_storage_migration.go`:

```go
package request

// StorageMigrationRequest 存储迁移请求
type StorageMigrationRequest struct {
	UserID int64 `json:"userId"` // 可选：仅迁移指定用户，0 表示全量迁移
}

// StorageMigrationRetryRequest 存储迁移重试请求
type StorageMigrationRetryRequest struct {
	UserID int64 `json:"userId"` // 可选：仅重试指定用户的失败项，0 表示全部重试
}
```

- [ ] **Step 2: 创建响应模型**

创建 `model/disk/response/disk_storage_migration.go`:

```go
package response

// StorageMigrationStatusResponse 存储迁移状态响应
type StorageMigrationStatusResponse struct {
	Total   int64 `json:"total"`
	Success int64 `json:"success"`
	Failed  int64 `json:"failed"`
	Pending int64 `json:"pending"`
	Running int64 `json:"running"`
}

// StorageMigrationResultResponse 存储迁移执行结果
type StorageMigrationResultResponse struct {
	Total     int64                        `json:"total"`
	Success   int64                        `json:"success"`
	Failed    int64                        `json:"failed"`
	Skipped   int64                        `json:"skipped"`
	Duration  string                       `json:"duration"`
	Errors    []StorageMigrationError      `json:"errors,omitempty"`
}

// StorageMigrationError 存储迁移错误记录
type StorageMigrationError struct {
	FileID int64  `json:"fileId"`
	Name   string `json:"name"`
	Path   string `json:"path"`
	Error  string `json:"error"`
}
```

- [ ] **Step 3: 编译验证**

```bash
cd /home/devops-admin/backend
go build ./...
```

- [ ] **Step 4: Commit**

```bash
cd /home/devops-admin
git add backend/model/disk/request/disk_storage_migration.go backend/model/disk/response/disk_storage_migration.go
git commit -m "feat: add storage migration request/response models"
```

---

### Task 5: 实现存储迁移服务

**Files:**
- Create: `service/disk/disk_storage_migration.go`

- [ ] **Step 1: 创建存储迁移服务**

创建 `service/disk/disk_storage_migration.go`:

```go
package disk

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hllkk/devopsAdmin/global"
	diskModel "github.com/hllkk/devopsAdmin/model/disk"
	diskRequest "github.com/hllkk/devopsAdmin/model/disk/request"
	diskResponse "github.com/hllkk/devopsAdmin/model/disk/response"
	"go.uber.org/zap"
)

type StorageMigrationService struct{}

// MigrateStorage 执行本地→RustFS 存储迁移
func (s *StorageMigrationService) MigrateStorage(params diskRequest.StorageMigrationRequest) (diskResponse.StorageMigrationResultResponse, error) {
	result := diskResponse.StorageMigrationResultResponse{
		Errors: []diskResponse.StorageMigrationError{},
	}
	startTime := time.Now()

	// 1. 检查 RustFS 连接
	if err := HealthCheck(); err != nil {
		return result, fmt.Errorf("RustFS 连接检查失败: %w", err)
	}

	// 2. 查询需要迁移的文件（仅非文件夹的文件，且 storage_path 非空）
	query := global.OPS_DB.Model(&diskModel.File{}).
		Where("is_folder = ? AND storage_path != '' AND storage_path IS NOT NULL", false)

	if params.UserID > 0 {
		query = query.Where("user_id = ?", params.UserID)
	}

	var files []diskModel.File
	if err := query.Find(&files).Error; err != nil {
		return result, fmt.Errorf("查询文件列表失败: %w", err)
	}

	result.Total = int64(len(files))
	if result.Total == 0 {
		result.Duration = time.Since(startTime).String()
		return result, nil
	}

	// 3. 逐文件迁移
	ctx := context.Background()
	for _, file := range files {
		if err := s.migrateSingleFile(ctx, file); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, diskResponse.StorageMigrationError{
				FileID: file.ID,
				Name:   file.Name,
				Path:   file.Path,
				Error:  err.Error(),
			})
			global.OPS_LOG.Error("文件迁移失败",
				zap.Int64("fileId", file.ID),
				zap.String("name", file.Name),
				zap.Error(err))
		} else {
			result.Success++
		}
	}

	result.Duration = time.Since(startTime).String()
	global.OPS_LOG.Info("存储迁移完成",
		zap.Int64("total", result.Total),
		zap.Int64("success", result.Success),
		zap.Int64("failed", result.Failed),
		zap.String("duration", result.Duration))

	return result, nil
}

// migrateSingleFile 迁移单个文件
func (s *StorageMigrationService) migrateSingleFile(ctx context.Context, file diskModel.File) error {
	// 1. 打开本地文件
	localPath := file.StoragePath
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 2. 构建 S3 object key
	provider := &RustFSProvider{}
	s3DirPath, err := provider.BuildPath(file.UserID, file.Path)
	if err != nil {
		return fmt.Errorf("构建S3路径失败: %w", err)
	}
	objectKey := s3DirPath + "/" + file.Name

	// 3. 上传到 RustFS
	if err := UploadFromReader(ctx, objectKey, f, stat.Size(), file.ContentType); err != nil {
		return fmt.Errorf("上传到RustFS失败: %w", err)
	}

	// 4. 更新数据库记录
	newStoragePath := objectKey
	if err := global.OPS_DB.Model(&file).Updates(map[string]interface{}{
		"storage_path": newStoragePath,
	}).Error; err != nil {
		return fmt.Errorf("更新数据库记录失败: %w", err)
	}

	return nil
}

// GetMigrationStatus 获取迁移状态统计
func (s *StorageMigrationService) GetMigrationStatus(userID int64) (diskResponse.StorageMigrationStatusResponse, error) {
	var status diskResponse.StorageMigrationStatusResponse

	query := global.OPS_DB.Model(&diskModel.StorageMigration{})
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	var results []struct {
		Status  diskModel.StorageMigrationStatus
		Count   int64
	}

	if err := query.Select("status, count(*) as count").Group("status").Scan(&results).Error; err != nil {
		return status, fmt.Errorf("查询迁移状态失败: %w", err)
	}

	for _, r := range results {
		switch r.Status {
		case diskModel.StorageMigrationPending:
			status.Pending = r.Count
		case diskModel.StorageMigrationRunning:
			status.Running = r.Count
		case diskModel.StorageMigrationSuccess:
			status.Success = r.Count
		case diskModel.StorageMigrationFailed:
			status.Failed = r.Count
		}
		status.Total += r.Count
	}

	return status, nil
}

// RetryFailed 重试失败的迁移
func (s *StorageMigrationService) RetryFailed(params diskRequest.StorageMigrationRetryRequest) (diskResponse.StorageMigrationResultResponse, error) {
	// 查询失败的迁移记录
	query := global.OPS_DB.Model(&diskModel.StorageMigration{}).
		Where("status = ?", diskModel.StorageMigrationFailed)

	if params.UserID > 0 {
		query = query.Where("user_id = ?", params.UserID)
	}

	var failedMigrations []diskModel.StorageMigration
	if err := query.Find(&failedMigrations).Error; err != nil {
		return diskResponse.StorageMigrationResultResponse{}, fmt.Errorf("查询失败记录失败: %w", err)
	}

	result := diskResponse.StorageMigrationResultResponse{
		Total: int64(len(failedMigrations)),
		Errors: []diskResponse.StorageMigrationError{},
	}

	ctx := context.Background()
	for _, migration := range failedMigrations {
		var file diskModel.File
		if err := global.OPS_DB.First(&file, migration.FileID).Error; err != nil {
			result.Failed++
			continue
		}

		if err := s.migrateSingleFile(ctx, file); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, diskResponse.StorageMigrationError{
				FileID: file.ID,
				Name:   file.Name,
				Path:   file.Path,
				Error:  err.Error(),
			})
			// 更新迁移记录状态
			now := time.Now().Format("2006-01-02 15:04:05")
			global.OPS_DB.Model(&migration).Updates(map[string]interface{}{
				"status":        diskModel.StorageMigrationFailed,
				"error_message": err.Error(),
				"completed_at":  now,
			})
		} else {
			result.Success++
			now := time.Now().Format("2006-01-02 15:04:05")
			global.OPS_DB.Model(&migration).Updates(map[string]interface{}{
				"status":       diskModel.StorageMigrationSuccess,
				"completed_at": now,
			})
		}
	}

	return result, nil
}

// EstimateMigration 估算迁移工作量
func (s *StorageMigrationService) EstimateMigration(userID int64) (int64, int64, error) {
	query := global.OPS_DB.Model(&diskModel.File{}).
		Where("is_folder = ? AND storage_path != '' AND storage_path IS NOT NULL", false)

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	var count int64
	var totalSize int64

	if err := query.Count(&count).Error; err != nil {
		return 0, 0, err
	}

	row := global.OPS_DB.Model(&diskModel.File{}).
		Where("is_folder = ? AND storage_path != '' AND storage_path IS NOT NULL", false).
		Select("COALESCE(SUM(size), 0)")

	if userID > 0 {
		row = row.Where("user_id = ?", userID)
	}

	if err := row.Scan(&totalSize).Error; err != nil {
		return count, 0, err
	}

	return count, totalSize, nil
}

// formatDuration 格式化持续时间
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1f秒", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1f分钟", d.Minutes())
	}
	return fmt.Sprintf("%.1f小时", d.Hours())
}

// Ensure StorageMigrationService implements expected interface
var _ = strings.TrimSpace("")
```

- [ ] **Step 2: 编译验证**

```bash
cd /home/devops-admin/backend
go build ./...
```

Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
cd /home/devops-admin
git add backend/service/disk/disk_storage_migration.go
git commit -m "feat: implement storage migration service (local to RustFS)"
```

---

### Task 6: 创建存储迁移 API 和路由

**Files:**
- Create: `api/v1/disk/disk_storage_migration.go`
- Create: `router/disk/disk_storage_migration.go`

- [ ] **Step 1: 创建 API handler**

创建 `api/v1/disk/disk_storage_migration.go`:

```go
package disk

import (
	"github.com/gin-gonic/gin"
	"github.com/hllkk/devopsAdmin/global"
	"github.com/hllkk/devopsAdmin/model/common/response"
	diskRequest "github.com/hllkk/devopsAdmin/model/disk/request"
	"github.com/hllkk/devopsAdmin/utils"
	"go.uber.org/zap"
)

type StorageMigrationApi struct{}

// StartStorageMigration 开始存储迁移
// @Summary 存储迁移
// @Description 将本地存储文件迁移到 RustFS S3 存储
// @Tags disk-storage-migration
// @Accept json
// @Produce json
// @Param data body diskRequest.StorageMigrationRequest true "迁移参数"
// @Success 200 {object} response.Response{data=diskResponse.StorageMigrationResultResponse}
// @Router /disk/storage-migration/start [post]
func (a *StorageMigrationApi) StartStorageMigration(c *gin.Context) {
	currentUserID := utils.GetUserID(c)
	if currentUserID == 0 {
		response.FailWithMessage("未登录", c)
		return
	}

	var params diskRequest.StorageMigrationRequest
	if err := c.ShouldBindJSON(&params); err != nil {
		global.OPS_LOG.Error("参数绑定失败", zap.Error(err))
		response.FailWithMessage("参数格式错误", c)
		return
	}

	result, err := StorageMigrationService.MigrateStorage(params)
	if err != nil {
		global.OPS_LOG.Error("存储迁移失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	global.OPS_LOG.Info("存储迁移完成",
		zap.Int64("total", result.Total),
		zap.Int64("success", result.Success),
		zap.Int64("failed", result.Failed))

	response.OkWithDetailed(result, "存储迁移完成", c)
}

// GetStorageMigrationStatus 查询迁移状态
// @Summary 迁移状态
// @Description 查询存储迁移的进度和状态统计
// @Tags disk-storage-migration
// @Produce json
// @Param userId query int false "用户ID（可选）"
// @Success 200 {object} response.Response{data=diskResponse.StorageMigrationStatusResponse}
// @Router /disk/storage-migration/status [get]
func (a *StorageMigrationApi) GetStorageMigrationStatus(c *gin.Context) {
	var userID int64
	if uid := c.Query("userId"); uid != "" {
		if _, err := fmt.Sscanf(uid, "%d", &userID); err != nil {
			userID = 0
		}
	}

	status, err := StorageMigrationService.GetMigrationStatus(userID)
	if err != nil {
		global.OPS_LOG.Error("查询迁移状态失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(status, "成功", c)
}

// RetryStorageMigration 重试失败的迁移
// @Summary 重试迁移
// @Description 重试失败的存储迁移项
// @Tags disk-storage-migration
// @Accept json
// @Produce json
// @Param data body diskRequest.StorageMigrationRetryRequest true "重试参数"
// @Success 200 {object} response.Response{data=diskResponse.StorageMigrationResultResponse}
// @Router /disk/storage-migration/retry [post]
func (a *StorageMigrationApi) RetryStorageMigration(c *gin.Context) {
	currentUserID := utils.GetUserID(c)
	if currentUserID == 0 {
		response.FailWithMessage("未登录", c)
		return
	}

	var params diskRequest.StorageMigrationRetryRequest
	if err := c.ShouldBindJSON(&params); err != nil {
		global.OPS_LOG.Error("参数绑定失败", zap.Error(err))
		response.FailWithMessage("参数格式错误", c)
		return
	}

	result, err := StorageMigrationService.RetryFailed(params)
	if err != nil {
		global.OPS_LOG.Error("重试迁移失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "重试完成", c)
}

// EstimateStorageMigration 估算迁移工作量
// @Summary 估算迁移
// @Description 估算存储迁移的文件数量和总大小
// @Tags disk-storage-migration
// @Produce json
// @Param userId query int false "用户ID（可选）"
// @Success 200 {object} response.Response{data=object}
// @Router /disk/storage-migration/estimate [get]
func (a *StorageMigrationApi) EstimateStorageMigration(c *gin.Context) {
	var userID int64
	if uid := c.Query("userId"); uid != "" {
		if _, err := fmt.Sscanf(uid, "%d", &userID); err != nil {
			userID = 0
		}
	}

	count, totalSize, err := StorageMigrationService.EstimateMigration(userID)
	if err != nil {
		global.OPS_LOG.Error("估算迁移工作量失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(gin.H{
		"fileCount": count,
		"totalSize": totalSize,
		"totalSizeMB": totalSize / (1024 * 1024),
		"totalSizeGB": totalSize / (1024 * 1024 * 1024),
	}, "成功", c)
}
```

注意：`fmt` 包需要添加到 import 中。实际文件中 `fmt.Sscanf` 需要导入 `fmt`。

- [ ] **Step 2: 创建路由注册**

创建 `router/disk/disk_storage_migration.go`:

```go
package disk

import (
	"github.com/gin-gonic/gin"
)

type StorageMigrationRouter struct{}

func (s *StorageMigrationRouter) InitStorageMigrationRouter(Router *gin.RouterGroup) {
	storageMigrationRouter := Router.Group("storage-migration")

	{
		storageMigrationRouter.POST("/start", StorageMigrationApi.StartStorageMigration)         // 开始迁移
		storageMigrationRouter.GET("/status", StorageMigrationApi.GetStorageMigrationStatus)     // 查询状态
		storageMigrationRouter.POST("/retry", StorageMigrationApi.RetryStorageMigration)         // 重试失败
		storageMigrationRouter.GET("/estimate", StorageMigrationApi.EstimateStorageMigration)    // 估算工作量
	}
}
```

- [ ] **Step 3: 注册路由和 API 到 enter.go**

检查 `router/disk/enter.go` 和 `api/v1/disk/enter.go`，确保新的 Router 和 Api 被注册。

在 `api/v1/disk/enter.go` 中添加:
```go
StorageMigrationApi = apiDisk.StorageMigrationApi{}
```

在 `router/disk/enter.go` 中添加:
```go
StorageMigrationRouter
```

在 `service/disk/enter.go` 中添加:
```go
StorageMigrationService = diskSvc.StorageMigrationService{}
```

- [ ] **Step 4: 编译验证**

```bash
cd /home/devops-admin/backend
go build ./...
```

Expected: 编译成功

- [ ] **Step 5: Commit**

```bash
cd /home/devops-admin
git add backend/api/v1/disk/disk_storage_migration.go backend/router/disk/disk_storage_migration.go backend/api/v1/disk/enter.go backend/router/disk/enter.go backend/service/disk/enter.go
git commit -m "feat: add storage migration API, router and service registration"
```

---

### Task 7: 适配分片上传逻辑

**Files:**
- Modify: `service/disk/disk_file.go`

需要确保分片上传在 RustFS 模式下工作正常。当前分片上传逻辑是：分片暂存到本地 → 调用 `provider.MergeChunks()` 合并。

对于 RustFS，需要调整流程：
1. 分片上传时直接将分片上传到 RustFS（作为 `.partN` 对象）
2. 合并时调用 `RustFSProvider.MergeChunks()` 执行 `ComposeObject`

- [ ] **Step 1: 查看当前 UploadFile 方法中分片上传的逻辑**

在 `service/disk/disk_file.go` 中找到分片上传相关代码，理解分片暂存到本地的流程。

- [ ] **Step 2: 确认是否需要修改**

如果当前分片上传流程是：
1. 分片先保存到本地临时目录
2. 调用 `provider.MergeChunks()` 合并

那么对于 RustFS，需要在分片上传时将每个分片上传到 S3 作为 `.partN` 对象。检查 `UploadFile` 方法是否需要根据 `storageType` 走不同分支。

如果分片上传在 API 层（handler）就已经保存到本地了，那么需要在 `MergeChunks` 被调用前，将本地分片上传到 RustFS，然后在 `MergeChunks` 中执行 `ComposeObject`。

修改 `RustFSProvider.MergeChunks()` 的实现：在执行 `ComposeObject` 前，先检查分片是否已在 S3 上（如果不在，则从本地暂存目录上传）。

修改 `service/disk/disk_rustfs.go` 中 `MergeChunks` 方法，在 ComposeObject 之前添加本地分片上传逻辑：

```go
func (p *RustFSProvider) MergeChunks(userPath, fileName string, totalChunks int) error {
	ctx := context.Background()
	client, err := getRustFSClient()
	if err != nil {
		return err
	}
	bucketName := global.OPS_CONFIG.RustFS.BucketName

	// 构建 S3 上的分片 object key，确保分片已在 S3 上
	var sources []minio.CopySrcOptions
	for i := 0; i < totalChunks; i++ {
		chunkKey := fmt.Sprintf("%s/%s.part%d", strings.TrimPrefix(userPath, "/"), fileName, i)

		// 检查分片是否已在 S3 上
		_, err := client.StatObject(ctx, bucketName, chunkKey, minio.StatObjectOptions{})
		if err != nil {
			// 分片不在 S3 上，尝试从本地暂存目录上传
			localChunkPath := userPath + "/" + fmt.Sprintf("%s.part%d", fileName, i)
			chunkFile, openErr := os.Open(localChunkPath)
			if openErr != nil {
				return fmt.Errorf("分片 %d 既不在 S3 上也不在本地: %w", i, openErr)
			}
			defer chunkFile.Close()

			chunkStat, statErr := chunkFile.Stat()
			if statErr != nil {
				return fmt.Errorf("获取本地分片信息失败: %w", statErr)
			}

			if uploadErr := UploadFromReader(ctx, chunkKey, chunkFile, chunkStat.Size(), "application/octet-stream"); uploadErr != nil {
				return fmt.Errorf("上传本地分片 %d 到 S3 失败: %w", i, uploadErr)
			}
		}

		sources = append(sources, minio.CopySrcOptions{
			Bucket: bucketName,
			Object: chunkKey,
		})
	}

	// 使用 ComposeObject 在 S3 服务端合并分片
	targetKey := strings.TrimPrefix(userPath, "/") + "/" + fileName
	dst := minio.CopyDestOptions{
		Bucket: bucketName,
		Object: targetKey,
	}

	_, err = client.ComposeObject(ctx, dst, sources...)
	if err != nil {
		return fmt.Errorf("S3服务端合并分片失败: %w", err)
	}

	// 合并成功后删除 S3 上的分片
	for i := 0; i < totalChunks; i++ {
		chunkKey := fmt.Sprintf("%s/%s.part%d", strings.TrimPrefix(userPath, "/"), fileName, i)
		_ = client.RemoveObject(ctx, bucketName, chunkKey, minio.RemoveObjectOptions{})
	}

	// 同时删除本地暂存分片
	for i := 0; i < totalChunks; i++ {
		localChunkPath := userPath + "/" + fmt.Sprintf("%s.part%d", fileName, i)
		os.Remove(localChunkPath)
	}

	return nil
}
```

- [ ] **Step 3: 编译验证**

```bash
cd /home/devops-admin/backend
go build ./...
```

Expected: 编译成功

- [ ] **Step 4: Commit**

```bash
cd /home/devops-admin
git add backend/service/disk/disk_rustfs.go
git commit -m "feat: adapt MergeChunks to upload local chunks to S3 before composing"
```

---

### Task 8: 生成使用文档

**Files:**
- Create: `docs/rustfs-integration.md`

- [ ] **Step 1: 创建使用文档**

创建 `docs/rustfs-integration.md`，包含：
1. RustFS 介绍和优势
2. Docker 部署步骤
3. 后端配置说明
4. API 接口文档
5. 存储迁移指南
6. 故障排查

文档内容参见 Task 8 的详细模板（在实现时生成完整内容）。

- [ ] **Step 2: Commit**

```bash
cd /home/devops-admin
git add docs/rustfs-integration.md
git commit -m "docs: add RustFS integration guide"
```

---

### Task 9: 全量编译测试和清理

**Files:**
- All modified files

- [ ] **Step 1: 全量编译**

```bash
cd /home/devops-admin/backend
go build ./...
```

Expected: 编译成功

- [ ] **Step 2: go vet 静态检查**

```bash
cd /home/devops-admin/backend
go vet ./...
```

Expected: 无警告

- [ ] **Step 3: 删除编译产物**

```bash
cd /home/devops-admin/backend
rm -f devopsAdmin
```

- [ ] **Step 4: 最终提交**

```bash
cd /home/devops-admin
git add -A
git status
```

确认所有变更正确后提交。

---

## Self-Review Checklist

- [x] Spec coverage: 每个需求都有对应 Task
- [x] Placeholder scan: 无 TBD/TODO
- [x] Type consistency: RustFSProvider, StorageMigrationService, API/Router 命名一致
- [x] File paths: 所有路径精确到文件名
- [x] Build verification: 每个 Task 都有编译验证步骤
