package disk

import (
	"context"
	"errors"
	"mime"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/disk"
	diskReq "github.com/hllkk/devops-admin/server/model/disk/request"
	diskRes "github.com/hllkk/devops-admin/server/model/disk/response"
	"github.com/hllkk/devops-admin/server/utils/upload"
)

// DiskFileService 网盘文件服务(对齐前端 /file-meta/* 资源)。
// 阶段0 仅实现只读:列表(按用户隔离 + 目录树直接子节点)与路径解析(面包屑)。
// 上传/秒传/合并/续传/配额等第3期能力后续补充。
type DiskFileService struct{}

// diskListOrder 列表统一排序:文件夹优先(is_directory DESC,true 在前),再按指定列。
// col/dir 经白名单映射,杜绝 SQL 注入。
func diskListOrder(sortBy, sortOrder string) string {
	dir := "ASC"
	if strings.EqualFold(sortOrder, "DESC") {
		dir = "DESC"
	}
	col := "name"
	switch sortBy {
	case "name":
		col = "name"
	case "size":
		col = "size"
	case "time":
		col = "update_time"
	}
	return "is_directory DESC, " + col + " " + dir
}

// applyQueryType 按文件分类筛选(对齐前端 contentTypeToFileType)。
// 启用分类筛选时排除文件夹(与前端 mock 行为一致:只看该类型的文件)。
func applyQueryType(db *gorm.DB, queryType string) *gorm.DB {
	switch queryType {
	case "image":
		return db.Where("is_directory = ? AND content_type LIKE ?", false, "image/%")
	case "video":
		return db.Where("is_directory = ? AND content_type LIKE ?", false, "video/%")
	case "audio":
		return db.Where("is_directory = ? AND content_type LIKE ?", false, "audio/%")
	case "document":
		return db.Where("is_directory = ? AND (content_type LIKE ? OR content_type LIKE ? OR content_type LIKE ? OR content_type LIKE ? OR content_type LIKE ? OR content_type LIKE ? OR content_type = ?)",
			false, "%pdf%", "%text%", "%word%", "%excel%", "%spreadsheet%", "%presentation%", "application/json")
	case "other":
		return db.Where("is_directory = ?", false)
	default: // all / "" → 不过滤
		return db
	}
}

// resolveParentId 由当前目录全路径解析其 fileId(目录树父节点)。
// 根目录("/")→0;命中目录→其 FileId;目录不存在→found=false(调用方按空列表返回)。
// 用独立 tx 查询(Take 是 finisher),避免污染主列表 db(见 backend-layer-rules「GORM 链式查询」)。
func (s *DiskFileService) resolveParentId(ctx context.Context, userId int64, currentDirectory string) (parentId int64, found bool, err error) {
	if currentDirectory == "" || currentDirectory == "/" {
		return 0, true, nil
	}
	var folder disk.DiskFile
	e := global.OPS_DB.WithContext(ctx).
		Where("user_id = ? AND path = ? AND is_directory = ?", userId, currentDirectory, true).
		Take(&folder).Error
	if e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return 0, false, nil
		}
		return 0, false, e
	}
	return folder.FileId, true, nil
}

// GetFileList 分页查当前目录的直接子节点(对齐前端 GET /file-meta/list)。
// userId 强制取 JWT(防 IDOR);按 parent_id 列子节点;文件夹优先 + 排序;queryType/keyword 过滤。
func (s *DiskFileService) GetFileList(ctx context.Context, userId int64, q diskReq.FileListSearch) (diskRes.FileListResponse, error) {
	empty := diskRes.FileListResponse{List: []diskRes.FileItem{}, Total: 0, Page: q.PageNum, Size: q.PageSize}

	parentId, found, err := s.resolveParentId(ctx, userId, q.CurrentDirectory)
	if err != nil {
		return empty, err
	}
	if !found {
		return empty, nil
	}

	db := global.OPS_DB.WithContext(ctx).Model(&disk.DiskFile{}).
		Where("user_id = ?", userId).
		Where("parent_id = ?", parentId).
		Where("status = ?", disk.DiskFileStatusNormal)

	db = applyQueryType(db, q.QueryType)
	if q.Keyword != "" {
		db = db.Where("name LIKE ?", "%"+q.Keyword+"%")
	}

	var total int64
	var files []disk.DiskFile
	limit, offset := q.LimitOffset()
	order := diskListOrder(q.SortBy, q.SortOrder)
	if limit > 0 {
		err = db.Count(&total).Order(order).Limit(limit).Offset(offset).Find(&files).Error
	} else {
		err = db.Count(&total).Order(order).Find(&files).Error
	}
	if err != nil {
		return empty, err
	}

	// 当前目录即所有子项的 filePath(父目录路径)
	filePath := q.CurrentDirectory
	if filePath == "" {
		filePath = "/"
	}

	list := make([]diskRes.FileItem, 0, len(files))
	for _, f := range files {
		list = append(list, toFileItem(f, filePath))
	}
	return diskRes.FileListResponse{List: list, Total: total, Page: q.PageNum, Size: q.PageSize}, nil
}

// GetFileForDownload 下载预检:按 fileId + JWT userId 取文件(防 IDOR),校验正常状态 + 非目录 + 有存储路径。
// 仅做元数据查询(Select 限定列),物理读取由 handler 经 OSS 抽象 GetObject 流式输出。
func (s *DiskFileService) GetFileForDownload(ctx context.Context, userId, fileId int64) (disk.DiskFile, error) {
	var f disk.DiskFile
	err := global.OPS_DB.WithContext(ctx).
		Select("file_id", "user_id", "name", "is_directory", "content_type", "storage_path").
		Where("file_id = ? AND user_id = ?", fileId, userId).
		Where("status = ?", disk.DiskFileStatusNormal).
		Take(&f).Error
	if err != nil {
		return f, errors.New("文件不存在或已删除")
	}
	if f.IsDirectory {
		return f, errors.New("暂不支持下载目录") // 目录打包下载属后续增强
	}
	if f.StoragePath == "" {
		return f, errors.New("文件存储路径为空")
	}
	return f, nil
}

// toFileItem 将 DiskFile 模型转为前端 FileItem DTO。
func toFileItem(f disk.DiskFile, filePath string) diskRes.FileItem {
	return diskRes.FileItem{
		ID:          strconv.FormatInt(f.FileId, 10),
		Name:        f.Name,
		ExtendName:  f.ExtendName,
		IsDir:       f.IsDirectory,
		Size:        f.Size,
		UpdateTime:  f.UpdatedAt.Format(time.RFC3339),
		CreateTime:  f.CreatedAt.Format(time.RFC3339),
		ContentType: f.ContentType,
		FilePath:    filePath,
		IsFavorite:  f.IsFavorite,
		UserId:      f.UserId,
		// isShare/sharedUserCount/sharedDeptCount/mediaCover/showCover 第4/5期实现,阶段0 恒零值
	}
}

// ResolvePath 路径解析:按路径段还原面包屑(对齐前端 GET /file-meta/path-resolve)。
// 仅解析目录路径;任一段不存在→报错(前端 restoreFromPath 会 catch 回退根目录)。
func (s *DiskFileService) ResolvePath(ctx context.Context, userId int64, p string) (diskRes.PathResolveResponse, error) {
	resp := diskRes.PathResolveResponse{
		FileID:     nil,
		FileName:   "根目录",
		ParentID:   nil,
		FilePath:   "/",
		Breadcrumb: []diskRes.BreadcrumbItem{{FileID: nil, FileName: "根目录", FilePath: "/"}},
	}
	segs := splitPathSegments(p)
	if len(segs) == 0 {
		return resp, nil
	}

	// 累积各段全路径,单次查询所有祖先目录(独立 Find,无 finisher 污染)
	paths := make([]string, 0, len(segs))
	cur := ""
	for _, seg := range segs {
		cur = cur + "/" + seg
		paths = append(paths, cur)
	}
	var folders []disk.DiskFile
	if err := global.OPS_DB.WithContext(ctx).
		Where("user_id = ? AND is_directory = ? AND path IN ?", userId, true, paths).
		Find(&folders).Error; err != nil {
		return resp, err
	}
	if len(folders) != len(paths) {
		// 存在不存在的目录段,路径无效
		return resp, errors.New("路径不存在")
	}

	byPath := make(map[string]disk.DiskFile, len(folders))
	for _, f := range folders {
		byPath[f.Path] = f
	}

	breadcrumb := []diskRes.BreadcrumbItem{{FileID: nil, FileName: "根目录", FilePath: "/"}}
	var lastID, prevID interface{}
	cur = ""
	for _, seg := range segs {
		cur = cur + "/" + seg
		f := byPath[cur]
		breadcrumb = append(breadcrumb, diskRes.BreadcrumbItem{FileID: f.FileId, FileName: seg, FilePath: cur})
		prevID = lastID
		lastID = f.FileId
	}

	resp.FileID = lastID
	resp.FileName = segs[len(segs)-1]
	resp.ParentID = prevID
	resp.FilePath = "/" + strings.Join(segs, "/")
	resp.Breadcrumb = breadcrumb
	return resp, nil
}

// splitPathSegments 将路径拆为目录段(去首尾斜杠、过滤空段)。
func splitPathSegments(p string) []string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

// ===================== 第2期 文件 CRUD =====================
// 语义要点(适配本项目 disk_files 模型:path=自身全路径 + parent_id 树 + status 回收站):
//   - rename/move 文件夹时,事务内用 CONCAT(新前缀, SUBSTR(path, 旧长度+1)) WHERE path LIKE 旧前缀/%
//     一条 SQL 重写整棵子树的 path(借鉴 remote,适配"path=自身全路径")。
//   - 同名冲突统一报错(非自动改名),简洁一致。
//   - 第2期为纯元数据操作(当前无真实上传文件,storage_path 为空);物理存储/ref_count 处理留 TODO 给第3期。
//   - 读操作一律用 fresh session(global.OPS_DB.WithContext),避 finisher 污染事务 tx(见 backend-layer-rules)。

// validateFileName 校验文件/目录名合法性(对齐 remote ValidateFolderName 精简版)。
// 规则:非空;首尾非点;≤255;禁 /\:*?"<>|;禁 Windows 保留名(CON/PRN/AUX/NUL/COM1-9/LPT1-9)。
func validateFileName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return errors.New("文件名不能为空")
	}
	if len(name) > 255 {
		return errors.New("文件名过长(>255)")
	}
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") {
		return errors.New("文件名首尾不能为点")
	}
	for _, ch := range `\/:*?"<>|` {
		if strings.ContainsRune(name, ch) {
			return errors.New(`文件名不能包含特殊字符 / \ : * ? " < > |`)
		}
	}
	base := name
	if i := strings.LastIndex(name, "."); i > 0 {
		base = name[:i]
	}
	upper := strings.ToUpper(base)
	switch upper {
	case "CON", "PRN", "AUX", "NUL":
		return errors.New("文件名不能为系统保留名")
	}
	for _, p := range []string{"COM", "LPT"} {
		if strings.HasPrefix(upper, p) {
			if n, err := strconv.Atoi(strings.TrimPrefix(upper, p)); err == nil && n >= 1 && n <= 9 {
				return errors.New("文件名不能为系统保留名")
			}
		}
	}
	// 禁控制字符(含空字节 \x00、换行 \n \r、DEL 0x7f),防文件系统异常/日志污染/路径注入
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return errors.New("文件名不能包含控制字符")
		}
	}
	return nil
}

// joinPath 拼接父目录全路径 + 名称:parentPath 为 "/" 或 "" → "/name";否则 parentPath+"/"+name。
func joinPath(parentPath, name string) string {
	if parentPath == "" || parentPath == "/" {
		return "/" + name
	}
	return parentPath + "/" + name
}

// parentDir 取全路径的父目录:"/a/b"→"/a";"/a"→"/";"/"→"/"。
func parentDir(p string) string {
	if p == "" || p == "/" {
		return "/"
	}
	if idx := strings.LastIndex(p, "/"); idx > 0 {
		return p[:idx]
	}
	return "/"
}

// parseFileIds 将前端 string ID 列表转为 []int64,跳过非法/非正项。
func parseFileIds(ss []string) []int64 {
	out := make([]int64, 0, len(ss))
	for _, s := range ss {
		if n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil && n > 0 {
			out = append(out, n)
		}
	}
	return out
}

// resolveFolderByPath 按全路径解析目录记录。根目录("/")→哨兵(FileId=0,Path="/")found=true;
// 命中→其记录;不存在→found=false。fresh session(Take 是 finisher)。
func (s *DiskFileService) resolveFolderByPath(ctx context.Context, userId int64, folderPath string) (disk.DiskFile, bool, error) {
	if folderPath == "" || folderPath == "/" {
		return disk.DiskFile{FileId: 0, Path: "/"}, true, nil
	}
	var f disk.DiskFile
	e := global.OPS_DB.WithContext(ctx).
		Where("user_id = ? AND path = ? AND is_directory = ?", userId, folderPath, true).
		Take(&f).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return f, false, nil
	}
	if e != nil {
		return f, false, e
	}
	return f, true, nil
}

// EnsureFolderByRelPath 解析上传目标父目录并按相对路径懒建子目录(文件夹上传共用)。
// currentDirectory 必须存在且属当前用户;relativePath 按层级逐段建(每段走 validateFileName)。
// 整体穿越拦截:relativePath 不得含 ..;返回最终父目录。DiskUploadService.resolveFolderForUpload 共用此方法。
func (s *DiskFileService) EnsureFolderByRelPath(ctx context.Context, userId int64, currentDirectory, relativePath string) (disk.DiskFile, error) {
	if strings.Contains(relativePath, "..") {
		return disk.DiskFile{}, errors.New("非法上传路径")
	}
	parent, found, err := s.resolveFolderByPath(ctx, userId, currentDirectory)
	if err != nil {
		return disk.DiskFile{}, err
	}
	if !found {
		return disk.DiskFile{}, errors.New("目标目录不存在: " + currentDirectory)
	}
	cur := parent
	curPath := currentDirectory
	if curPath == "" {
		curPath = "/"
	}
	for _, seg := range splitPathSegments(relativePath) {
		if e := validateFileName(seg); e != nil {
			return disk.DiskFile{}, e
		}
		var next disk.DiskFile
		e := global.OPS_DB.WithContext(ctx).
			Where("user_id = ? AND parent_id = ? AND name = ? AND is_directory = ?", userId, cur.FileId, seg, true).
			Take(&next).Error
		if errors.Is(e, gorm.ErrRecordNotFound) {
			next = disk.DiskFile{
				UserId:      userId,
				ParentId:    cur.FileId,
				Name:        seg,
				IsDirectory: true,
				Path:        joinPath(curPath, seg),
				Status:      disk.DiskFileStatusNormal,
				RefCount:    1,
			}
			next.CreateBy = userId
			next.UpdateBy = userId
			if e = global.OPS_DB.WithContext(ctx).Create(&next).Error; e != nil {
				return disk.DiskFile{}, e
			}
		} else if e != nil {
			return disk.DiskFile{}, e
		}
		cur = next
		curPath = joinPath(curPath, seg)
	}
	return cur, nil
}

// EnsureFolders 批量预建目录(POST /file-meta/ensure-folders,文件夹上传前预建含空目录)。
// 逐个 relativePath 懒建;已存在则复用;路径非法/穿越则整批失败。幂等,可重复调用。
func (s *DiskFileService) EnsureFolders(ctx context.Context, userId int64, currentDirectory string, paths []string) error {
	for _, rel := range paths {
		if _, err := s.EnsureFolderByRelPath(ctx, userId, currentDirectory, rel); err != nil {
			return err
		}
	}
	return nil
}
func (s *DiskFileService) sameNameExists(ctx context.Context, userId, parentId int64, name string, excludeId int64) (bool, error) {
	var cnt int64
	q := global.OPS_DB.WithContext(ctx).Model(&disk.DiskFile{}).
		Where("user_id = ? AND parent_id = ? AND name = ? AND status = ?",
			userId, parentId, name, disk.DiskFileStatusNormal)
	if excludeId > 0 {
		q = q.Where("file_id <> ?", excludeId)
	}
	if err := q.Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// Mkdir 新建文件夹(对齐 POST /file-meta/mkdir)。父目录存在 + 同级无同名 + 名合法。
func (s *DiskFileService) Mkdir(ctx context.Context, userId int64, req diskReq.MkdirReq) error {
	name := strings.TrimSpace(req.FolderName)
	if err := validateFileName(name); err != nil {
		return err
	}
	parent, found, err := s.resolveFolderByPath(ctx, userId, req.ParentPath)
	if err != nil {
		return err
	}
	if !found {
		return errors.New("父目录不存在")
	}
	dup, err := s.sameNameExists(ctx, userId, parent.FileId, name, 0)
	if err != nil {
		return err
	}
	if dup {
		return errors.New("当前目录下已存在同名项")
	}
	f := disk.DiskFile{
		UserId:      userId,
		ParentId:    parent.FileId,
		Name:        name,
		IsDirectory: true,
		Path:        joinPath(parent.Path, name),
		Status:      disk.DiskFileStatusNormal,
		RefCount:    1,
	}
	f.CreateBy = userId
	f.UpdateBy = userId
	return global.OPS_DB.WithContext(ctx).Create(&f).Error
}

// CreateFile 新建空文件(对齐 POST /file-meta/create-file)。
// 校验 父目录存在 + 同级无同名 + 名合法 → 上传 0 字节占位对象到 OSS → 写 disk_files。
// QuickHash/StrongHash 留空:避免被 CheckUpload 误判秒传(空文件不参与秒传指纹匹配)。
// DB 写失败时回滚已上传对象(比 Merge 更稳)。
func (s *DiskFileService) CreateFile(ctx context.Context, userId int64, req diskReq.CreateFileReq) (diskRes.CreateFileResp, error) {
	var resp diskRes.CreateFileResp
	name := strings.TrimSpace(req.FileName)
	if err := validateFileName(name); err != nil {
		return resp, err
	}
	parent, found, err := s.resolveFolderByPath(ctx, userId, req.ParentPath)
	if err != nil {
		return resp, err
	}
	if !found {
		return resp, errors.New("父目录不存在")
	}
	dup, err := s.sameNameExists(ctx, userId, parent.FileId, name, 0)
	if err != nil {
		return resp, err
	}
	if dup {
		return resp, errors.New("当前目录下已存在同名项")
	}

	// 0 字节占位对象落对象存储(照搬 Merge 的 OSS 推送范式)
	tmp, err := os.CreateTemp("", "disk-createfile-*")
	if err != nil {
		return resp, err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath) // 清理空临时文件

	fh, cleanup, err := upload.BuildFileHeader(tmpPath, "file", name)
	if err != nil {
		return resp, err
	}
	defer cleanup()
	oss := upload.NewOss()
	_, key, err := oss.UploadFile(ctx, fh)
	if err != nil {
		return resp, err
	}

	ext := ""
	if i := strings.LastIndex(name, "."); i >= 0 {
		ext = strings.TrimPrefix(name[i:], ".")
	}
	f := disk.DiskFile{
		UserId:      userId,
		ParentId:    parent.FileId,
		Name:        name,
		IsDirectory: false,
		Path:        joinPath(parent.Path, name),
		Size:        0,
		ExtendName:  ext,
		ContentType: mime.TypeByExtension("." + ext),
		Md5:         "d41d8cd98f00b204e9800998ecf8427e", // 空内容 MD5(留档,不影响秒传)
		QuickHash:   "",                                 // 留空:防 CheckUpload 误判秒传
		StrongHash:  "",
		StorageType: global.OPS_CONFIG.System.OssType,
		StoragePath: key,
		RefCount:    1,
		Status:      disk.DiskFileStatusNormal,
	}
	f.CreateBy = userId
	f.UpdateBy = userId
	if err = global.OPS_DB.WithContext(ctx).Create(&f).Error; err != nil {
		_ = oss.DeleteFile(ctx, key) // DB 失败回滚已上传对象
		return resp, err
	}
	resp.FileId = strconv.FormatInt(f.FileId, 10)
	return resp, nil
}

// Rename 重命名(对齐 POST /file-meta/rename)。文件夹时事务内级联重写后代 path 前缀。
func (s *DiskFileService) Rename(ctx context.Context, userId int64, req diskReq.RenameReq) error {
	name := strings.TrimSpace(req.NewName)
	if err := validateFileName(name); err != nil {
		return err
	}
	var f disk.DiskFile
	if e := global.OPS_DB.WithContext(ctx).Where("file_id = ? AND user_id = ?", req.FileId, userId).Take(&f).Error; e != nil {
		return errors.New("文件不存在")
	}
	if f.Name == name {
		return nil
	}
	dup, err := s.sameNameExists(ctx, userId, f.ParentId, name, f.FileId)
	if err != nil {
		return err
	}
	if dup {
		return errors.New("当前目录下已存在同名项")
	}
	oldFullPath := f.Path
	newFullPath := joinPath(parentDir(f.Path), name)
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&disk.DiskFile{}).Where("file_id = ?", f.FileId).
			Updates(map[string]interface{}{"name": name, "path": newFullPath, "update_by": userId}).Error; e != nil {
			return e
		}
		if f.IsDirectory {
			// 后代 path 前缀替换:CONCAT(新前缀, SUBSTR(path, 旧长度+1)) WHERE path LIKE 旧前缀/%
			sql := "UPDATE disk_files SET path = CONCAT(?, SUBSTR(path, ?)) WHERE user_id = ? AND path LIKE ? AND deleted_at IS NULL"
			return tx.Exec(sql, newFullPath, len(oldFullPath)+1, userId, oldFullPath+"/%").Error
		}
		return nil
	})
}

// Move 移动(对齐 PUT /file-meta/move)。批量;循环校验(不能移入自身/后代);文件夹级联后代 path。
func (s *DiskFileService) Move(ctx context.Context, userId int64, req diskReq.MoveReq) error {
	ids := parseFileIds(req.FileIds)
	if len(ids) == 0 {
		return errors.New("未选择文件")
	}
	if strings.Contains(req.TargetPath, "..") {
		return errors.New("非法目标路径")
	}
	target, found, err := s.resolveFolderByPath(ctx, userId, req.TargetPath)
	if err != nil {
		return err
	}
	if !found {
		return errors.New("目标目录不存在")
	}
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			var f disk.DiskFile
			if e := global.OPS_DB.WithContext(ctx).Where("file_id = ? AND user_id = ?", id, userId).Take(&f).Error; e != nil {
				return errors.New("文件不存在(fileId=" + strconv.FormatInt(id, 10) + ")")
			}
			// 循环校验:目标不能是自身或自身后代
			if req.TargetPath == f.Path || strings.HasPrefix(req.TargetPath, f.Path+"/") {
				return errors.New("不能将文件夹移动到自身或其子目录中")
			}
			dup, e := s.sameNameExists(ctx, userId, target.FileId, f.Name, 0)
			if e != nil {
				return e
			}
			if dup {
				return errors.New("目标目录下已存在同名项: " + f.Name)
			}
			srcFullPath := f.Path
			newFullPath := joinPath(target.Path, f.Name)
			if e := tx.Model(&disk.DiskFile{}).Where("file_id = ?", f.FileId).
				Updates(map[string]interface{}{"parent_id": target.FileId, "path": newFullPath, "update_by": userId}).Error; e != nil {
				return e
			}
			if f.IsDirectory {
				sql := "UPDATE disk_files SET path = CONCAT(?, SUBSTR(path, ?)) WHERE user_id = ? AND path LIKE ? AND deleted_at IS NULL"
				if e := tx.Exec(sql, newFullPath, len(srcFullPath)+1, userId, srcFullPath+"/%").Error; e != nil {
					return e
				}
			}
		}
		return nil
	})
}

// Copy 复制(对齐 POST /file-meta/copy)。批量;文件夹递归深拷贝整棵子树。
// 复制共享物理对象(storage_path 复用):新节点 ref_count=1,源文件节点 ref_count++(与 mergeInstant 秒传复用一致)。
func (s *DiskFileService) Copy(ctx context.Context, userId int64, req diskReq.CopyReq) error {
	ids := parseFileIds(req.FileIds)
	if len(ids) == 0 {
		return errors.New("未选择文件")
	}
	if strings.Contains(req.TargetPath, "..") {
		return errors.New("非法目标路径")
	}
	target, found, err := s.resolveFolderByPath(ctx, userId, req.TargetPath)
	if err != nil {
		return err
	}
	if !found {
		return errors.New("目标目录不存在")
	}
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			var f disk.DiskFile
			if e := global.OPS_DB.WithContext(ctx).Where("file_id = ? AND user_id = ?", id, userId).Take(&f).Error; e != nil {
				return errors.New("文件不存在(fileId=" + strconv.FormatInt(id, 10) + ")")
			}
			dup, e := s.sameNameExists(ctx, userId, target.FileId, f.Name, 0)
			if e != nil {
				return e
			}
			if dup {
				return errors.New("目标目录下已存在同名项: " + f.Name)
			}
			srcFullPath := f.Path
			newFullPath := joinPath(target.Path, f.Name)
			newId, e := s.copyOne(tx, &f, target.FileId, newFullPath, userId)
			if e != nil {
				return e
			}
			if f.IsDirectory {
				if e := s.copyDescendants(tx, ctx, userId, srcFullPath, newFullPath, newId); e != nil {
					return e
				}
			}
		}
		return nil
	})
}

// copyOne 在事务内复制单条记录(新 file_id 自增),返回新 id。
func (s *DiskFileService) copyOne(tx *gorm.DB, src *disk.DiskFile, newParentId int64, newPath string, userId int64) (int64, error) {
	nf := disk.DiskFile{
		UserId:      src.UserId,
		ParentId:    newParentId,
		Name:        src.Name,
		IsDirectory: src.IsDirectory,
		Path:        newPath,
		Size:        src.Size,
		ExtendName:  src.ExtendName,
		ContentType: src.ContentType,
		Md5:         src.Md5,
		QuickHash:   src.QuickHash,
		StrongHash:  src.StrongHash,
		MidHash:     src.MidHash,
		StorageType: src.StorageType,
		StoragePath: src.StoragePath,
		RefCount:    1, // 新节点持有一份引用(原 src.RefCount 拷贝致计数虚高/错乱)
		Status:      disk.DiskFileStatusNormal,
	}
	nf.CreateBy = userId
	nf.UpdateBy = userId
	if e := tx.Create(&nf).Error; e != nil {
		return 0, e
	}
	// 共享物理对象(文件且 storage_path 非空):源 ref_count++,与 mergeInstant 秒传复用一致
	if !src.IsDirectory && src.StoragePath != "" {
		if e := tx.Model(&disk.DiskFile{}).Where("file_id = ?", src.FileId).
			UpdateColumn("ref_count", gorm.Expr("ref_count + 1")).Error; e != nil {
			return 0, e
		}
	}
	return nf.FileId, nil
}

// copyDescendants 递归复制 srcFullPath 下所有正常后代到 newFullPath 下。
// 按 path 升序保证父先于子;newIdByNewPath 维护"新 path→新 file_id"以回填后代 parent_id。
// 依赖不变式:status=1 的后代其祖先必均为 status=1(因 MoveToTrash 级联回收),故父必已复制。
func (s *DiskFileService) copyDescendants(tx *gorm.DB, ctx context.Context, userId int64, srcFullPath, newFullPath string, newRootId int64) error {
	var children []disk.DiskFile
	if e := global.OPS_DB.WithContext(ctx).
		Where("user_id = ? AND path LIKE ? AND status = ?", userId, srcFullPath+"/%", disk.DiskFileStatusNormal).
		Order("path ASC").Find(&children).Error; e != nil {
		return e
	}
	newIdByNewPath := map[string]int64{newFullPath: newRootId}
	for _, child := range children {
		suffix := child.Path[len(srcFullPath):] // "/sub" 或 "/sub/a.txt"
		childNewPath := newFullPath + suffix
		newParentId := newIdByNewPath[parentDir(childNewPath)] // 父必已复制(path 升序)
		copied := disk.DiskFile{
			UserId:      child.UserId,
			ParentId:    newParentId,
			Name:        child.Name,
			IsDirectory: child.IsDirectory,
			Path:        childNewPath,
			Size:        child.Size,
			ExtendName:  child.ExtendName,
			ContentType: child.ContentType,
			Md5:         child.Md5,
			QuickHash:   child.QuickHash,
			StrongHash:  child.StrongHash,
			MidHash:     child.MidHash,
			StorageType: child.StorageType,
			StoragePath: child.StoragePath,
			RefCount:    1, // 新节点持有一份引用(原 child.RefCount 拷贝致计数虚高/错乱)
			Status:      disk.DiskFileStatusNormal,
		}
		copied.CreateBy = userId
		copied.UpdateBy = userId
		if e := tx.Create(&copied).Error; e != nil {
			return e
		}
		// 共享物理对象(文件且 storage_path 非空):源 child ref_count++,与 mergeInstant 一致
		if !child.IsDirectory && child.StoragePath != "" {
			if e := tx.Model(&disk.DiskFile{}).Where("file_id = ?", child.FileId).
				UpdateColumn("ref_count", gorm.Expr("ref_count + 1")).Error; e != nil {
				return e
			}
		}
		newIdByNewPath[childNewPath] = copied.FileId
	}
	return nil
}

// MoveToTrash 删除→移入回收站(对齐 POST /file-meta/delete)。批量;文件夹级联后代进回收站(status=2)。
// 第2期纯元数据(不动物理存储);彻底删除/物理释放留待回收站独立页(第6期)+第3期上传落地。
func (s *DiskFileService) MoveToTrash(ctx context.Context, userId int64, req diskReq.DeleteReq) error {
	ids := parseFileIds(req.FileIds)
	if len(ids) == 0 {
		return errors.New("未选择文件")
	}
	totalSize := int64(0)
	err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			var f disk.DiskFile
			if e := global.OPS_DB.WithContext(ctx).Where("file_id = ? AND user_id = ?", id, userId).Take(&f).Error; e != nil {
				return errors.New("文件不存在(fileId=" + strconv.FormatInt(id, 10) + ")")
			}
			// 先算待删子树 size(改 status 前,status=normal 口径),用于配额释放对账
			var subSum float64
			if e := tx.Raw("SELECT COALESCE(SUM(size), 0) FROM disk_files WHERE user_id = ? AND path LIKE ? AND is_directory = ? AND status = ?",
				userId, f.Path+"/%", false, disk.DiskFileStatusNormal).Scan(&subSum).Error; e != nil {
				return e
			}
			totalSize += f.Size + int64(subSum)
			if e := tx.Model(&disk.DiskFile{}).Where("file_id = ?", f.FileId).
				Updates(map[string]interface{}{"status": disk.DiskFileStatusTrashed, "update_by": userId}).Error; e != nil {
				return e
			}
			if f.IsDirectory {
				// 后代一并进回收站
				if e := tx.Model(&disk.DiskFile{}).
					Where("user_id = ? AND path LIKE ? AND deleted_at IS NULL", userId, f.Path+"/%").
					Updates(map[string]interface{}{"status": disk.DiskFileStatusTrashed}).Error; e != nil {
					return e
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// 释放配额(子树文件 size 之和,GREATEST 下限兜底防负;best effort 不阻断删除)
	releaseUserSpace(ctx, userId, totalSize)
	return nil
}

// GetFolderTree 返回用户全部正常目录的树(对齐 GET /file-meta/folder-tree,移动/复制目标选择器用)。
// 按 parent_id 分组后从根(parent_id=0)递归构建,避免值拷贝导致的子节点丢失。
func (s *DiskFileService) GetFolderTree(ctx context.Context, userId int64) ([]diskRes.FolderTreeNode, error) {
	var dirs []disk.DiskFile
	if err := global.OPS_DB.WithContext(ctx).
		Where("user_id = ? AND is_directory = ? AND status = ?", userId, true, disk.DiskFileStatusNormal).
		Order("name ASC").Find(&dirs).Error; err != nil {
		return nil, err
	}
	childrenOf := make(map[int64][]disk.DiskFile)
	for i := range dirs {
		childrenOf[dirs[i].ParentId] = append(childrenOf[dirs[i].ParentId], dirs[i])
	}
	var build func(parentId int64) []diskRes.FolderTreeNode
	build = func(parentId int64) []diskRes.FolderTreeNode {
		kids := childrenOf[parentId]
		nodes := make([]diskRes.FolderTreeNode, 0, len(kids))
		for _, k := range kids {
			nodes = append(nodes, diskRes.FolderTreeNode{
				ID:       strconv.FormatInt(k.FileId, 10),
				Name:     k.Name,
				Path:     k.Path,
				Children: build(k.FileId),
			})
		}
		return nodes
	}
	return build(0), nil
}
