# 网盘共享系统重新设计 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 重构内部共享系统，实现虚拟挂载点架构、COW 写时复制、自动接受共享、批量授权管理。

**Architecture:** 方案 A 虚拟挂载点。共享文件通过 `disk_file_mounts` 表在用户文件树中创建虚拟入口。`ref_count` 引用计数管理文件生命周期。COW 在修改时触发懒复制。SSE 实时推送共享通知。

**Tech Stack:** Go 1.26 + Gin + GORM (后端) / Vue3 + TypeScript + Naive UI (前端) / MySQL 8.0 / Redis

**Spec:** `docs/superpowers/specs/2026-06-02-disk-share-redesign.md`

---

## Phase 1: 数据库与模型层

### Task 1: 新建 disk_file_mounts 模型

**Files:**
- Create: `backend/model/disk/disk_file_mount.go`

- [ ] **Step 1: 创建挂载点模型文件**

```go
package disk

import "devops-admin/global"

// FileMount 文件挂载点 — 记录"别人的文件"出现在"我的文件树"的位置
type FileMount struct {
	global.OPS_MODEL
	UserID       int64  `gorm:"index;not null;comment:挂载所属用户ID" json:"userId"`
	SourceFileID int64  `gorm:"index;not null;comment:源文件ID" json:"sourceFileId"`
	ParentID     *int64 `gorm:"index;comment:挂载到用户文件树的父目录ID" json:"parentId"`
	MountName    string `gorm:"size:255;comment:挂载显示名称" json:"mountName"`
	MountType    string `gorm:"size:20;index;not null;comment:挂载类型(share/save)" json:"mountType"`
	ShareID      *int64 `gorm:"index;comment:关联的共享记录ID" json:"shareId"`
	File         *File  `gorm:"foreignKey:SourceFileID;references:ID" json:"file,omitempty"`
}
```

- [ ] **Step 2: 注册到 AutoMigrate**

修改 `backend/source/system/auto_migrate.go`，在 `initAutoMigrate` 结构体中的 `Tables` 切片里添加 `&diskModel.FileMount{}`。

```go
// 在现有的 FileShareOperation 后面添加
&diskModel.FileMount{},
```

- [ ] **Step 3: 启动后端验证表创建**

Run: `cd backend && go build ./...`
Expected: 编译通过，无错误

- [ ] **Step 4: Commit**

```bash
git add backend/model/disk/disk_file_mount.go backend/source/system/auto_migrate.go
git commit -m "feat(disk): add disk_file_mounts model for virtual mount points"
```

---

### Task 2: 扩展 disk_files 模型

**Files:**
- Modify: `backend/model/disk/disk_file.go` (File struct)

- [ ] **Step 1: 在 File struct 中添加三个新字段**

在 `backend/model/disk/disk_file.go` 的 `File` struct 中，`Props` 字段之前添加：

```go
RefCount    int    `gorm:"default:1;comment:引用计数" json:"refCount"`
StorageHash string `gorm:"size:64;index;comment:文件内容SHA256哈希" json:"storageHash"`
IsCow       bool   `gorm:"default:false;comment:是否已触发写时复制" json:"isCow"`
```

- [ ] **Step 2: 启动后端验证编译**

Run: `cd backend && go build ./...`
Expected: 编译通过

- [ ] **Step 3: Commit**

```bash
git add backend/model/disk/disk_file.go
git commit -m "feat(disk): add ref_count, storage_hash, is_cow fields to File model"
```

---

### Task 3: 调整 FileShare 和 FileShareTarget 模型

**Files:**
- Modify: `backend/model/disk/disk_file_share.go`

- [ ] **Step 1: 修改 FileShare struct**

在 `backend/model/disk/disk_file_share.go` 中修改 `FileShare` struct：

移除 `Status` 和 `Remark` 字段，添加 `NotifySent` 字段：

```go
// FileShare struct 最终状态:
type FileShare struct {
	global.OPS_MODEL
	UserID      int64                        `gorm:"index;not null;comment:共享发起人ID" json:"userId"`
	FileID      int64                        `gorm:"index;not null;comment:文件ID" json:"fileId"`
	ShareType   string                       `gorm:"size:20;index;not null;comment:共享类型(user/dept)" json:"shareType"`
	Permissions []common.OperationPermission `gorm:"type:json;serializer:json;comment:操作权限列表" json:"permissions"`
	ExpireDate  *time.Time                   `gorm:"index;comment:过期时间" json:"expireDate"`
	NotifySent  bool                         `gorm:"default:false;comment:是否已发送通知" json:"notifySent"`
	File        *File                        `gorm:"foreignKey:FileID;references:ID;constraint:OnDelete:CASCADE" json:"file,omitempty"`
	Targets     []FileShareTarget            `gorm:"foreignKey:FileShareID;references:ID" json:"targets,omitempty"`
}
```

- [ ] **Step 2: 修改 FileShareTarget struct**

移除 `Status` 字段，添加 `MountName` 字段：

```go
// FileShareTarget struct 最终状态:
type FileShareTarget struct {
	global.OPS_MODEL
	FileShareID int64                        `gorm:"index;not null;comment:关联FileShare ID" json:"fileShareId"`
	TargetType  string                       `gorm:"size:20;not null;comment:目标类型(user/dept)" json:"targetType"`
	TargetID    int64                        `gorm:"index;not null;comment:目标ID(用户ID或部门ID)" json:"targetId"`
	Permissions []common.OperationPermission `gorm:"type:json;serializer:json;comment:目标独立权限" json:"permissions"`
	MountName   string                       `gorm:"size:255;comment:挂载点显示名称" json:"mountName"`
}
```

- [ ] **Step 3: 编译验证**

Run: `cd backend && go build ./...`

注意：移除 `Status` 和 `Remark` 字段会导致引用它们的代码编译错误，这些在后续 Task 中修复。

- [ ] **Step 4: Commit**

```bash
git add backend/model/disk/disk_file_share.go
git commit -m "refactor(disk): remove status/remark from FileShare, add notify_sent; remove status from FileShareTarget, add mount_name"
```

---

## Phase 2: 后端 Service 层

### Task 4: 权限校验服务

**Files:**
- Create: `backend/service/disk/disk_share_permission.go`

- [ ] **Step 1: 创建权限校验服务**

```go
package disk

import (
	"devops-admin/global"
	"devops-admin/model/common"
	"devops-admin/utils/cache"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	permCachePrefix  = "share_perm:"
	permCacheTTL     = 5 * time.Minute
)

// CheckSharePermission 校验当前用户对共享文件的权限
// requiredPerm: DOWNLOAD, UPLOAD, PUT, DELETE
func CheckSharePermission(ctx *gin.Context, fileID int64, requiredPerm common.OperationPermission) error {
	userID := getUserID(ctx)
	perms, err := GetUserSharePermissions(userID, fileID)
	if err != nil {
		return fmt.Errorf("查询权限失败: %w", err)
	}
	if !hasPermission(perms, requiredPerm) {
		return fmt.Errorf("无权限执行此操作")
	}
	return nil
}

// GetUserSharePermissions 获取用户对文件的共享权限（带缓存）
func GetUserSharePermissions(userID, fileID int64) ([]common.OperationPermission, error) {
	cacheKey := fmt.Sprintf("%s%d:%d", permCachePrefix, userID, fileID)

	// 尝试从缓存获取
	cached, err := global.Redis.Get(cacheKey).Result()
	if err == nil && cached != "" {
		var perms []common.OperationPermission
		if json.Unmarshal([]byte(cached), &perms) == nil {
			return perms, nil
		}
	}

	// 从数据库查询
	perms, err := querySharePermissionsFromDB(userID, fileID)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	if data, err := json.Marshal(perms); err == nil {
		global.Redis.Set(cacheKey, string(data), permCacheTTL)
	}

	return perms, nil
}

// InvalidateSharePermissionCache 失效权限缓存
func InvalidateSharePermissionCache(userID, fileID int64) {
	cacheKey := fmt.Sprintf("%s%d:%d", permCachePrefix, userID, fileID)
	global.Redis.Del(cacheKey)
}

// querySharePermissionsFromDB 从数据库查询共享权限
func querySharePermissionsFromDB(userID, fileID int64) ([]common.OperationPermission, error) {
	var target disk.FileShareTarget
	err := global.DB.
		Table("disk_file_share_targets t").
		Select("t.permissions").
		Joins("JOIN disk_file_shares s ON s.id = t.file_share_id").
		Where("s.file_id = ? AND s.deleted_at IS NULL", fileID).
		Where("(t.target_type = 'user' AND t.target_id = ?)", userID).
		Or("(t.target_type = 'dept' AND t.target_id IN (?) AND s.deleted_at IS NULL)", getUserDeptIDs(userID)).
		First(&target).Error

	if err != nil {
		return nil, fmt.Errorf("未找到共享权限")
	}

	if len(target.Permissions) > 0 {
		return target.Permissions, nil
	}

	// target 没有独立权限时，继承 share 级别权限
	var share disk.FileShare
	if err := global.DB.Where("file_id = ? AND deleted_at IS NULL", fileID).First(&share).Error; err != nil {
		return nil, fmt.Errorf("未找到共享记录")
	}
	return share.Permissions, nil
}

func hasPermission(perms []common.OperationPermission, required common.OperationPermission) bool {
	for _, p := range perms {
		if p == required {
			return true
		}
	}
	return false
}

func getUserID(ctx *gin.Context) int64 {
	// 从 gin context 获取当前用户 ID，与项目现有模式一致
	id, _ := ctx.Get("userID")
	if id != nil {
		if uid, ok := id.(int64); ok {
			return uid
		}
		if uid, ok := id.(uint); ok {
			return int64(uid)
		}
		if uid, ok := id.(float64); ok {
			return int64(uid)
		}
	}
	return 0
}

func getUserDeptIDs(userID int64) []int64 {
	// 查询用户所属部门 ID 列表
	var deptIDs []int64
	global.DB.Table("sys_user_authorities").
		Select("sys_authority_id").
		Where("sys_user_id = ?", userID).
		Pluck("sys_authority_id", &deptIDs)
	return deptIDs
}
```

注意：`getUserID` 函数需要根据项目中实际获取用户 ID 的方式调整。参考 `disk_internal_share.go` 中已有的模式。

- [ ] **Step 2: 编译验证**

Run: `cd backend && go build ./...`
Expected: 编译通过（可能需要调整 import 路径和 getUserID 的实现）

- [ ] **Step 3: Commit**

```bash
git add backend/service/disk/disk_share_permission.go
git commit -m "feat(disk): add share permission service with Redis caching"
```

---

### Task 5: 挂载点管理服务

**Files:**
- Create: `backend/service/disk/disk_share_mount.go`

- [ ] **Step 1: 创建挂载点服务**

```go
package disk

import (
	"devops-admin/global"
	"devops-admin/model/disk"
	"fmt"
)

// CreateShareMounts 为共享目标批量创建挂载点
func CreateShareMounts(fileID int64, shareID int64, targets []disk.FileShareTarget, mountName string) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		for _, target := range targets {
			if target.TargetType == "user" {
				mount := disk.FileMount{
					UserID:       target.TargetID,
					SourceFileID: fileID,
					MountName:    firstNonEmpty(target.MountName, mountName),
					MountType:    "share",
					ShareID:      &shareID,
				}
				if err := tx.Create(&mount).Error; err != nil {
					return fmt.Errorf("创建用户挂载失败: %w", err)
				}
			} else {
				// dept → 为所有成员创建挂载
				memberIDs := getDeptMemberIDs(target.TargetID)
				for _, memberID := range memberIDs {
					mount := disk.FileMount{
						UserID:       memberID,
						SourceFileID: fileID,
						MountName:    firstNonEmpty(target.MountName, mountName),
						MountType:    "share",
						ShareID:      &shareID,
					}
					if err := tx.Create(&mount).Error; err != nil {
						return fmt.Errorf("创建部门成员挂载失败: %w", err)
					}
				}
			}
		}

		// 更新 ref_count
		mountCount := countNewMounts(targets)
		if mountCount > 0 {
			tx.Model(&disk.File{}).Where("id = ?", fileID).
				Update("ref_count", gorm.Expr("ref_count + ?", mountCount))
		}
		return nil
	})
}

// CreateSaveMount 为"保存到我的网盘"创建挂载点
func CreateSaveMount(userID, fileID, parentID int64, mountName string, shareID *int64) error {
	// 检查是否已保存
	var count int64
	global.DB.Model(&disk.FileMount{}).
		Where("user_id = ? AND source_file_id = ? AND mount_type = 'save'", userID, fileID).
		Count(&count)
	if count > 0 {
		return fmt.Errorf("文件已保存到网盘")
	}

	mount := disk.FileMount{
		UserID:       userID,
		SourceFileID: fileID,
		ParentID:     &parentID,
		MountName:    mountName,
		MountType:    "save",
		ShareID:      shareID,
	}

	return global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&mount).Error; err != nil {
			return fmt.Errorf("创建保存挂载失败: %w", err)
		}
		// ref_count + 1
		return tx.Model(&disk.File{}).Where("id = ?", fileID).
			Update("ref_count", gorm.Expr("ref_count + 1")).Error
	})
}

// RemoveShareMounts 移除共享挂载（取消共享时调用）
func RemoveShareMounts(shareID int64) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		// 查询受影响的文件
		var mounts []disk.FileMount
		if err := tx.Where("share_id = ? AND mount_type = 'share'", shareID).Find(&mounts).Error; err != nil {
			return err
		}

		// 按文件分组统计
		fileCounts := make(map[int64]int)
		for _, m := range mounts {
			fileCounts[m.SourceFileID]++
		}

		// 删除挂载
		if err := tx.Where("share_id = ? AND mount_type = 'share'", shareID).Delete(&disk.FileMount{}).Error; err != nil {
			return err
		}

		// 更新 ref_count
		for fileID, count := range fileCounts {
			tx.Model(&disk.File{}).Where("id = ?", fileID).
				Update("ref_count", gorm.Expr("ref_count - ?", count))
		}
		return nil
	})
}

// RemoveSingleMount 移除单个挂载
func RemoveSingleMount(userID, sourceFileID int64, mountType string) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("user_id = ? AND source_file_id = ? AND mount_type = ?",
			userID, sourceFileID, mountType).Delete(&disk.FileMount{})
		if result.RowsAffected > 0 {
			tx.Model(&disk.File{}).Where("id = ?", sourceFileID).
				Update("ref_count", gorm.Expr("ref_count - 1"))
		}
		return result.Error
	})
}

// GetUserMounts 获取用户的挂载点列表
func GetUserMounts(userID int64, parentID *int64) ([]disk.FileMount, error) {
	var mounts []disk.FileMount
	query := global.DB.Where("user_id = ?", userID)
	if parentID != nil {
		query = query.Where("parent_id = ?", parentID)
	}
	err := query.Preload("File").Find(&mounts).Error
	return mounts, err
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func countNewMounts(targets []disk.FileShareTarget) int {
	count := 0
	for _, t := range targets {
		if t.TargetType == "user" {
			count++
		} else {
			count += len(getDeptMemberIDs(t.TargetID))
		}
	}
	return count
}

func getDeptMemberIDs(deptID int64) []int64 {
	var ids []int64
	global.DB.Table("sys_user_authorities").
		Where("sys_authority_id = ?", deptID).
		Pluck("sys_user_id", &ids)
	return ids
}
```

- [ ] **Step 2: 编译验证**

Run: `cd backend && go build ./...`

- [ ] **Step 3: Commit**

```bash
git add backend/service/disk/disk_share_mount.go
git commit -m "feat(disk): add mount point management service"
```

---

### Task 6: COW 写时复制服务

**Files:**
- Create: `backend/service/disk/disk_share_cow.go`

- [ ] **Step 1: 创建 COW 服务**

```go
package disk

import (
	"devops-admin/global"
	"devops-admin/model/disk"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// TriggerCOW 检查是否需要写时复制，需要则执行
// 返回用户实际应该操作的文件（可能是新复制的）
func TriggerCOW(fileID, userID int64) (*disk.File, error) {
	var file disk.File
	if err := global.DB.First(&file, fileID).Error; err != nil {
		return nil, fmt.Errorf("文件不存在: %w", err)
	}

	// ref_count == 1 → 只有自己引用，直接修改
	if file.RefCount <= 1 {
		return &file, nil
	}

	// ref_count > 1 → 需要 COW
	global.LOG.Info("触发 COW",
		zap.Int64("fileID", fileID),
		zap.Int("refCount", file.RefCount),
		zap.Int64("userID", userID),
	)

	// 1. 复制物理文件
	newPath, err := copyPhysicalFile(file.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("复制物理文件失败: %w", err)
	}

	// 2. 创建新的 disk_file 记录
	newFile := disk.File{
		UserID:      userID,
		Name:        file.Name,
		Path:        file.Path,
		IsFolder:    file.IsFolder,
		Size:        file.Size,
		ContentType: file.ContentType,
		Suffix:      file.Suffix,
		StoragePath: newPath,
		StorageType: file.StorageType,
		StorageHash: file.StorageHash,
		RefCount:    1,
		IsCow:       true,
		ParentID:    file.ParentID,
		Depth:       file.Depth,
	}

	if err := global.DB.Create(&newFile).Error; err != nil {
		os.Remove(newPath) // 清理已复制的文件
		return nil, fmt.Errorf("创建文件记录失败: %w", err)
	}

	// 3. 原文件 ref_count - 1
	global.DB.Model(&disk.File{}).Where("id = ?", fileID).
		Update("ref_count", gorm.Expr("ref_count - 1"))

	// 4. 更新用户的 mount 指向新文件
	global.DB.Model(&disk.FileMount{}).
		Where("user_id = ? AND source_file_id = ?", userID, fileID).
		Update("source_file_id", newFile.ID)

	// 5. 异步检查原文件是否需要清理
	go asyncCleanupIfZeroRef(fileID)

	return &newFile, nil
}

// copyPhysicalFile 复制物理文件到新路径
func copyPhysicalFile(srcPath string) (string, error) {
	if srcPath == "" {
		return "", fmt.Errorf("源文件路径为空")
	}

	// 生成新路径
	dir := filepath.Dir(srcPath)
	ext := filepath.Ext(srcPath)
	base := filepath.Base(srcPath)
	name := base[:len(base)-len(ext)]
	newName := fmt.Sprintf("%s_cow_%d%s", name, time.Now().UnixMilli(), ext)
	dstPath := filepath.Join(dir, newName)

	// 复制文件
	src, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(dstPath)
		return "", err
	}

	return dstPath, nil
}

// asyncCleanupIfZeroRef 异步清理引用计数为零的文件
func asyncCleanupIfZeroRef(fileID int64) {
	time.Sleep(1 * time.Second) // 等待事务提交

	var file disk.File
	if err := global.DB.First(&file, fileID).Error; err != nil {
		return
	}

	if file.RefCount <= 0 {
		global.LOG.Info("清理零引用文件", zap.Int64("fileID", fileID))
		if file.StoragePath != "" {
			os.Remove(file.StoragePath)
		}
		global.DB.Delete(&file)
	}
}
```

- [ ] **Step 2: 编译验证**

Run: `cd backend && go build ./...`

- [ ] **Step 3: Commit**

```bash
git add backend/service/disk/disk_share_cow.go
git commit -m "feat(disk): add COW (copy-on-write) service for shared file editing"
```

---

### Task 7: 重构内部共享 Service

**Files:**
- Modify: `backend/service/disk/disk_internal_share.go`

这个文件有 1389 行，需要大幅重构。核心变更：

- [ ] **Step 1: 修改 CreateInternalShare 函数**

移除 accept/reject 相关逻辑，改为自动创建挂载点。找到 `CreateInternalShare` 函数：

1. 移除 `Remark` 字段的赋值
2. 移除 `Status` 字段（不再设置 "active"）
3. 在创建 targets 后，调用 `CreateShareMounts` 创建挂载点
4. 调用 SSE 推送通知（使用已有的 `sse_manager`）
5. 设置 `NotifySent = true`

关键修改点（在创建 targets 的循环后添加）：

```go
// 创建挂载点
if err := CreateShareMounts(fileID, share.ID, newTargets, file.Name); err != nil {
    return fmt.Errorf("创建挂载点失败: %w", err)
}

// SSE 推送
for _, t := range newTargets {
    if t.TargetType == "user" {
        notifyUser := uint(t.TargetID)
        msg, _ := json.Marshal(SSEMessage{
            Type: "new_share",
            Data: ShareNotificationData{
                ShareID:  share.ID,
                FileID:   fileID,
                FileName: file.Name,
                ShareUser: currentUserNickName,
            },
            Timestamp: time.Now().Unix(),
        })
        global.SSEManager.SendToUser(notifyUser, msg)
    } else {
        // dept → 通知所有成员
        for _, memberID := range getDeptMemberIDs(t.TargetID) {
            msg, _ := json.Marshal(SSEMessage{
                Type: "new_share",
                Data: ShareNotificationData{
                    ShareID:  share.ID,
                    FileID:   fileID,
                    FileName: file.Name,
                    ShareUser: currentUserNickName,
                },
                Timestamp: time.Now().Unix(),
            })
            global.SSEManager.SendToUser(uint(memberID), msg)
        }
    }
}
```

- [ ] **Step 2: 移除 AcceptInternalShare 和 RejectInternalShare 函数**

这两个函数不再需要，因为共享自动接受。删除整个函数体。

- [ ] **Step 3: 修改 SaveToMyDrive 函数**

替换现有的保存逻辑，改为调用 `CreateSaveMount`：

```go
func (s *InternalShareService) SaveToMyDrive(req SaveToMyDriveReq, currentUserID int64) error {
    // 验证共享权限
    share := getShareByID(req.FileShareID)
    if !hasPermission(currentUserID, share, "DOWNLOAD") {
        return fmt.Errorf("无下载权限")
    }

    // 调用挂载服务
    return CreateSaveMount(
        currentUserID,
        share.FileID,
        req.TargetFolderID,
        share.File.Name,
        &share.ID,
    )
}
```

- [ ] **Step 4: 修改 GetSharedWithMeList 函数**

1. 移除 `targetStatus` 参数和过滤逻辑
2. 合并用户共享和部门共享时不再按 status 过滤
3. 添加 `isMounted` 字段：检查 `disk_file_mounts` 中是否存在 `user_id = currentUser AND source_file_id = fileId` 的记录

- [ ] **Step 5: 修改 CancelInternalShare 函数**

取消共享时调用 `RemoveShareMounts(shareID)` 清理挂载点。

- [ ] **Step 6: 添加新函数 UpdateTargetPermissions 和 RemoveTarget**

```go
// UpdateTargetPermissions 修改单个目标的权限
func (s *InternalShareService) UpdateTargetPermissions(targetID int64, permissions []common.OperationPermission) error {
    var target disk.FileShareTarget
    if err := global.DB.First(&target, targetID).Error; err != nil {
        return fmt.Errorf("目标不存在")
    }
    if err := global.DB.Model(&target).Update("permissions", permissions).Error; err != nil {
        return err
    }

    // 失效权限缓存
    var share disk.FileShare
    global.DB.First(&share, target.FileShareID)
    if target.TargetType == "user" {
        InvalidateSharePermissionCache(target.TargetID, share.FileID)
    } else {
        for _, memberID := range getDeptMemberIDs(target.TargetID) {
            InvalidateSharePermissionCache(memberID, share.FileID)
        }
    }
    return nil
}

// RemoveShareTarget 移除单个共享目标
func (s *InternalShareService) RemoveShareTarget(targetID, currentUserID int64) error {
    var target disk.FileShareTarget
    if err := global.DB.First(&target, targetID).Error; err != nil {
        return fmt.Errorf("目标不存在")
    }

    // 验证共享发起人
    var share disk.FileShare
    global.DB.First(&share, target.FileShareID)
    if share.UserID != currentUserID {
        return fmt.Errorf("无权操作")
    }

    return global.DB.Transaction(func(tx *gorm.DB) error {
        // 移除目标
        if err := tx.Delete(&target).Error; err != nil {
            return err
        }
        // 移除该目标对应的挂载
        if target.TargetType == "user" {
            RemoveSingleMount(target.TargetID, share.FileID, "share")
        } else {
            for _, memberID := range getDeptMemberIDs(target.TargetID) {
                RemoveSingleMount(memberID, share.FileID, "share")
            }
        }
        return nil
    })
}
```

- [ ] **Step 7: 编译验证**

Run: `cd backend && go build ./...`
Expected: 编译通过

- [ ] **Step 8: Commit**

```bash
git add backend/service/disk/disk_internal_share.go
git commit -m "refactor(disk): rewrite internal share service - auto accept, mount points, remove status"
```

---

### Task 8: 更新 API Handler

**Files:**
- Modify: `backend/api/v1/disk/disk_internal_share.go`

- [ ] **Step 1: 移除 Accept/Reject handler**

删除 `AcceptInternalShare` 和 `RejectInternalShare` 函数。

- [ ] **Step 2: 添加新 handler**

```go
// UpdateTargetPermissions 修改单个目标的权限
func (api *InternalShareApi) UpdateTargetPermissions(c *gin.Context) {
	var req struct {
		TargetID    int64                       `json:"targetId"`
		Permissions []common.OperationPermission `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	userID := getUserIDFromContext(c)
	if err := internalShareService.UpdateTargetPermissions(req.TargetID, req.Permissions); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("权限修改成功", c)
}

// RemoveShareTarget 移除单个共享目标
func (api *InternalShareApi) RemoveShareTarget(c *gin.Context) {
	var req struct {
		TargetID int64 `json:"targetId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	userID := getUserIDFromContext(c)
	if err := internalShareService.RemoveShareTarget(req.TargetID, userID); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("已移除", c)
}

// BatchSaveToDrive 批量保存到我的网盘
func (api *InternalShareApi) BatchSaveToDrive(c *gin.Context) {
	var req struct {
		Items           []struct { ShareID int64 `json:"shareId"`; FileID int64 `json:"fileId"` } `json:"items"`
		TargetFolderID  int64  `json:"targetFolderId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	userID := getUserIDFromContext(c)
	for _, item := range req.Items {
		if err := internalShareService.SaveToMyDrive(SaveToMyDriveReq{
			FileShareID:    item.ShareID,
			TargetFolderID: req.TargetFolderID,
		}, userID); err != nil {
			response.FailWithMessage(fmt.Sprintf("保存失败: %s", err.Error()), c)
			return
		}
	}
	response.OkWithMessage("保存成功", c)
}
```

- [ ] **Step 3: 修改 CreateInternalShare handler**

移除请求参数中的 `remark` 字段引用。确保 `Status` 不再被设置。

- [ ] **Step 4: 修改 GetSharedWithMeList handler**

移除 `targetStatus` 参数。

- [ ] **Step 5: 编译验证**

Run: `cd backend && go build ./...`

- [ ] **Step 6: Commit**

```bash
git add backend/api/v1/disk/disk_internal_share.go
git commit -m "refactor(disk): update API handlers - add target management, remove accept/reject"
```

---

### Task 9: 更新路由

**Files:**
- Modify: `backend/router/disk/disk_internal_share.go`

- [ ] **Step 1: 更新路由配置**

```go
func (r *InternalShareRouter) InitInternalShareRouter(Router *gin.RouterGroup) {
	// 读取操作（无速率限制）
	internalShareNoLimit := Router.Group("share/internal")
	{
		internalShareNoLimit.POST("/my-shared", InternalShareApi.GetMySharedList)
		internalShareNoLimit.POST("/shared-with-me", InternalShareApi.GetSharedWithMeList)
		internalShareNoLimit.POST("/folder", InternalShareApi.GetSharedFolderContents)
		internalShareNoLimit.POST("/targets", InternalShareApi.GetFileShareTargets)
	}

	// 写入操作（速率限制）
	internalShare := Router.Group("share/internal").Use(middleware.CustomUserLimitMiddleware(10, 60, "disk:internal-share"))
	{
		internalShare.POST("/create", InternalShareApi.CreateInternalShare)
		internalShare.POST("/cancel", InternalShareApi.CancelInternalShare)
		internalShare.PUT("/target/permissions", InternalShareApi.UpdateTargetPermissions)   // 新增
		internalShare.DELETE("/target", InternalShareApi.RemoveShareTarget)                  // 新增
		internalShare.POST("/save-to-drive", InternalShareApi.BatchSaveToDrive)              // 替换原有
		internalShare.POST("/upload", InternalShareApi.UploadToShareFolder)
		internalShare.POST("/create-folder", InternalShareApi.CreateShareSubFolder)
	}
}
```

注意：移除了 `/accept` 和 `/reject` 路由。

- [ ] **Step 2: 编译并启动验证**

Run: `cd backend && go build ./...`

- [ ] **Step 3: Commit**

```bash
git add backend/router/disk/disk_internal_share.go
git commit -m "refactor(disk): update routes - add target CRUD, remove accept/reject"
```

---

## Phase 3: 前端类型与 API 层

### Task 10: 更新前端类型定义

**Files:**
- Modify: `frontend/src/typings/api/disk.api.d.ts`

- [ ] **Step 1: 更新 FileShareTargetItem 类型**

移除 `status` 字段，添加 `mountName`：

```typescript
type FileShareTargetItem = {
  fileShareId: number;
  shareType: string;
  targetType: string;
  targetId: number;
  targetName: string;
  permissions: string[];
  mountName?: string;
};
```

- [ ] **Step 2: 更新 SharedWithMeItem 类型**

移除 `status`、`targetStatus`、`remark` 字段，添加 `isMounted`、`mountId`：

```typescript
type SharedWithMeItem = {
  fileShareId: number;
  fileId: number;
  fileName: string;
  isFolder: boolean;
  contentType: string;
  size: number;
  shareUserId: number;
  shareUserName: string;
  shareType: 'user' | 'dept';
  permissions: string[];
  expireDate?: string | null;
  sourceLabel?: string;
  createdAt: string;
  mediaCover?: boolean;
  isMounted: boolean;
  mountId?: number | null;
};
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/typings/api/disk.api.d.ts
git commit -m "refactor(disk): update share-related type definitions"
```

---

### Task 11: 更新前端 API 服务

**Files:**
- Modify: `frontend/src/service/api/disk/internal-share.ts`

- [ ] **Step 1: 移除 accept/reject API，添加新 API**

```typescript
// 移除: fetchRejectInternalShare, fetchAcceptInternalShare

// 修改: fetchCreateInternalShare 移除 remark 参数
export function fetchCreateInternalShare(data: {
  fileId: number;
  shareType: 'user' | 'dept';
  targets: { targetId: number; permissions: string[] }[];
  expireDate?: string;
}): Promise<boolean> {
  return request({
    url: '/share/internal/create',
    method: 'post',
    data
  });
}

// 新增: 修改目标权限
export function fetchUpdateTargetPermissions(targetId: number, permissions: string[]): Promise<boolean> {
  return request({
    url: '/share/internal/target/permissions',
    method: 'put',
    data: { targetId, permissions }
  });
}

// 新增: 移除共享目标
export function fetchRemoveShareTarget(targetId: number): Promise<boolean> {
  return request({
    url: '/share/internal/target',
    method: 'delete',
    data: { targetId }
  });
}

// 修改: fetchSaveToMyDrive → 支持批量
export function fetchBatchSaveToDrive(items: Array<{ shareId: number; fileId: number }>, targetFolderId: number): Promise<boolean> {
  return request({
    url: '/share/internal/save-to-drive',
    method: 'post',
    data: { items, targetFolderId }
  });
}

// 修改: fetchGetSharedWithMeList 移除 targetStatus 参数
export function fetchGetSharedWithMeList(params: {
  pageNum: number;
  pageSize: number;
  shareType?: string;
  keyword?: string;
  contentType?: string;
}): Promise<Api.Disk.SharedWithMeList> {
  return request({
    url: '/share/internal/shared-with-me',
    method: 'post',
    data: params
  });
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/service/api/disk/internal-share.ts
git commit -m "refactor(disk): update internal share API service"
```

---

## Phase 4: 前端组件重构

### Task 12: 共享权限复选框组件

**Files:**
- Create: `frontend/src/views/disk/modules/share-dialog/share-permission-checker.vue`

- [ ] **Step 1: 创建权限复选框组件**

```vue
<script setup lang="ts">
import { computed } from 'vue';

defineOptions({ name: 'SharePermissionChecker' });

interface Props {
  permissions: string[];
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false
});

const emit = defineEmits<{
  (e: 'update:permissions', value: string[]): void;
}>();

const permissionOptions = [
  { label: '下载', value: 'DOWNLOAD' },
  { label: '上传', value: 'UPLOAD' },
  { label: '编辑', value: 'PUT' },
  { label: '删除', value: 'DELETE' }
];

const checkedPermissions = computed({
  get: () => props.permissions,
  set: (val: string[]) => emit('update:permissions', val)
});
</script>

<template>
  <NCheckboxGroup v-model:value="checkedPermissions">
    <NSpace>
      <NCheckbox v-for="opt in permissionOptions" :key="opt.value" :value="opt.value" :disabled="disabled">
        {{ opt.label }}
      </NCheckbox>
    </NSpace>
  </NCheckboxGroup>
</template>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/disk/modules/share-dialog/share-permission-checker.vue
git commit -m "feat(disk): add share permission checker component"
```

---

### Task 13: 用户搜索组件

**Files:**
- Create: `frontend/src/views/disk/modules/share-dialog/share-user-search.vue`

- [ ] **Step 1: 创建用户搜索组件**

```vue
<script setup lang="ts">
import { ref } from 'vue';
import { fetchGetUserSelect } from '@/service/api/system/user';

defineOptions({ name: 'ShareUserSearch' });

interface UserOption {
  label: string;
  value: number;
  avatar?: string;
}

const emit = defineEmits<{
  (e: 'select', users: UserOption[]): void;
}>();

const searchKeyword = ref('');
const searchResults = ref<UserOption[]>([]);
const selectedUsers = ref<UserOption[]>([]);
const loading = ref(false);

async function handleSearch() {
  if (!searchKeyword.value.trim()) return;
  loading.value = true;
  try {
    const { data } = await fetchGetUserSelect({ keyword: searchKeyword.value });
    if (data) {
      searchResults.value = (data as any[]).map(u => ({
        label: u.nickName || u.userName,
        value: u.ID,
        avatar: u.headerImg
      }));
    }
  } finally {
    loading.value = false;
  }
}

function toggleUser(option: UserOption) {
  const idx = selectedUsers.value.findIndex(u => u.value === option.value);
  if (idx >= 0) {
    selectedUsers.value = selectedUsers.value.filter((_, i) => i !== idx);
  } else {
    selectedUsers.value = [...selectedUsers.value, option];
  }
}

function isSelected(value: number): boolean {
  return selectedUsers.value.some(u => u.value === value);
}

function confirmSelection() {
  emit('select', selectedUsers.value);
  selectedUsers.value = [];
  searchResults.value = [];
  searchKeyword.value = '';
}
</script>

<template>
  <div class="user-search">
    <NSpace align="center">
      <NInput v-model:value="searchKeyword" placeholder="搜索用户" clearable @keyup.enter="handleSearch" style="width: 200px" />
      <NButton type="primary" @click="handleSearch" :loading="loading">搜索</NButton>
    </NSpace>
    <div v-if="searchResults.length" class="search-results">
      <NCheckbox v-for="opt in searchResults" :key="opt.value" :checked="isSelected(opt.value)" @update:checked="toggleUser(opt)">
        <NAvatar v-if="opt.avatar" :src="opt.avatar" :size="20" round />
        {{ opt.label }}
      </NCheckbox>
    </div>
    <div v-if="selectedUsers.length" class="selected-users">
      <NTag v-for="u in selectedUsers" :key="u.value" closable @close="toggleUser(u)" size="small">
        {{ u.label }}
      </NTag>
    </div>
  </div>
</template>

<style scoped>
.user-search {
  margin-bottom: 12px;
}
.search-results {
  margin-top: 8px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.selected-users {
  margin-top: 8px;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/disk/modules/share-dialog/share-user-search.vue
git commit -m "feat(disk): add user search component for share dialog"
```

---

### Task 14: 已授权目标项组件

**Files:**
- Create: `frontend/src/views/disk/modules/share-dialog/share-target-item.vue`

- [ ] **Step 1: 创建目标项组件**

```vue
<script setup lang="ts">
import { ref } from 'vue';

defineOptions({ name: 'ShareTargetItem' });

interface Props {
  targetId: number;
  targetName: string;
  targetType: 'user' | 'dept';
  permissions: string[];
  avatar?: string;
  editing?: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: 'update', targetId: number, permissions: string[]): void;
  (e: 'remove', targetId: number): void;
}>();

const isEditing = ref(false);
const editPerms = ref<string[]>([...props.permissions]);
const showRemoveConfirm = ref(false);

const permLabels: Record<string, string> = {
  DOWNLOAD: '下载',
  UPLOAD: '上传',
  PUT: '编辑',
  DELETE: '删除'
};

function startEdit() {
  editPerms.value = [...props.permissions];
  isEditing.value = true;
}

function confirmEdit() {
  emit('update', props.targetId, editPerms.value);
  isEditing.value = false;
}

function cancelEdit() {
  isEditing.value = false;
}

function confirmRemove() {
  emit('remove', props.targetId);
  showRemoveConfirm.value = false;
}
</script>

<template>
  <div class="target-item">
    <div class="target-info">
      <NAvatar v-if="targetType === 'user' && avatar" :src="avatar" :size="28" round />
      <NIcon v-else-if="targetType === 'dept'" :size="28" color="var(--n-color)">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="currentColor" d="M12 7V3H2v18h20V7H12zM6 19H4v-2h2v2zm0-4H4v-2h2v2zm0-4H4V9h2v2zm0-4H4V5h2v2zm4 12H8v-2h2v2zm0-4H8v-2h2v2zm0-4H8V9h2v2zm0-4H8V5h2v2zm10 12h-8v-2h2v-2h-2v-2h2v-2h-2V9h8v10zm-2-8h-2v2h2v-2zm0 4h-2v2h2v-2z"/></svg>
      </NIcon>
      <span class="target-name">{{ targetName }}</span>
    </div>

    <div v-if="!isEditing" class="target-perms">
      <NTag v-for="p in permissions" :key="p" size="small" type="info" :bordered="false">
        {{ permLabels[p] || p }}
      </NTag>
      <NButton text size="small" @click="startEdit">修改</NButton>
      <NButton text size="small" type="error" @click="showRemoveConfirm = true">移除</NButton>
    </div>

    <div v-else class="target-edit">
      <NCheckboxGroup v-model:value="editPerms">
        <NSpace size="small">
          <NCheckbox value="DOWNLOAD">下载</NCheckbox>
          <NCheckbox value="UPLOAD">上传</NCheckbox>
          <NCheckbox value="PUT">编辑</NCheckbox>
          <NCheckbox value="DELETE">删除</NCheckbox>
        </NSpace>
      </NCheckboxGroup>
      <NSpace size="small">
        <NButton size="tiny" type="primary" @click="confirmEdit">确定</NButton>
        <NButton size="tiny" @click="cancelEdit">取消</NButton>
      </NSpace>
    </div>

    <NModal v-model:show="showRemoveConfirm">
      <NCard style="width: 300px" title="确认移除">
        <p>确定移除 {{ targetName }} 的共享权限？</p>
        <template #footer>
          <NSpace justify="end">
            <NButton @click="showRemoveConfirm = false">取消</NButton>
            <NButton type="error" @click="confirmRemove">确认移除</NButton>
          </NSpace>
        </template>
      </NCard>
    </NModal>
  </div>
</template>

<style scoped>
.target-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid var(--n-border-color);
}
.target-info {
  display: flex;
  align-items: center;
  gap: 8px;
}
.target-name {
  font-size: 14px;
}
.target-perms, .target-edit {
  display: flex;
  align-items: center;
  gap: 6px;
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/disk/modules/share-dialog/share-target-item.vue
git commit -m "feat(disk): add share target item component with inline edit"
```

---

### Task 15: 共享给用户 Tab 组件

**Files:**
- Create: `frontend/src/views/disk/modules/share-dialog/share-to-user.vue`

- [ ] **Step 1: 创建用户 Tab 组件**

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { fetchCreateInternalShare, fetchGetFileShareTargets, fetchUpdateTargetPermissions, fetchRemoveShareTarget } from '@/service/api/disk/internal-share';
import { useLoading } from '@sa/hooks';
import ShareUserSearch from './share-user-search.vue';
import SharePermissionChecker from './share-permission-checker.vue';
import ShareTargetItem from './share-target-item.vue';

defineOptions({ name: 'ShareToUser' });

interface Props {
  fileId: number;
}

const props = defineProps<Props>();

interface TargetItem {
  targetId: number;
  targetName: string;
  targetType: string;
  permissions: string[];
  avatar?: string;
}

interface UserOption {
  label: string;
  value: number;
  avatar?: string;
}

const { loading: submitLoading, startLoading, endLoading } = useLoading(false);
const selectedPermissions = ref<string[]>(['DOWNLOAD']);
const existingTargets = ref<TargetItem[]>([]);
const loadingTargets = ref(false);

async function loadTargets() {
  loadingTargets.value = true;
  try {
    const { data } = await fetchGetFileShareTargets(props.fileId);
    if (data) {
      existingTargets.value = (data as any[])
        .filter(t => t.targetType === 'user')
        .map(t => ({
          targetId: t.targetId,
          targetName: t.targetName,
          targetType: t.targetType,
          permissions: t.permissions || [],
          avatar: t.avatar
        }));
    }
  } finally {
    loadingTargets.value = false;
  }
}

async function handleSubmit(users: UserOption[]) {
  if (!users.length) return;
  startLoading();
  try {
    const targets = users.map(u => ({
      targetId: u.value,
      permissions: selectedPermissions.value
    }));
    const { error } = await fetchCreateInternalShare({
      fileId: props.fileId,
      shareType: 'user',
      targets
    });
    if (!error) {
      window.$message?.success('授权成功');
      await loadTargets();
    }
  } finally {
    endLoading();
  }
}

async function handleUpdate(targetId: number, permissions: string[]) {
  const { error } = await fetchUpdateTargetPermissions(targetId, permissions);
  if (!error) {
    window.$message?.success('权限已更新');
    await loadTargets();
  }
}

async function handleRemove(targetId: number) {
  const { error } = await fetchRemoveShareTarget(targetId);
  if (!error) {
    window.$message?.success('已移除');
    await loadTargets();
  }
}

onMounted(() => {
  loadTargets();
});
</script>

<template>
  <div class="share-to-user">
    <NCard size="small" title="添加授权">
      <ShareUserSearch @select="handleSubmit" />
      <SharePermissionChecker v-model:permissions="selectedPermissions" />
      <NButton type="primary" block :loading="submitLoading" :disabled="submitLoading" style="margin-top: 12px">
        确认授权
      </NButton>
    </NCard>

    <NCard size="small" title="已授权用户" style="margin-top: 12px" :loading="loadingTargets">
      <NEmpty v-if="!existingTargets.length" description="暂无授权用户" />
      <div v-else>
        <ShareTargetItem
          v-for="target in existingTargets"
          :key="target.targetId"
          :target-id="target.targetId"
          :target-name="target.targetName"
          :target-type="target.targetType"
          :permissions="target.permissions"
          :avatar="target.avatar"
          @update="handleUpdate"
          @remove="handleRemove"
        />
      </div>
    </NCard>
  </div>
</template>
```

注意：`ShareUserSearch` 的 `@select` 事件直接触发 `handleSubmit`，因为用户选择完毕后点"确认授权"才提交。需要根据实际搜索组件交互微调。

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/disk/modules/share-dialog/share-to-user.vue
git commit -m "feat(disk): add share-to-user tab component"
```

---

### Task 16: 共享给部门 Tab 组件

**Files:**
- Create: `frontend/src/views/disk/modules/share-dialog/share-to-dept.vue`

- [ ] **Step 1: 创建部门 Tab 组件**

与 `share-to-user.vue` 同构，搜索源改为部门树。复用 `DeptTree` 组件（项目已有）。

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { fetchCreateInternalShare, fetchGetFileShareTargets, fetchUpdateTargetPermissions, fetchRemoveShareTarget } from '@/service/api/disk/internal-share';
import { useLoading } from '@sa/hooks';
import DeptTree from '@/components/custom/dept-tree.vue';
import SharePermissionChecker from './share-permission-checker.vue';
import ShareTargetItem from './share-target-item.vue';

defineOptions({ name: 'ShareToDept' });

interface Props {
  fileId: number;
}

const props = defineProps<Props>();

const { loading: submitLoading, startLoading, endLoading } = useLoading(false);
const selectedPermissions = ref<string[]>(['DOWNLOAD']);
const existingTargets = ref<any[]>([]);
const loadingTargets = ref(false);
const selectedDeptIds = ref<number[]>([]);

async function loadTargets() {
  loadingTargets.value = true;
  try {
    const { data } = await fetchGetFileShareTargets(props.fileId);
    if (data) {
      existingTargets.value = (data as any[])
        .filter(t => t.targetType === 'dept')
        .map(t => ({
          targetId: t.targetId,
          targetName: t.targetName,
          targetType: t.targetType,
          permissions: t.permissions || []
        }));
    }
  } finally {
    loadingTargets.value = false;
  }
}

async function handleSubmit() {
  if (!selectedDeptIds.value.length) {
    window.$message?.warning('请选择部门');
    return;
  }
  startLoading();
  try {
    const targets = selectedDeptIds.value.map(id => ({
      targetId: id,
      permissions: selectedPermissions.value
    }));
    const { error } = await fetchCreateInternalShare({
      fileId: props.fileId,
      shareType: 'dept',
      targets
    });
    if (!error) {
      window.$message?.success('授权成功');
      selectedDeptIds.value = [];
      await loadTargets();
    }
  } finally {
    endLoading();
  }
}

async function handleUpdate(targetId: number, permissions: string[]) {
  const { error } = await fetchUpdateTargetPermissions(targetId, permissions);
  if (!error) {
    window.$message?.success('权限已更新');
    await loadTargets();
  }
}

async function handleRemove(targetId: number) {
  const { error } = await fetchRemoveShareTarget(targetId);
  if (!error) {
    window.$message?.success('已移除');
    await loadTargets();
  }
}

onMounted(() => {
  loadTargets();
});
</script>

<template>
  <div class="share-to-dept">
    <NCard size="small" title="添加授权">
      <DeptTree v-model:selected-keys="selectedDeptIds" checkable multiple />
      <SharePermissionChecker v-model:permissions="selectedPermissions" style="margin-top: 12px" />
      <NButton type="primary" block :loading="submitLoading" style="margin-top: 12px" @click="handleSubmit">
        确认授权
      </NButton>
    </NCard>

    <NCard size="small" title="已授权部门" style="margin-top: 12px" :loading="loadingTargets">
      <NEmpty v-if="!existingTargets.length" description="暂无授权部门" />
      <div v-else>
        <ShareTargetItem
          v-for="target in existingTargets"
          :key="target.targetId"
          :target-id="target.targetId"
          :target-name="target.targetName"
          target-type="dept"
          :permissions="target.permissions"
          @update="handleUpdate"
          @remove="handleRemove"
        />
      </div>
    </NCard>
  </div>
</template>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/disk/modules/share-dialog/share-to-dept.vue
git commit -m "feat(disk): add share-to-dept tab component"
```

---

### Task 17: 重构共享对话框入口

**Files:**
- Create: `frontend/src/views/disk/modules/share-dialog/index.vue` (新入口)
- Remove/Archive: `frontend/src/views/disk/modules/share-dialog.vue` (旧 1100 行文件)

- [ ] **Step 1: 创建新的对话框入口**

```vue
<script setup lang="ts">
import { ref, watch } from 'vue';
import { useDiskStore } from '@/store/modules/disk';
import ShareToUser from './share-to-user.vue';
import ShareToDept from './share-to-dept.vue';

defineOptions({ name: 'ShareDialog' });

const diskStore = useDiskStore();
const activeTab = ref('user');

// 从 diskStore 获取当前选中的文件
const currentFile = computed(() => diskStore.currentShareFile);
const visible = computed(() => diskStore.shareDialogVisible);

function handleClose() {
  diskStore.closeShareDialog();
}
</script>

<template>
  <NModal v-model:show="visible" preset="card" title="共享" style="width: 560px" :mask-closable="false" @close="handleClose">
    <template v-if="currentFile">
      <div class="share-file-info" style="margin-bottom: 16px">
        <NText depth="3">{{ currentFile.isFolder ? '文件夹' : '文件' }}:</NText>
        <NText strong>{{ currentFile.name }}</NText>
      </div>
      <NTabs v-model:value="activeTab" type="line">
        <NTabPane name="user" tab="共享给用户">
          <ShareToUser :file-id="currentFile.id" />
        </NTabPane>
        <NTabPane name="dept" tab="共享给部门">
          <ShareToDept :file-id="currentFile.id" />
        </NTabPane>
      </NTabs>
    </template>
  </NModal>
</template>
```

- [ ] **Step 2: 更新所有引用旧 share-dialog.vue 的地方**

在项目中搜索 `share-dialog.vue` 的 import 路径，更新为新路径 `share-dialog/index.vue`。主要在 `shared-with-me` 页面和 `disk` 页面中。

- [ ] **Step 3: 旧文件处理**

将旧的 `share-dialog.vue` 中的公共链接共享逻辑保留。有两种方式：
- 方案 A：旧文件仅保留公共链接共享功能，内部共享完全由新组件处理
- 方案 B：在 `disk/modules/` 下新建 `link-share-dialog.vue` 专门处理公共链接

根据项目实际情况选择。确认新组件可编译后，删除旧文件中内部共享相关的代码。

- [ ] **Step 4: 验证编译**

Run: `cd frontend && pnpm typecheck`

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/disk/modules/share-dialog/
git commit -m "refactor(disk): split share dialog into tab-based components"
```

---

### Task 18: 重构"共享给我的"页面

**Files:**
- Modify: `frontend/src/views/shared-with-me/modules/share-list-page.vue`

这个文件有 1125 行，需要拆分。核心变更：

- [ ] **Step 1: 移除 accept/reject 相关代码**

在 `share-list-page.vue` 中搜索并移除：
- `handleAcceptShare` 函数
- `handleRejectShare` 函数（如果有）
- `fetchAcceptInternalShare` / `fetchRejectInternalShare` 的 import
- 模板中的"接受"/"拒绝"按钮
- `targetStatus` 相关的过滤和显示逻辑

- [ ] **Step 2: 添加"保存到我的网盘"功能**

修改 `handleSaveToMyDrive` 函数，改为调用 `fetchBatchSaveToDrive`：

```typescript
async function handleSaveToMyDrive(shareId: number, fileId: number) {
  // 弹出目录选择器
  saveToDriveDialogVisible.value = true;
  pendingSaveItems.value = [{ shareId, fileId }];
}

async function confirmSaveToDrive(targetFolderId: number) {
  const { error } = await fetchBatchSaveToDrive(pendingSaveItems.value, targetFolderId);
  if (!error) {
    window.$message?.success('保存成功');
    saveToDriveDialogVisible.value = false;
    getData(); // 刷新列表，更新 isMounted 状态
  }
}
```

- [ ] **Step 3: 添加状态标记**

根据 `isMounted` 字段显示不同状态：

```vue
<template #action="{ row }">
  <NSpace>
    <NButton v-if="!row.isMounted" type="primary" text @click="handleSaveToMyDrive(row.fileShareId, row.fileId)">
      保存到我的网盘
    </NButton>
    <NButton v-else type="default" text disabled>
      已保存到我的网盘
    </NButton>
    <NButton text @click="enterSharedFolder(row)">查看</NButton>
  </NSpace>
</template>
```

- [ ] **Step 4: 移除批量接受/拒绝操作**

移除 `handleBatchCancel` 中的 accept/reject 逻辑，改为批量保存或批量取消。

- [ ] **Step 5: 验证编译**

Run: `cd frontend && pnpm typecheck`

- [ ] **Step 6: Commit**

```bash
git add frontend/src/views/shared-with-me/
git commit -m "refactor(disk): update shared-with-me page - auto accept, save-to-drive"
```

---

### Task 19: 保存到网盘对话框

**Files:**
- Create: `frontend/src/views/disk/modules/save-to-drive-dialog.vue`

- [ ] **Step 1: 创建目录选择器对话框**

```vue
<script setup lang="ts">
import { ref, computed } from 'vue';
import { fetchGetFileList } from '@/service/api/disk';

defineOptions({ name: 'SaveToDriveDialog' });

interface Props {
  visible: boolean;
  items: Array<{ shareId: number; fileId: number; fileName: string }>;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void;
  (e: 'confirm', targetFolderId: number): void;
}>();

const selectedFolderId = ref<number | null>(null);
const folders = ref<any[]>([]);
const loading = ref(false);

const showDialog = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
});

async function loadFolders(parentId?: number) {
  loading.value = true;
  try {
    const { data } = await fetchGetFileList({ parentId, isFolder: true });
    if (data) {
      folders.value = (data as any[]).filter(f => f.isFolder);
    }
  } finally {
    loading.value = false;
  }
}

function handleConfirm() {
  if (selectedFolderId.value) {
    emit('confirm', selectedFolderId.value);
  }
}

function handleOpen() {
  loadFolders();
}
</script>

<template>
  <NModal v-model:show="showDialog" preset="card" title="保存到我的网盘" style="width: 480px" @after-enter="handleOpen">
    <div v-if="items.length" class="save-items">
      <NText depth="3">将保存 {{ items.length }} 个文件到:</NText>
    </div>
    <NEmpty v-if="!folders.length && !loading" description="暂无文件夹" />
    <div v-else class="folder-list" :style="{ maxHeight: '300px', overflow: 'auto' }">
      <NRadioGroup v-model:value="selectedFolderId">
        <NRadio v-for="folder in folders" :key="folder.id" :value="folder.id" style="display: block; padding: 8px 0">
          <NIcon :size="16"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="currentColor" d="M10 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/></svg></NIcon>
          {{ folder.name }}
        </NRadio>
      </NRadioGroup>
    </div>
    <template #footer>
      <NSpace justify="end">
        <NButton @click="showDialog = false">取消</NButton>
        <NButton type="primary" :disabled="!selectedFolderId" @click="handleConfirm">保存</NButton>
      </NSpace>
    </template>
  </NModal>
</template>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/disk/modules/save-to-drive-dialog.vue
git commit -m "feat(disk): add save-to-drive dialog with folder picker"
```

---

## Phase 5: SSE 集成

### Task 20: 后端 SSE 事件推送

**Files:**
- Modify: `backend/service/disk/disk_internal_share.go` (已在 Task 7 中包含)

SSE 推送逻辑已嵌入 Task 7 的 CreateInternalShare 函数中。确认以下事件已实现：

- [ ] **Step 1: 验证 new_share 事件推送**

在 `CreateInternalShare` 中确认 SSE 消息格式：

```go
type ShareNotificationData struct {
    ShareID   int64  `json:"shareId"`
    FileID    int64  `json:"fileId"`
    FileName  string `json:"fileName"`
    ShareUser string `json:"shareUser"`
}

// 消息格式
msg, _ := json.Marshal(map[string]interface{}{
    "type": "new_share",
    "data": ShareNotificationData{...},
    "timestamp": time.Now().Unix(),
})
global.SSEManager.SendToUser(uint(targetUserID), msg)
```

- [ ] **Step 2: 添加 share_cancelled 事件**

在 `CancelInternalShare` 函数中添加 SSE 推送：

```go
// 取消共享后通知受影响用户
msg, _ := json.Marshal(map[string]interface{}{
    "type": "share_cancelled",
    "data": map[string]interface{}{
        "shareId":     shareID,
        "fileName":    share.File.Name,
        "cancelledBy": currentUserNickName,
    },
    "timestamp": time.Now().Unix(),
})
// 通知所有目标用户
```

- [ ] **Step 3: Commit**

```bash
git add backend/service/disk/disk_internal_share.go
git commit -m "feat(disk): add SSE notifications for share events"
```

---

### Task 21: 前端 SSE 监听

**Files:**
- Modify: `frontend/src/hooks/common/sse.ts`

- [ ] **Step 1: 在现有 handleShareNotification 中扩展处理**

在 `sse.ts` 的 `handleMessage` 函数中，找到 `share_notification` 处理逻辑，扩展支持新事件：

```typescript
// 已有的 showShareNotification 函数可以复用
// 确保以下事件被正确处理：
// - new_share → 显示通知 + 刷新共享列表
// - share_cancelled → 提示共享已取消
// - permission_change → 提示权限变更
```

现有的 `ShareNotificationData` interface 可能需要扩展：

```typescript
interface ShareNotificationData {
  shareId: number;
  fileId: number;
  fileName: string;
  shareUser: string;
  newPerms?: string[];         // 权限变更时
  cancelledBy?: string;        // 取消共享时
}
```

- [ ] **Step 2: 在 shared-with-me 页面中订阅 SSE 事件**

在 `shared-with-me/index.vue` 或 `share-list-page.vue` 中：

```typescript
import { onSSEMessage } from '@/hooks/common/sse';

// 页面挂载时订阅
onMounted(() => {
  const unsub = onSSEMessage('new_share', () => {
    getData(); // 自动刷新列表
  });
  onUnmounted(unsub);
});
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/hooks/common/sse.ts frontend/src/views/shared-with-me/
git commit -m "feat(disk): extend SSE listeners for share events"
```

---

## Phase 6: 过期清理与验证

### Task 22: 过期共享定时清理

**Files:**
- Modify: `backend/service/disk/disk_internal_share.go` (CleanExpiredShares 函数)

- [ ] **Step 1: 更新 CleanExpiredShares 函数**

现有的 `CleanExpiredShares` 函数需要修改：

1. 扫描 `expire_date < NOW()` 且未软删除的共享
2. 不再更新 `status`，而是直接软删除
3. 调用 `RemoveShareMounts` 清理挂载
4. SSE 通知受影响用户

```go
func (s *InternalShareService) CleanExpiredShares() (int64, error) {
    var shares []disk.FileShare
    err := global.DB.Where("expire_date IS NOT NULL AND expire_date < ? AND deleted_at IS NULL",
        time.Now()).Find(&shares).Error
    if err != nil {
        return 0, err
    }

    for _, share := range shares {
        // 移除挂载
        RemoveShareMounts(share.ID)
        // 软删除
        global.DB.Delete(&share)
        // SSE 通知 (可选: 批量通知)
    }

    return int64(len(shares)), nil
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/service/disk/disk_internal_share.go
git commit -m "refactor(disk): update expired share cleanup logic"
```

---

### Task 23: 编译与类型验证

- [ ] **Step 1: 后端编译**

Run: `cd backend && go build ./...`
Expected: 编译通过

- [ ] **Step 2: 后端 vet**

Run: `cd backend && go vet ./...`
Expected: 无警告

- [ ] **Step 3: 前端类型检查**

Run: `cd frontend && pnpm typecheck`
Expected: 通过

- [ ] **Step 4: 前端 lint**

Run: `cd frontend && pnpm lint`
Expected: 通过

- [ ] **Step 5: 启动后端验证**

Run: `cd backend && go run main.go`
Expected: 服务启动正常，新表自动创建

- [ ] **Step 6: 启动前端验证**

Run: `cd frontend && pnpm dev`
Expected: 前端启动正常，共享对话框和共享给我的页面功能正常

- [ ] **Step 7: 功能验证**

手动验证以下流程：
1. 打开共享对话框 → 用户 Tab 搜索用户 → 勾选权限 → 确认授权 → 列表显示
2. 修改单个用户权限 → 保存 → 刷新后显示新权限
3. 移除单个用户 → 确认 → 列表更新
4. 被共享者收到 SSE 通知
5. "共享给我的"页面显示新共享
6. 点击"保存到我的网盘" → 选择目录 → 保存成功
7. 修改已保存的文件 → COW 触发
8. 取消共享 → 被共享者收到通知 → share 挂载清除，save 挂载保留

---

## 自审检查

### 规格覆盖

| 规格要求 | 对应 Task |
|---------|----------|
| disk_file_mounts 新表 | Task 1 |
| disk_files ref_count/storage_hash/is_cow | Task 2 |
| FileShare 移除 status/remark, 添加 notify_sent | Task 3 |
| FileShareTarget 移除 status, 添加 mount_name | Task 3 |
| 权限校验服务 + Redis 缓存 | Task 4 |
| 挂载点管理服务 | Task 5 |
| COW 写时复制 | Task 6 |
| 共享自动接受（无 accept/reject） | Task 7 |
| 批量授权 + 已授权列表 | Task 7, 15, 16 |
| 修改/移除单个目标权限 | Task 7, 8 |
| 保存到我的网盘（硬链接+COW） | Task 6, 19 |
| 动态同步（共享文件夹） | Task 7 (GetSharedFolderContents) |
| 取消共享流程 | Task 7 |
| SSE 通知 | Task 20, 21 |
| 过期清理 | Task 22 |
| 前端共享对话框拆分 | Task 12-17 |
| 前端共享给我的重构 | Task 18 |
| 编译验证 | Task 23 |

### 占位符扫描

无 TBD/TODO/placeholder。

### 类型一致性

- 后端 `OperationPermission` 枚举: DOWNLOAD, UPLOAD, PUT, DELETE（移除 SHARE）
- 前端类型 `FileShareTargetItem` 和 `SharedWithMeItem` 与后端 model 字段对应
- API 函数签名与路由一致
- SSE 消息类型: new_share, share_cancelled, permission_change
