package request

import commonReq "github.com/hllkk/devops-admin/server/model/common/request"

// FileListSearch 网盘文件列表查询(对齐前端 GET /file-meta/list,query 传输)。
//
// currentDirectory 为当前目录全路径(根="/");queryType 为文件分类筛选(空或 all 不过滤);
// keyword 文件名模糊(复用 PageInfo.Keyword);sortBy/sortOrder 排序。
//
// 注:userId 不走请求参数,由后端从 JWT 取(utils.GetUserID),防 IDOR 越权列举他人目录。
type FileListSearch struct {
	commonReq.PageInfo
	CurrentDirectory string `json:"currentDirectory" form:"currentDirectory"` // 当前目录全路径(根="/")
	QueryType        string `json:"queryType" form:"queryType"`               // 文件分类 all/image/document/video/audio/other
	SortBy           string `json:"sortBy" form:"sortBy"`                     // 排序字段 name/size/time
	SortOrder        string `json:"sortOrder" form:"sortOrder"`               // 排序方式 asc/desc
}

// PathResolveReq 路径解析请求(对齐前端 GET /file-meta/path-resolve)。
type PathResolveReq struct {
	Path string `json:"path" form:"path"` // 待解析的全路径(根="/")
}

// ---- 第2期 文件 CRUD 请求体(对齐前端 /file-meta/* 写操作) ----
// 注:userId 一律由后端从 JWT 取,不进请求体(防 IDOR)。
// FileIds 用 []string 对齐前端 IdType(string|number),service 内 parseFileIds 转 []int64。

// MkdirReq 新建文件夹。ParentPath 为父目录全路径(根="/")。
type MkdirReq struct {
	ParentPath string `json:"parentPath" form:"parentPath"`                    // 父目录全路径(根="/")
	FolderName string `json:"folderName" form:"folderName" binding:"required"` // 新文件夹名
}

// CreateFileReq 新建空文件(行内新建)。ParentPath 为父目录全路径(根="/")。
// 后端创建 0 字节占位对象落对象存储 + 写 disk_files(IsDirectory=false, Size=0)。
type CreateFileReq struct {
	ParentPath string `json:"parentPath" form:"parentPath"`                // 父目录全路径(根="/")
	FileName   string `json:"fileName" form:"fileName" binding:"required"` // 新文件名(含扩展名)
}

// RenameReq 重命名。FileId 为目标文件 ID。
type RenameReq struct {
	FileId  int64  `json:"fileId,string" form:"fileId,string" binding:"required"` // 文件ID
	NewName string `json:"newName" form:"newName" binding:"required"`             // 新名称
}

// MoveReq 移动。FileIds 批量;TargetPath 为目标文件夹全路径。
type MoveReq struct {
	FileIds    []string `json:"fileIds" binding:"required,min=1"` // 待移动文件ID列表
	TargetPath string   `json:"targetPath" binding:"required"`    // 目标文件夹全路径
}

// CopyReq 复制。结构与 MoveReq 一致。
type CopyReq struct {
	FileIds    []string `json:"fileIds" binding:"required,min=1"` // 待复制文件ID列表
	TargetPath string   `json:"targetPath" binding:"required"`    // 目标文件夹全路径
}

// DeleteReq 删除(移入回收站)。FileIds 批量。
type DeleteReq struct {
	FileIds []string `json:"fileIds" binding:"required,min=1"` // 待删除文件ID列表
}

// DownloadReq 下载请求(对齐前端 GET /file-meta/download)。
// userId 由后端从 JWT 取,防 IDOR 越权下载他人文件。
type DownloadReq struct {
	FileId int64 `json:"fileId,string" form:"fileId,string" binding:"required"` // 待下载文件ID
}
