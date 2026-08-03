package disk

import "github.com/gin-gonic/gin"

// DiskFileRouter 网盘文件路由(对齐前端 /file-meta/* 资源)。
type DiskFileRouter struct{}

// InitDiskFileRouter 网盘文件相关路由。
//
// 网盘 /file-meta/* 为个人自有数据操作(每个登录用户管理自己的文件,service 按 JWT userId 鉴权隔离),
// 不适用 PrivateGroup 的角色级 Casbin 门控与部门 DataScope;故由 initialize/router.go 建专用
// diskGroup(仅挂 JWTAuth + OperationRecord)挂载本路由。
func (r *DiskFileRouter) InitDiskFileRouter(Router *gin.RouterGroup) {
	fileMeta := Router.Group("file-meta")
	{
		fileMeta.GET("list", diskFileApi.GetFileList)                 // 文件列表(当前目录子节点)
		fileMeta.GET("path-resolve", diskFileApi.ResolvePath)         // 路径解析(面包屑/深链恢复)
		fileMeta.GET("folder-tree", diskFileApi.GetFolderTree)        // 目录树(移动/复制目标选择器)
		fileMeta.GET("download", diskFileApi.Download)                // 下载文件(后端代理流式+Range,私有鉴权)
		fileMeta.GET("package-download", diskFileApi.PackageDownload) // 打包下载(多文件/文件夹,流式Zip)
		fileMeta.POST("mkdir", diskFileApi.Mkdir)                     // 新建文件夹
		fileMeta.POST("create-file", diskFileApi.CreateFile)          // 新建空文件(行内新建)
		fileMeta.POST("ensure-folders", diskFileApi.EnsureFolders)    // 批量预建目录(文件夹上传前预建含空目录)
		fileMeta.POST("rename", diskFileApi.Rename)                   // 重命名(含后代 path 级联)
		fileMeta.PUT("move", diskFileApi.Move)                        // 移动(批量+循环校验+级联)
		fileMeta.POST("copy", diskFileApi.Copy)                       // 复制(文件夹递归深拷贝)
		fileMeta.POST("delete", diskFileApi.Delete)                   // 删除→回收站(级联后代)
		fileMeta.GET("upload", diskFileApi.CheckUpload)               // 秒传/续传检测
		fileMeta.POST("upload", diskFileApi.UploadChunk)              // 上传分片(multipart)
		fileMeta.POST("merge", diskFileApi.MergeUpload)               // 合并分片
		fileMeta.DELETE("upload", diskFileApi.CancelUpload)           // 取消上传
	}
}
