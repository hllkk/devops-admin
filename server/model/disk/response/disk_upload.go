package response

// UploadCheckResp 秒传/续传检测结果(GET /file-meta/upload)。
// pass=true 时前端须调 POST /file-meta/merge(instant=true) 落库:同位置同名幂等返回,
// 不同位置则建新 disk_files 节点引用源物理对象 + 源 ref_count++(秒传不传任何分片)。
type UploadCheckResp struct {
	Pass         bool   `json:"pass"`         // true=秒传命中(同用户已有同 quickHash+strongHash 文件)
	FileId       string `json:"fileId"`       // 秒传源文件ID(pass=true 时有值,前端 merge instant 透传)
	SourceFileId string `json:"sourceFileId"` // 秒传源文件ID(pass=true 时有值,后端 merge 按 quickHash+strongHash 查源,此字段冗余校验用)
	UploadId     string `json:"uploadId"`     // 上传会话ID(继续上传/合并用)
	Resume       []int  `json:"resume"`       // 已传分片序号(0-based,断点续传;nil 表示无)
	Merge        bool   `json:"merge"`        // 全部分片已到齐可直接合并
}

// UploadMergeResp 合并结果(POST /file-meta/merge)。
type UploadMergeResp struct {
	FileId string `json:"fileId"` // 生成的 disk_files.file_id
	Url    string `json:"url"`     // 访问URL(预览/下载用,storage_type=local 时为本地路径)
}

// CreateFileResp 新建空文件结果(POST /file-meta/create-file)。返回新 fileId 便于前端定位。
type CreateFileResp struct {
	FileId string `json:"fileId"` // 生成的 disk_files.file_id
}
