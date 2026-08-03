package disk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/disk"
	diskReq "github.com/hllkk/devops-admin/server/model/disk/request"
	diskRes "github.com/hllkk/devops-admin/server/model/disk/response"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/hllkk/devops-admin/server/utils/upload"
)

// DiskUploadService 网盘分片上传服务(对齐前端 /file-meta/upload|merge)。
// 镜像 media_upload.go 的 DB-backed 范式(会话+分片表+原子状态机),适配 disk_files:
//   - 秒传:同用户已有同 quickHash+strongHash 正常文件 → 直接复用(pass=true)。
//   - 续传:DB 会话 + 分片表,返回已收分片序号 resume[](多实例/重启可恢复)。
//   - 合并:原子抢占 uploading→merging → 流式合并算完整 MD5 → OSS 推送 → 建 disk_files → 清理。
//   - 物理分片存 utils/upload.DiskChunkDir(sessionID),与 media 隔离。
type DiskUploadService struct{}

// 上传安全上限(任一项为 0 表示该项不限)。
// 单文件上限由 sys_disk_config 配置(运维后台"网盘配置"页可调,见 diskUploadLimit);分片上限为技术约束。
const (
	maxChunkSize   int64 = 64 << 20 // 单分片最大 64MB(需 ≥ 前端 getChunkSize 最大档 50MB + multipart 边界余量)
	maxTotalChunks       = 10000    // 最大分片数
)

// diskUploadLimit 从 sys_disk_config 读取单文件上传上限(字节,1024 口径对齐前端显示),0=不限。
// 配置由后台网盘配置页维护;Current 走内存缓存热读(启动 LoadAll 已加载)。
func diskUploadLimit(ctx context.Context) int64 {
	cfg := (&system.DiskConfigService{}).Current(ctx)
	if cfg.MaxUploadSize <= 0 {
		return 0
	}
	var unit int64
	switch strings.ToUpper(cfg.MaxUploadSizeUnit) {
	case "MB":
		unit = 1 << 20
	case "TB":
		unit = 1 << 40
	default: // GB 或未知单位兜底
		unit = 1 << 30
	}
	return int64(cfg.MaxUploadSize * float64(unit))
}

// Check 秒传 + 续传检测(对齐 GET /file-meta/upload)。
func (s *DiskUploadService) Check(ctx context.Context, userId int64, req diskReq.UploadCheckReq) (diskRes.UploadCheckResp, error) {
	var resp diskRes.UploadCheckResp
	resp.Resume = []int{}

	// 0) 安全上限校验(防分片洪泛/超大文件,建会话前拦截)
	if maxTotalChunks > 0 && req.TotalChunks > maxTotalChunks {
		return resp, fmt.Errorf("分片数超过上限 %d", maxTotalChunks)
	}
	if limit := diskUploadLimit(ctx); limit > 0 && req.TotalSize > limit {
		return resp, errors.New("文件大小超过上限")
	}
	if maxChunkSize > 0 && req.ChunkSize > maxChunkSize {
		return resp, errors.New("分片大小超过上限")
	}

	// 0.1) 配额预检(H3):非超管且配额上限内放不下 totalSize → 早拒省上传带宽(Merge 仍兜底对账)
	if !isSuperAdmin(ctx, userId) {
		if quota := diskQuotaLimitBytes(ctx); quota > 0 {
			var used float64
			global.OPS_DB.WithContext(ctx).Table("sys_users").
				Where("id = ?", userId).Select("take_up_space").Scan(&used)
			if int64(used)+req.TotalSize > quota {
				return resp, errors.New("存储空间不足")
			}
		}
	}

	// 1) 秒传:同用户已有同 quickHash+strongHash 的正常文件
	if req.QuickHash != "" {
		var existing disk.DiskFile
		e := global.OPS_DB.WithContext(ctx).
			Where("user_id = ? AND quick_hash = ? AND strong_hash = ? AND status = ?",
				userId, req.QuickHash, req.StrongHash, disk.DiskFileStatusNormal).
			Take(&existing).Error
		if e == nil && existing.FileId != 0 {
			// 中间块二次校验(防首尾采样碰撞致内容张冠李戴):
			// 源有 mid_hash 且前端有 mid_hash(>4MB 文件)时必须相符,不符降级为未命中走正常上传;
			// 源无 mid_hash(历史文件)或前端无 mid_hash(<=4MB 首尾已全覆盖)时跳过,保持兼容。
			midOK := existing.MidHash == "" || req.MidHash == "" || existing.MidHash == req.MidHash
			if midOK {
				resp.Pass = true
				resp.FileId = strconv.FormatInt(existing.FileId, 10)
				resp.SourceFileId = resp.FileId
				return resp, nil
			}
			// midOK=false:中间块不符,非真同文件,不秒传,继续走会话/上传
		}
	}

	// 2) 取/建进行中会话(identifier 缺省用 quickHash)
	identifier := req.Identifier
	if identifier == "" {
		identifier = req.QuickHash
	}
	var sess disk.DiskUploadSession
	e := global.OPS_DB.WithContext(ctx).
		Where("user_id = ? AND identifier = ? AND status = ?", userId, identifier, disk.UploadStatusUploading).
		Take(&sess).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		sess = disk.DiskUploadSession{
			UserId:           userId,
			Identifier:       identifier,
			FileName:         req.FileName,
			RelativePath:     req.RelativePath,
			TotalSize:        req.TotalSize,
			ChunkSize:        req.ChunkSize,
			TotalChunks:      req.TotalChunks,
			CurrentDirectory: req.CurrentDirectory,
			Status:           disk.UploadStatusUploading,
			QuickHash:        req.QuickHash,
			StrongHash:       req.StrongHash,
		}
		if e = global.OPS_DB.WithContext(ctx).Create(&sess).Error; e != nil {
			return resp, e
		}
	} else if e != nil {
		return resp, e
	}

	// 3) 已收分片序号
	var idx []int
	global.OPS_DB.WithContext(ctx).Model(&disk.DiskUploadChunk{}).Where("upload_id = ?", sess.UploadId).
		Order("chunk_number").Pluck("chunk_number", &idx)
	if len(idx) == 0 {
		idx = []int{}
	}
	resp.UploadId = strconv.FormatInt(sess.UploadId, 10)
	resp.Resume = idx
	resp.Merge = sess.TotalChunks > 0 && len(idx) == sess.TotalChunks
	return resp, nil
}

// SaveChunkStream 流式收一个分片(multipart reader)。chunkNumber 0-based;chunkHash=分片 MD5;
// 幂等(ON CONFLICT DO NOTHING)。分片边界校验 + 流式落盘(不整片入内存)。
func (s *DiskUploadService) SaveChunkStream(ctx context.Context, userId, uploadId int64, chunkNumber int, chunkHash string, reader io.Reader) error {
	var sess disk.DiskUploadSession
	if e := global.OPS_DB.WithContext(ctx).First(&sess, uploadId).Error; e != nil {
		return errors.New("上传会话不存在")
	}
	if sess.UserId != userId {
		return errors.New("无权操作该上传")
	}
	if sess.Status != disk.UploadStatusUploading {
		return errors.New("上传会话状态不允许收片")
	}
	// 分片序号边界
	if chunkNumber < 0 || (sess.TotalChunks > 0 && chunkNumber >= sess.TotalChunks) {
		return fmt.Errorf("非法分片序号 %d", chunkNumber)
	}
	// 流式落盘 + 边写边算 md5(写到 .tmp)
	size, gotMd5, err := upload.SaveDiskChunkStream(uploadId, chunkNumber, reader)
	if err != nil {
		return err
	}
	// 单片大小上限(末片可更小但不得更大)
	if maxChunkSize > 0 && size > maxChunkSize {
		_ = upload.CommitDiskChunk(uploadId, chunkNumber, false)
		return fmt.Errorf("分片 %d 超过最大分片大小", chunkNumber)
	}
	if sess.ChunkSize > 0 && size > sess.ChunkSize {
		_ = upload.CommitDiskChunk(uploadId, chunkNumber, false)
		return fmt.Errorf("分片 %d 超过声明分片大小", chunkNumber)
	}
	// 分片 MD5 校验
	if gotMd5 != chunkHash {
		_ = upload.CommitDiskChunk(uploadId, chunkNumber, false)
		return fmt.Errorf("分片 %d 校验失败", chunkNumber)
	}
	// 校验通过:提交 .tmp → .part
	if err := upload.CommitDiskChunk(uploadId, chunkNumber, true); err != nil {
		return err
	}
	rec := disk.DiskUploadChunk{UploadId: uploadId, ChunkNumber: chunkNumber, ChunkHash: chunkHash, Size: size}
	return global.OPS_DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "upload_id"}, {Name: "chunk_number"}},
		DoNothing: true,
	}).Create(&rec).Error
}

// Merge 合并分片 → 算完整 MD5 → 推 OSS → 建 disk_files → 清理(对齐 POST /file-meta/merge)。
func (s *DiskUploadService) Merge(ctx context.Context, userId int64, req diskReq.UploadMergeReq) (diskRes.UploadMergeResp, error) {
	// 秒传复用分支(H1):Check 命中 pass=true 后前端置 instant=true,不合并分片直接落库建引用节点
	if req.Instant {
		return s.mergeInstant(ctx, userId, req)
	}

	var resp diskRes.UploadMergeResp

	// 找会话(按 identifier + user,取最新一条)
	var sess disk.DiskUploadSession
	e := global.OPS_DB.WithContext(ctx).
		Where("user_id = ? AND identifier = ?", userId, req.Identifier).
		Order("upload_id DESC").Take(&sess).Error
	if e != nil {
		return resp, errors.New("上传会话不存在")
	}

	// 原子抢占 uploading→merging(防并发双合并)
	res := global.OPS_DB.WithContext(ctx).Model(&disk.DiskUploadSession{}).
		Where("upload_id = ? AND status = ?", sess.UploadId, disk.UploadStatusUploading).
		Update("status", disk.UploadStatusMerging)
	if res.Error != nil {
		return resp, res.Error
	}
	if res.RowsAffected == 0 {
		return resp, errors.New("上传不在可合并状态(可能已在合并或已完成)")
	}
	fail := func(err error) (diskRes.UploadMergeResp, error) {
		global.OPS_DB.WithContext(ctx).Model(&disk.DiskUploadSession{}).Where("upload_id = ?", sess.UploadId).
			Update("status", disk.UploadStatusFailed)
		return resp, err
	}

	// 校验分片齐全
	var cnt int64
	global.OPS_DB.WithContext(ctx).Model(&disk.DiskUploadChunk{}).Where("upload_id = ?", sess.UploadId).Count(&cnt)
	if int(cnt) != sess.TotalChunks {
		return fail(fmt.Errorf("分片不全: %d/%d", cnt, sess.TotalChunks))
	}

	// 流式合并 + 算完整 MD5(hash-while-merge)
	merged := filepath.Join(upload.DiskChunkDir(sess.UploadId), "merged.bin")
	gotMd5, err := upload.MergeDiskChunks(sess.UploadId, sess.TotalChunks, merged)
	if err != nil {
		return fail(err)
	}
	// 合并完整性校验(M1):实测 size 与声明 totalSize 一致,防分片静默缺失/损坏(分片级 chunkHash 已挡内容篡改)
	mergedSize := int64(-1)
	if stat, se := os.Stat(merged); se == nil {
		mergedSize = stat.Size()
	}
	if mergedSize != sess.TotalSize {
		return fail(fmt.Errorf("合并大小不一致: 实测 %d / 声明 %d", mergedSize, sess.TotalSize))
	}

	// 经 OSS 接口推送(local/rustfs 由 System.OssType 决定)
	fh, cleanup, err := upload.BuildFileHeader(merged, "file", sess.FileName)
	if err != nil {
		return fail(err)
	}
	defer cleanup()
	oss := upload.NewOss()
	urlStr, key, err := oss.UploadFile(ctx, fh)
	if err != nil {
		return fail(err)
	}

	// 解析目标父目录(currentDirectory + relativePath 懒建子目录,支持文件夹上传)。
	// relativePath 优先取 merge 请求体:前端 check 阶段未透传时 session 内为空,
	// 仅靠 session 会让文件夹上传退化为平铺到 currentDirectory;回退 session 兜底。
	relPath := req.RelativePath
	if relPath == "" {
		relPath = sess.RelativePath
	}
	parent, perr := s.resolveFolderForUpload(ctx, userId, sess.CurrentDirectory, relPath)
	if perr != nil {
		return fail(perr)
	}

	name := sess.FileName
	ext := ""
	if i := strings.LastIndex(name, "."); i >= 0 {
		ext = strings.TrimPrefix(name[i:], ".")
	}
	// 同名去重(H4):与 Mkdir/CreateFile 一致,同级同名直接拒绝(非自动改名),保持简洁一致
	var fileSvc DiskFileService
	dup, derr := fileSvc.sameNameExists(ctx, userId, parent.FileId, name, 0)
	if derr != nil {
		return fail(derr)
	}
	if dup {
		return fail(errors.New("当前目录下已存在同名项: " + name))
	}
	// 配额对账(H3):建节点前原子预占 take_up_space;不足回滚已上传 OSS 对象
	if qerr := reserveUserSpace(ctx, userId, sess.TotalSize); qerr != nil {
		_ = oss.DeleteFile(ctx, key)
		return fail(qerr)
	}
	file := disk.DiskFile{
		UserId:      userId,
		ParentId:    parent.FileId,
		Name:        name,
		IsDirectory: false,
		Path:        joinPath(parent.Path, name),
		Size:        sess.TotalSize,
		ExtendName:  ext,
		Md5:         gotMd5,
		QuickHash:   sess.QuickHash,
		StrongHash:  sess.StrongHash,
		MidHash:     req.MidHash,
		StorageType: global.OPS_CONFIG.System.OssType,
		StoragePath: key,
		RefCount:    1,
		Status:      disk.DiskFileStatusNormal,
	}
	file.CreateBy = userId
	file.UpdateBy = userId
	if err = global.OPS_DB.WithContext(ctx).Create(&file).Error; err != nil {
		releaseUserSpace(ctx, userId, sess.TotalSize) // 建节点失败:退还预占配额
		return fail(err)
	}

	// 回填会话 + 清理分片
	global.OPS_DB.WithContext(ctx).Model(&disk.DiskUploadSession{}).Where("upload_id = ?", sess.UploadId).
		Updates(map[string]interface{}{
			"status":      disk.UploadStatusCompleted,
			"file_id":     file.FileId,
			"merged_md5":  gotMd5,
			"storage_key": key,
		})
	global.OPS_DB.WithContext(ctx).Where("upload_id = ?", sess.UploadId).Delete(&disk.DiskUploadChunk{})
	_ = upload.RemoveDiskUploadDir(sess.UploadId)

	resp.FileId = strconv.FormatInt(file.FileId, 10)
	resp.Url = urlStr
	return resp, nil
}

// Cancel 取消并清理分片/会话(对齐 DELETE /file-meta/upload)。
func (s *DiskUploadService) Cancel(ctx context.Context, userId int64, identifier string) error {
	var sess disk.DiskUploadSession
	e := global.OPS_DB.WithContext(ctx).
		Where("user_id = ? AND identifier = ?", userId, identifier).
		Order("upload_id DESC").Take(&sess).Error
	if e != nil {
		return nil // 会话不存在视为已取消
	}
	global.OPS_DB.WithContext(ctx).Where("upload_id = ?", sess.UploadId).Delete(&disk.DiskUploadChunk{})
	_ = upload.RemoveDiskUploadDir(sess.UploadId)
	return global.OPS_DB.WithContext(ctx).Where("upload_id = ?", sess.UploadId).Delete(&disk.DiskUploadSession{}).Error
}

// resolveFolderForUpload 解析上传目标父目录(文件夹上传懒建子目录)。复用 DiskFileService.EnsureFolderByRelPath
// (与 mkdir/ensure-folders 共用校验+懒建逻辑,避免重复实现漂移)。返回最终父目录。
//
// relativePath 是文件相对路径(前端 webkitRelativePath,含文件名末段,如 "123/456/789.txt")。
// 只懒建其目录段,文件名末段交给 Merge 建文件节点;否则 EnsureFolderByRelPath 会把文件名也建成
// 目录节点,出现"文件被存成文件夹、文件夹下又套同名文件"。
func (s *DiskUploadService) resolveFolderForUpload(ctx context.Context, userId int64, currentDirectory, relativePath string) (disk.DiskFile, error) {
	var fileSvc DiskFileService
	return fileSvc.EnsureFolderByRelPath(ctx, userId, currentDirectory, relPathDirPart(relativePath))
}

// relPathDirPart 取文件相对路径(webkitRelativePath,含文件名末段)的目录段:
// 去末段文件名,返回纯目录相对路径("123/456/789.txt"→"123/456","789.txt"→"")。
// 供上传 merge 懒建中间目录用;文件名段由 Merge 建文件节点,不得作为目录建。
func relPathDirPart(rel string) string {
	if rel == "" {
		return ""
	}
	if idx := strings.LastIndex(rel, "/"); idx >= 0 {
		return rel[:idx]
	}
	return ""
}

// mergeInstant 秒传复用落库(POST /file-meta/merge with instant=true,H1)。
// Check 命中 pass=true 后前端置 instant=true:不合并分片,按 quickHash+strongHash 查源文件,
// 在目标目录建新 disk_files 节点引用同一物理对象,源 ref_count++。
// 同位置同名同内容(源就在此) → 幂等返回;同名不同内容 → 报错(与 H4 一致)。
func (s *DiskUploadService) mergeInstant(ctx context.Context, userId int64, req diskReq.UploadMergeReq) (diskRes.UploadMergeResp, error) {
	var resp diskRes.UploadMergeResp
	if req.QuickHash == "" {
		return resp, errors.New("秒传缺少指纹")
	}
	// 查源文件(同用户+同指纹+正常)
	var src disk.DiskFile
	e := global.OPS_DB.WithContext(ctx).
		Where("user_id = ? AND quick_hash = ? AND strong_hash = ? AND status = ?",
			userId, req.QuickHash, req.StrongHash, disk.DiskFileStatusNormal).
		Take(&src).Error
	if e != nil {
		return resp, errors.New("秒传源文件不存在")
	}
	// 解析目标父目录(懒建子目录,文件夹秒传)
	parent, perr := s.resolveFolderForUpload(ctx, userId, req.CurrentDirectory, req.RelativePath)
	if perr != nil {
		return resp, perr
	}
	// 同名去重(H4):同位置同名 → 若同内容(源在此)幂等返回,否则报同名冲突
	var fileSvc DiskFileService
	dup, derr := fileSvc.sameNameExists(ctx, userId, parent.FileId, req.FileName, 0)
	if derr != nil {
		return resp, derr
	}
	if dup {
		var exist disk.DiskFile
		ge := global.OPS_DB.WithContext(ctx).
			Where("user_id = ? AND parent_id = ? AND name = ? AND status = ?", userId, parent.FileId, req.FileName, disk.DiskFileStatusNormal).
			Take(&exist).Error
		if ge == nil && exist.QuickHash == src.QuickHash && exist.StrongHash == src.StrongHash {
			resp.FileId = strconv.FormatInt(exist.FileId, 10)
			return resp, nil
		}
		return resp, errors.New("当前目录下已存在同名项: " + req.FileName)
	}
	// 配额对账(H3):秒传复用物理对象仍记一份逻辑占用(用户视角=文件数,与 remote 同款)
	if qerr := reserveUserSpace(ctx, userId, req.TotalSize); qerr != nil {
		return resp, qerr
	}
	ext := ""
	if i := strings.LastIndex(req.FileName, "."); i >= 0 {
		ext = strings.TrimPrefix(req.FileName[i:], ".")
	}
	file := disk.DiskFile{
		UserId:      userId,
		ParentId:    parent.FileId,
		Name:        req.FileName,
		IsDirectory: false,
		Path:        joinPath(parent.Path, req.FileName),
		Size:        req.TotalSize,
		ExtendName:  ext,
		Md5:         src.Md5,
		QuickHash:   src.QuickHash,
		StrongHash:  src.StrongHash,
		MidHash:     src.MidHash,
		StorageType: src.StorageType,
		StoragePath: src.StoragePath,
		RefCount:    1,
		Status:      disk.DiskFileStatusNormal,
	}
	file.CreateBy = userId
	file.UpdateBy = userId
	// 事务:建节点 + 源 ref_count++ 原子(失败整体回滚 + 退还预占配额,防账不平)
	if err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(&file).Error; e != nil {
			return e
		}
		if e := tx.Model(&disk.DiskFile{}).Where("file_id = ?", src.FileId).
			UpdateColumn("ref_count", gorm.Expr("ref_count + 1")).Error; e != nil {
			return e
		}
		return nil
	}); err != nil {
		releaseUserSpace(ctx, userId, req.TotalSize) // 回滚后退还预占配额
		return resp, err
	}
	resp.FileId = strconv.FormatInt(file.FileId, 10)
	return resp, nil
}

// CleanupStaleChunks 清理过期上传会话的分片(定时任务调用,M3)。
// 扫描 disk_upload_sessions 中 status=uploading/merging 且 update_time 早于 ttl 的会话:
// 删分片目录 + 删 chunks 记录 + 标 session failed(留痕排查,不物理删 session)。
func (s *DiskUploadService) CleanupStaleChunks(ctx context.Context, ttlHours int) error {
	if ttlHours <= 0 {
		ttlHours = 24
	}
	cutoff := time.Now().Add(-time.Duration(ttlHours) * time.Hour)
	var sessions []disk.DiskUploadSession
	if err := global.OPS_DB.WithContext(ctx).
		Where("status IN ? AND update_time < ?", []string{disk.UploadStatusUploading, disk.UploadStatusMerging}, cutoff).
		Find(&sessions).Error; err != nil {
		return err
	}
	for i := range sessions {
		sess := sessions[i]
		global.OPS_DB.WithContext(ctx).Where("upload_id = ?", sess.UploadId).Delete(&disk.DiskUploadChunk{})
		_ = upload.RemoveDiskUploadDir(sess.UploadId)
		global.OPS_DB.WithContext(ctx).Model(&disk.DiskUploadSession{}).Where("upload_id = ?", sess.UploadId).
			Update("status", disk.UploadStatusFailed)
	}
	return nil
}
