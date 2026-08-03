package disk

import (
	"archive/zip"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hllkk/devops-admin/server/model/common/response"
	diskReq "github.com/hllkk/devops-admin/server/model/disk/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
	"github.com/hllkk/devops-admin/server/utils/upload"
)

// DiskFileApi 网盘文件管理(对齐前端 /file-meta/* 资源)。
// 阶段0:只读(列表 + 路径解析);上传/秒传/合并等第3期补充。
type DiskFileApi struct{}

// GetFileList
// @Tags      DiskFile
// @Summary   分页获取当前目录文件列表
// @Description 网盘个人文件列表。userId 由后端从 JWT 取(防 IDOR);返回 {list,total,page,size}(网盘沿用 jmal 风格契约,非通用 rows 分页)。
// @Produce   application/json
// @Param     currentDirectory  query  string  false  "当前目录全路径(根='/')"
// @Param     queryType         query  string  false  "文件分类 all/image/document/video/audio/other"
// @Param     keyword           query  string  false  "文件名模糊"
// @Param     sortBy            query  string  false  "排序字段 name/size/time"
// @Param     sortOrder         query  string  false  "排序方式 asc/desc"
// @Param     pageNum           query  int     true   "页码"
// @Param     pageSize          query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=object,msg=string}  "data: disk.response.FileListResponse{list,total,page,size}"
// @Router    /file-meta/list [get]
func (d *DiskFileApi) GetFileList(c *gin.Context) {
	var q diskReq.FileListSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	resp, err := diskFileService.GetFileList(c.Request.Context(), utils.GetUserID(c), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取网盘文件列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(resp, "获取成功", c)
}

// Download
// @Tags      DiskFile
// @Summary   下载文件(后端代理流式 + Range)
// @Description 鉴权后从对象存储流式回写;userId 由 JWT 取防 IDOR;支持 Range 续传(http.ServeContent)。
//
//	生产 RustFS 桶仅 uploads/ 前缀公开,网盘文件存私有 file/ 前缀,经此接口鉴权代理下载(不走 /oss/ 公开反代)。
//
// @Produce   application/octet-stream
// @Param     fileId  query  int  true  "文件ID"
// @Router    /file-meta/download [get]
func (d *DiskFileApi) Download(c *gin.Context) {
	var q diskReq.DownloadReq
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage("无效的 fileId", c)
		return
	}
	f, err := diskFileService.GetFileForDownload(c.Request.Context(), utils.GetUserID(c), q.FileId)
	if err != nil {
		response.FailWithMessage("文件不存在或已删除", c)
		return
	}
	oss := upload.NewOss()
	mn, ok := oss.(*upload.Minio)
	if !ok {
		response.FailWithMessage("网盘需要 minio/rustfs 存储后端", c)
		return
	}
	obj, err := mn.GetObject(c.Request.Context(), f.StoragePath)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("网盘下载取对象失败")
		response.FailWithMessage("文件读取失败", c)
		return
	}
	defer obj.Close()
	// Stat 预检:对象不存在/损坏时在写 body 前返回错误(此时 header 未发,Fail 安全)
	if _, err := obj.Stat(); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("网盘下载对象 Stat 失败")
		response.FailWithMessage("文件存储异常", c)
		return
	}
	// 文件名 RFC5987 编码(中文/特殊字符安全,对标 jmal 范式,避 ISO-8859-1 乱码)
	c.Header("Content-Disposition", `attachment; filename*=UTF-8''`+url.PathEscape(f.Name))
	// 下载统一 octet-stream(attachment 语义):避免 .json 等文件被前端下载拦截器按
	// Content-Type: application/json 误判为错误响应;文件名靠 Content-Disposition filename* 传达。
	// (http.ServeContent 见 Content-Type 已设则不嗅探覆盖)
	c.Header("Content-Type", "application/octet-stream")
	// http.ServeContent 自动处理 Range 请求(206 + Content-Range)、Content-Length;
	// *minio.Object 实现 io.ReadSeeker,Seek 时按需向 rustfs 发 Range 请求,内存恒定。
	// modtime 传零值:不设 Last-Modified、跳过 If-Modified-Since 304 协商(私有文件无需缓存协商)。
	http.ServeContent(c.Writer, c.Request, f.Name, time.Time{}, obj)
}

// PackageDownload
// @Tags      DiskFile
// @Summary   打包下载(多文件/文件夹,流式 Zip)
// @Description 鉴权后递归收集选中项(含目录后代),流式 Zip 边读对象边压缩边写响应,不落临时盘。支持 OSS 对象打包(jmal 仅本地)。
// @Produce   application/zip
// @Param     fileIds  query  []string  true  "文件/文件夹ID列表(重复传 fileIds=N)"
// @Router    /file-meta/package-download [get]
func (d *DiskFileApi) PackageDownload(c *gin.Context) {
	// query array: ?fileIds=1&fileIds=2(原生 a 标签/window.open 同源带 httpOnly cookie 鉴权)
	rawIds := c.QueryArray("fileIds")
	fileIds := make([]int64, 0, len(rawIds))
	for _, s := range rawIds {
		if id, err := strconv.ParseInt(s, 10, 64); err == nil && id > 0 {
			fileIds = append(fileIds, id)
		}
	}
	if len(fileIds) == 0 {
		response.FailWithMessage("未选择有效文件", c)
		return
	}
	entries, err := diskFileService.CollectPackageFiles(c.Request.Context(), utils.GetUserID(c), fileIds)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	oss := upload.NewOss()
	mn, ok := oss.(*upload.Minio)
	if !ok {
		response.FailWithMessage("网盘需要 minio/rustfs 存储后端", c)
		return
	}
	// header 必须在写 body 前设;此后 GetObject/Zip 失败只能跳过该文件(已发 header 无法改返回错误)
	c.Header("Content-Disposition", `attachment; filename*=UTF-8''`+url.PathEscape("disk-download.zip"))
	c.Header("Content-Type", "application/zip")
	zw := zip.NewWriter(c.Writer)
	for _, e := range entries {
		obj, err := mn.GetObject(c.Request.Context(), e.StoragePath)
		if err != nil {
			logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("打包下载取对象失败: " + e.RelPath)
			continue
		}
		w, err := zw.Create(e.RelPath)
		if err != nil {
			_ = obj.Close()
			continue
		}
		if _, err := io.Copy(w, obj); err != nil {
			logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("打包下载写入失败: " + e.RelPath)
		}
		_ = obj.Close()
	}
	if err := zw.Close(); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("打包下载关闭 zip 失败")
	}
}

// ResolvePath
// @Tags      DiskFile
// @Summary   路径解析(面包屑/URL 深链恢复)
// @Description 按目录路径还原面包屑链;任一段不存在则报错(前端 catch 后回退根目录)。
// @Produce   application/json
// @Param     path  query  string  true  "目录全路径(根='/')"
// @Success   200  {object}  response.Response{data=object,msg=string}  "data: disk.response.PathResolveResponse{fileId,fileName,parentId,filePath,breadcrumb}"
// @Router    /file-meta/path-resolve [get]
func (d *DiskFileApi) ResolvePath(c *gin.Context) {
	var q diskReq.PathResolveReq
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	resp, err := diskFileService.ResolvePath(c.Request.Context(), utils.GetUserID(c), q.Path)
	if err != nil {
		response.FailWithMessage("路径不存在", c)
		return
	}
	response.OkWithDetailed(resp, "获取成功", c)
}

// GetFolderTree
// @Tags      DiskFile
// @Summary   获取目录树(移动/复制目标选择器)
// @Description 返回当前用户全部正常目录的树结构(仅目录)。
// @Produce   application/json
// @Success   200  {object}  response.Response{data=object,msg=string}  "data: []disk.response.FolderTreeNode"
// @Router    /file-meta/folder-tree [get]
func (d *DiskFileApi) GetFolderTree(c *gin.Context) {
	resp, err := diskFileService.GetFolderTree(c.Request.Context(), utils.GetUserID(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取目录树失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(resp, "获取成功", c)
}

// ---- 第2期 文件 CRUD ----

// Mkdir
// @Tags      DiskFile
// @Summary   新建文件夹
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  request.MkdirReq  true  "父目录路径 + 文件夹名"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /file-meta/mkdir [post]
func (d *DiskFileApi) Mkdir(c *gin.Context) {
	var req diskReq.MkdirReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := diskFileService.Mkdir(c.Request.Context(), utils.GetUserID(c), req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "创建成功", c)
}

// CreateFile
// @Tags      DiskFile
// @Summary   新建空文件(行内新建)
// @Description 校验后创建 0 字节占位对象落对象存储 + 写文件记录,返回新 fileId。
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  request.CreateFileReq  true  "父目录路径 + 文件名(含扩展名)"
// @Success   200  {object}  response.Response{data=object,msg=string}  "data: disk.response.CreateFileResp{fileId}"
// @Router    /file-meta/create-file [post]
func (d *DiskFileApi) CreateFile(c *gin.Context) {
	var req diskReq.CreateFileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	resp, err := diskFileService.CreateFile(c.Request.Context(), utils.GetUserID(c), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(resp, "创建成功", c)
}

// EnsureFolders
// @Tags      DiskFile
// @Summary   批量预建目录(文件夹上传前预建含空目录)
// @Description 按相对路径列表懒建子目录(已存在则复用,空目录也建)。文件夹上传前端先调此接口预建目录树,再传文件。
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  request.EnsureFoldersReq  true  "目标父目录 + 相对目录路径列表"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /file-meta/ensure-folders [post]
func (d *DiskFileApi) EnsureFolders(c *gin.Context) {
	var req diskReq.EnsureFoldersReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := diskFileService.EnsureFolders(c.Request.Context(), utils.GetUserID(c), req.CurrentDirectory, req.Paths); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "创建成功", c)
}

// Rename
// @Tags      DiskFile
// @Summary   重命名(文件夹含后代 path 级联)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  request.RenameReq  true  "文件ID + 新名称"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /file-meta/rename [post]
func (d *DiskFileApi) Rename(c *gin.Context) {
	var req diskReq.RenameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := diskFileService.Rename(c.Request.Context(), utils.GetUserID(c), req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "重命名成功", c)
}

// Move
// @Tags      DiskFile
// @Summary   移动(批量;文件夹含后代 path 级联;循环校验)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  request.MoveReq  true  "文件ID列表 + 目标目录全路径"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /file-meta/move [put]
func (d *DiskFileApi) Move(c *gin.Context) {
	var req diskReq.MoveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := diskFileService.Move(c.Request.Context(), utils.GetUserID(c), req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "移动成功", c)
}

// Copy
// @Tags      DiskFile
// @Summary   复制(批量;文件夹递归深拷贝子树)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  request.CopyReq  true  "文件ID列表 + 目标目录全路径"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /file-meta/copy [post]
func (d *DiskFileApi) Copy(c *gin.Context) {
	var req diskReq.CopyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := diskFileService.Copy(c.Request.Context(), utils.GetUserID(c), req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "复制成功", c)
}

// Delete
// @Tags      DiskFile
// @Summary   删除(移入回收站;文件夹级联后代)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  request.DeleteReq  true  "文件ID列表"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /file-meta/delete [post]
func (d *DiskFileApi) Delete(c *gin.Context) {
	var req diskReq.DeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := diskFileService.MoveToTrash(c.Request.Context(), utils.GetUserID(c), req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "已移入回收站", c)
}

// ---- 第3期 文件上传(分片/秒传/断点续传) ----

// CheckUpload
// @Tags      DiskFile
// @Summary   秒传/续传检测
// @Description 同用户已有同 quickHash+strongHash 文件→秒传(pass=true);否则返回会话与已收分片 resume[](断点续传)。
// @Produce   application/json
// @Param     identifier        query  string  true   "会话标识(quickHash)"
// @Param     quickHash         query  string  false  "采样MD5"
// @Param     strongHash        query  string  false  "采样SHA256"
// @Param     fileName          query  string  false  "文件名"
// @Param     totalSize         query  int     false  "总字节数"
// @Param     totalChunks       query  int     false  "分片总数"
// @Param     currentDirectory  query  string  false  "目标父目录全路径"
// @Success   200  {object}  response.Response{data=object,msg=string}  "data: disk.response.UploadCheckResp"
// @Router    /file-meta/upload [get]
func (d *DiskFileApi) CheckUpload(c *gin.Context) {
	var q diskReq.UploadCheckReq
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	resp, err := diskUploadService.Check(c.Request.Context(), utils.GetUserID(c), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("上传检测失败")
		response.FailWithMessage("检测失败", c)
		return
	}
	response.OkWithDetailed(resp, "获取成功", c)
}

// UploadChunk
// @Tags      DiskFile
// @Summary   上传分片(multipart,幂等)
// @Accept    multipart/form-data
// @Produce   application/json
// @Param     uploadId      formData  string  true  "会话ID"
// @Param     chunkNumber   formData  int     true  "分片序号(0-based)"
// @Param     chunkHash     formData  string  true  "分片MD5"
// @Param     file          formData  file    true  "分片文件"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /file-meta/upload [post]
func (d *DiskFileApi) UploadChunk(c *gin.Context) {
	uploadId, _ := strconv.ParseInt(c.PostForm("uploadId"), 10, 64)
	chunkNumber, _ := strconv.Atoi(c.PostForm("chunkNumber"))
	chunkHash := c.PostForm("chunkHash")
	if uploadId <= 0 {
		response.FailWithMessage("缺少 uploadId", c)
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.FailWithMessage("未接收到分片文件", c)
		return
	}
	src, err := fileHeader.Open()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	defer src.Close()
	// 流式收片:不再 io.ReadAll 整片入内存,由 service 边写边算 md5
	if err := diskUploadService.SaveChunkStream(c.Request.Context(), utils.GetUserID(c), uploadId, chunkNumber, chunkHash, src); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "上传成功", c)
}

// MergeUpload
// @Tags      DiskFile
// @Summary   合并分片(原子抢占→流式合并算MD5→推OSS→建文件→清理)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  request.UploadMergeReq  true  "合并请求"
// @Success   200  {object}  response.Response{data=object,msg=string}  "data: disk.response.UploadMergeResp"
// @Router    /file-meta/merge [post]
func (d *DiskFileApi) MergeUpload(c *gin.Context) {
	var req diskReq.UploadMergeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	resp, err := diskUploadService.Merge(c.Request.Context(), utils.GetUserID(c), req)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("合并失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(resp, "合并成功", c)
}

// CancelUpload
// @Tags      DiskFile
// @Summary   取消上传并清理分片/会话
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  request.UploadCancelReq  true  "取消请求"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /file-meta/upload [delete]
func (d *DiskFileApi) CancelUpload(c *gin.Context) {
	var req diskReq.UploadCancelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := diskUploadService.Cancel(c.Request.Context(), utils.GetUserID(c), req.Identifier); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "已取消", c)
}
