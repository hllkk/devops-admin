package disk

import "github.com/hllkk/devops-admin/server/global"

// DiskFile 网盘文件/目录节点(目录树 + 上传元数据合一)。
//
// 阶段0(只读列表)仅用目录树字段:user_id/parent_id/name/is_directory/path/size/extend_name/content_type/is_favorite/status。
// md5/quick_hash/strong_hash/storage_type/storage_path/ref_count 为第3期上传体系预留(秒传/断点续传/去重/存储落点),
// 表为新建,AutoMigrate 一次性建齐,后续上传落地无需追加列迁移。
//
// 字段经 model/disk/response.FileItem DTO 转换后对齐前端 Api.Disk.BackendFileItem。
type DiskFile struct {
	global.OPS_AUDIT_MODEL
	FileId      int64  `json:"fileId,string" gorm:"primarykey;comment:文件ID"`     // 文件ID(业务命名主键,DB自增)
	UserId      int64  `json:"userId,string" gorm:"index;comment:归属用户ID"`        // 归属用户ID(目录树按用户隔离)
	ParentId    int64  `json:"parentId,string" gorm:"index;comment:父目录ID(根=0)"`  // 父目录ID(根目录=0)
	Name        string `json:"name" gorm:"size:255;comment:文件/目录名"`              // 文件/目录名
	IsDirectory bool   `json:"isDirectory" gorm:"comment:是否目录"`                  // 是否目录
	Path        string `json:"path" gorm:"size:512;index;comment:全路径(含自身名,根为/)"` // 全路径(如 /docs/a.txt),根目录为 /
	Size        int64  `json:"size" gorm:"default:0;comment:字节数(目录=0)"`          // 字节数(目录=0)
	ExtendName  string `json:"extendName" gorm:"size:20;comment:扩展名(不含点)"`       // 扩展名(不含前导点)
	ContentType string `json:"contentType" gorm:"size:100;comment:MIME类型"`       // MIME类型
	// 以下为第3期上传体系预留:秒传指纹 / 存储落点 / 引用计数
	Md5         string `json:"md5" gorm:"size:32;index;comment:完整内容MD5(合并后算,权威指纹)"`
	QuickHash   string `json:"quickHash" gorm:"size:32;index;comment:采样MD5(首尾2MB+size),秒传初筛"`
	StrongHash  string `json:"strongHash" gorm:"size:64;comment:采样SHA256,防碰撞"`
	MidHash     string `json:"midHash" gorm:"size:32;comment:中间2MB MD5,秒传二次校验防首尾碰撞"`
	StorageType string `json:"storageType" gorm:"size:16;comment:存储类型 local/rustfs"`
	StoragePath string `json:"storagePath" gorm:"size:512;comment:对象key/本地相对路径"`
	RefCount    int    `json:"refCount" gorm:"default:1;comment:物理引用计数(跨用户秒传复用)"`
	Status      int    `json:"status" gorm:"default:1;comment:状态 1正常 2回收站 3删除中"`
	IsFavorite  bool   `json:"isFavorite" gorm:"default:false;comment:是否收藏"`
}

// TableName 表名(网盘文件表)
func (DiskFile) TableName() string {
	return "disk_files"
}

// DiskFile 状态枚举
const (
	DiskFileStatusNormal   = 1 // 正常
	DiskFileStatusTrashed  = 2 // 回收站
	DiskFileStatusDeleting = 3 // 删除中
)
