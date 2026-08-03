package response

// FileItem 网盘文件列表项(对齐前端 Api.Disk.BackendFileItem,JSON tag 一一对应)。
//
// 由 model/disk.DiskFile 转换而来:ID←FileId;FilePath←父目录 Path;时间格式化为 RFC3339 字符串。
// isShare/sharedUserCount/sharedDeptCount(第5期分享)/mediaCover/showCover(第4期预览)阶段0 恒为零值。
type FileItem struct {
	ID              string `json:"id"`              // 文件ID(对应 model.FileId,字符串对齐前端 IdType)
	Name            string `json:"name"`            // 文件/目录名
	ExtendName      string `json:"extendName"`      // 扩展名(保留原始值,可能含前导点号)
	IsDir           bool   `json:"isDir"`           // 是否文件夹
	Size            int64  `json:"size"`            // 字节数
	UpdateTime      string `json:"updateTime"`      // 更新时间(RFC3339)
	CreateTime      string `json:"createTime"`      // 创建时间(RFC3339)
	ContentType     string `json:"contentType"`     // MIME类型
	FilePath        string `json:"filePath"`        // 所在目录全路径(父目录 Path,根目录文件为 /)
	IsFavorite      bool   `json:"isFavorite"`      // 是否收藏
	IsShare         bool   `json:"isShare"`         // 是否已分享(外链,第5期)
	SharedUserCount int    `json:"sharedUserCount"` // 已共享用户数(第5期)
	SharedDeptCount int    `json:"sharedDeptCount"` // 已共享部门数(第5期)
	UserId          int64  `json:"userId"`          // 归属用户ID
	MediaCover      bool   `json:"mediaCover"`      // 是否有媒体封面/缩略图(第4期)
	ShowCover       bool   `json:"showCover"`       // 是否显示封面(第4期)
}

// FileListResponse 文件列表响应(对齐前端 Api.Disk.BackendFileListResponse)。
//
// 网盘列表沿用 jmal 风格 {list,total,page,size}(第1期前端契约已定型、vue-tsc 已过),
// 非项目通用分页 {rows,total,pageNum,pageSize};Swagger 据此标注实际形状。
type FileListResponse struct {
	List  []FileItem `json:"list"`  // 当页文件项
	Total int64      `json:"total"` // 总数
	Page  int        `json:"page"`  // 当前页码
	Size  int        `json:"size"`  // 每页大小
}

// BreadcrumbItem 面包屑项(对齐前端 Api.Disk.BreadcrumbItem)。
// FileID 用 interface{}:根目录为 nil(JSON null),文件夹为 int64(JSON number,非零)。
type BreadcrumbItem struct {
	FileID   interface{} `json:"fileId"`   // 文件夹ID(根目录为 null)
	FileName string      `json:"fileName"` // 文件夹名称
	FilePath string      `json:"filePath"` // 路径
}

// PathResolveResponse 路径解析响应(对齐前端 Api.Disk.PathResolveResponse)。
type PathResolveResponse struct {
	FileID     interface{}      `json:"fileId"`     // 最终文件夹ID(根/未命中为 null)
	FileName   string           `json:"fileName"`   // 文件夹名称
	ParentID   interface{}      `json:"parentId"`   // 父目录ID(根为 null)
	FilePath   string           `json:"filePath"`   // 路径字符串
	Breadcrumb []BreadcrumbItem `json:"breadcrumb"` // 面包屑链(首项为根目录 fileId=null)
}

// FolderTreeNode 目录树节点(移动/复制目标选择器用,GET /file-meta/folder-tree)。
type FolderTreeNode struct {
	ID       string           `json:"id"`       // 文件夹ID(字符串)
	Name     string           `json:"name"`     // 文件夹名
	Path     string           `json:"path"`     // 全路径
	Children []FolderTreeNode `json:"children"` // 子目录
}
