package disk

import (
	"time"

	"github.com/hllkk/devops-admin/server/global"
)

// 上传会话状态
const (
	UploadStatusUploading = "uploading" // 收片中
	UploadStatusMerging   = "merging"   // 合并中(原子抢占)
	UploadStatusCompleted = "completed" // 已完成
	UploadStatusFailed    = "failed"    // 合并失败
)

// DiskUploadSession 网盘大文件上传会话(断点续传 + 秒传判定)。
// identifier = quickHash(前端 Web Worker 采样 MD5),作会话去重键;
// 续传状态 DB 落库(D2 决策),多实例/重启可恢复(不学 remote/jmal 纯内存)。
type DiskUploadSession struct {
	global.OPS_AUDIT_MODEL
	UploadId         int64  `json:"uploadId,string" gorm:"primarykey;comment:上传会话ID"`
	UserId           int64  `json:"userId,string" gorm:"index;comment:用户ID"`
	Identifier       string `json:"identifier" gorm:"size:64;index;comment:会话标识(quickHash)"`
	FileName         string `json:"fileName" gorm:"size:255;comment:文件名"`
	RelativePath     string `json:"relativePath" gorm:"size:512;comment:文件夹上传相对路径"`
	TotalSize        int64  `json:"totalSize" gorm:"comment:总字节数"`
	ChunkSize        int64  `json:"chunkSize" gorm:"comment:分片大小"`
	TotalChunks      int    `json:"totalChunks" gorm:"comment:分片总数"`
	CurrentDirectory string `json:"currentDirectory" gorm:"size:512;comment:目标父目录全路径"`
	Status           string `json:"status" gorm:"size:16;index;comment:状态 uploading/merging/completed/failed"`
	QuickHash        string `json:"quickHash" gorm:"size:32;index;comment:采样MD5(秒传初筛)"`
	StrongHash       string `json:"strongHash" gorm:"size:64;comment:采样SHA256(防碰撞)"`
	MergedMd5        string `json:"mergedMd5" gorm:"size:32;comment:合并后完整MD5(权威指纹)"`
	FileId           int64  `json:"fileId,string" gorm:"comment:合并后生成的 disk_files.file_id"`
	StorageKey       string `json:"storageKey" gorm:"size:512;comment:OSS 对象 key"`
}

// TableName 表名
func (DiskUploadSession) TableName() string {
	return "disk_upload_sessions"
}

// DiskUploadChunk 网盘上传分片记录(幂等收片:唯一索引 upload_id+chunk_number,ON CONFLICT DO NOTHING)。
type DiskUploadChunk struct {
	ID          int64     `json:"id,string" gorm:"primarykey"`
	UploadId    int64     `json:"uploadId,string" gorm:"uniqueIndex:idx_disk_upload_chunk;comment:会话ID"`
	ChunkNumber int       `json:"chunkNumber" gorm:"uniqueIndex:idx_disk_upload_chunk;comment:分片序号(0-based)"`
	ChunkHash   string    `json:"chunkHash" gorm:"size:32;comment:分片MD5"`
	Size        int64     `json:"size" gorm:"comment:分片字节数"`
	CreatedAt   time.Time `json:"createTime" gorm:"column:create_time;comment:创建时间"`
}

// TableName 表名
func (DiskUploadChunk) TableName() string {
	return "disk_upload_chunks"
}
