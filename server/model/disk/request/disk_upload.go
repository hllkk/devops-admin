package request

// UploadCheckReq 秒传/续传检测(GET /file-meta/upload,query 传输)。
// userId 由后端从 JWT 取;identifier=quickHash 作会话去重键。
type UploadCheckReq struct {
	Identifier       string `json:"identifier" form:"identifier"`             // 会话标识(quickHash)
	QuickHash        string `json:"quickHash" form:"quickHash"`               // 采样MD5(秒传初筛)
	StrongHash       string `json:"strongHash" form:"strongHash"`             // 采样SHA256(防碰撞)
	MidHash          string `json:"midHash" form:"midHash"`                   // 中间2MB MD5(秒传二次校验,防首尾碰撞)
	FileName         string `json:"fileName" form:"fileName"`                 // 文件名
	TotalSize        int64  `json:"totalSize" form:"totalSize"`               // 总字节数(数量,非ID,裸number传输)
	TotalChunks      int    `json:"totalChunks" form:"totalChunks"`           // 分片总数
	ChunkSize        int64  `json:"chunkSize" form:"chunkSize"`               // 分片大小(数量,非ID,裸number传输)
	RelativePath     string `json:"relativePath" form:"relativePath"`         // 文件夹上传相对路径
	CurrentDirectory string `json:"currentDirectory" form:"currentDirectory"` // 目标父目录全路径(根="/")
}

// UploadMergeReq 合并请求(POST /file-meta/merge)。
// Instant=true 时为秒传复用:不合并分片,直接按 quickHash+strongHash 查源文件,
// 在目标目录建新节点引用同一物理对象(源 ref_count++),用于同用户跨位置秒传落库(Check pass=true 时前端置 true)。
type UploadMergeReq struct {
	Identifier       string `json:"identifier" binding:"required"` // 会话标识(quickHash)
	FileName         string `json:"fileName" binding:"required"`   // 文件名
	TotalSize        int64  `json:"totalSize"`                     // 总字节数(数量,非ID,裸number传输;勿加,string)
	TotalChunks      int    `json:"totalChunks"`                   // 分片总数
	CurrentDirectory string `json:"currentDirectory"`              // 目标父目录全路径
	RelativePath     string `json:"relativePath"`                  // 文件夹相对路径
	QuickHash        string `json:"quickHash"`                     // 采样MD5
	StrongHash       string `json:"strongHash"`                    // 采样SHA256
	MidHash          string `json:"midHash"`                       // 中间2MB MD5(秒传二次校验,防首尾碰撞)
	Instant          bool   `json:"instant"`                       // 秒传复用模式(Check 命中 pass=true 时前端置 true)
}

// EnsureFoldersReq 批量预建目录(POST /file-meta/ensure-folders,文件夹上传前预建空目录)。
// Paths 为相对 currentDirectory 的目录相对路径列表(含空目录),逐段走与 mkdir 一致的校验+懒建。
type EnsureFoldersReq struct {
	CurrentDirectory string   `json:"currentDirectory" binding:"required"` // 目标父目录全路径(根="/")
	Paths            []string `json:"paths" binding:"required"`            // 相对目录路径列表(如 ["a/b","a/c"])
}

// UploadCancelReq 取消上传(DELETE /file-meta/upload)。
type UploadCancelReq struct {
	Identifier string `json:"identifier" binding:"required"` // 会话标识
}
