# 压缩包解压功能实现计划（解压到当前目录 / 解压到...）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 打通「解压到当前目录」「解压到...」全链路——后端补全「暂存→上传→入库登记」并加配额/越权/zip 炸弹防护，前端接线两个按钮（含目标目录选择器），前后端类型一一对应。

**Architecture:** 方案 A（暂存→校验→上传入库事务）。源文件下载到本地 temp → 解压到唯一暂存目录 → 安全校验 → 计算 effectiveDest（按 intoSubfolder）→ 规划条目（冲突自动重命名）→ 配额预检 → 事务内逐个上传到 rustfs/移动到本地 + 批量写 `disk_files` → 更新配额 → 失败全回滚清理。

**Tech Stack:** Go 1.26 + Gin + GORM（后端 submodule `backend/`）；Vue3 + TS + Naive UI + UnoCSS（前端 submodule `frontend/`）；存储 rustfs(S3 兼容)/local；测试 sqlite-in-memory + `t.TempDir()`。

## Global Constraints

- 后端代码在 `backend/` 子模块；前端代码在 `frontend/` 子模块。各任务在对应子模块内提交。模块名 `github.com/hllkk/devopsAdmin`。
- **类型安全**：前端禁止 `any`；前后端字段名/类型逐一对应（`fileId / destPath / intoSubfolder`），删除旧冗余签名。
- **安全**：zip slip（`..`/绝对路径/危险字符拒绝，`ValidatePath` 归一化）、zip bomb（≤50000 文件/≤50GB）、越权（源读权限 + dest 写权限 + 归属当前用户）、配额预检、文件名净化（`ValidateFolderName`）。
- **构建规范**：`if/else` 嵌套 ≤3 层（超 3 层用 switch/早返回）；函数 <50 行；构建后删除编译产物；改接口同步 Swagger 并 `swag init`。
- **国际化**：先定义类型再写翻译；zh-cn 与 en-us 同步。
- 提交消息用约定式提交（feat/fix/test/docs/chore）；归属已全局禁用，不加 Co-Authored-By。

## File Structure

后端（`backend/`）：
- `model/disk/disk_file_audit.go` — 新增 `FileOpExtract` 审计常量。
- `model/disk/archive.go` — 重写 `ExtractArchiveRequest`（path 化）。
- `api/v1/disk/disk_archive.go` — `ExtractArchive` handler 改签名 + Swagger。
- `service/disk/disk_archive.go` — 重写 `ExtractArchive` 编排 + 新增助手 `computeEffectiveDest`/`planExtraction`/`commitExtraction`/`uniqueNameFor`/`folderExistsByPath`。
- `service/disk/disk_archive_test.go` — 新增：纯函数 + 集成测试。
- `utils/archive_test.go` — 复用既有；本计划不改。

前端（`frontend/`）：
- `src/service/api/disk/archive.ts` — 重写 `fetchExtractArchive` + `ExtractArchiveParams`。
- `src/views/disk/modules/extract-to-dialog.vue` — 新建目标目录选择器。
- `src/views/disk/index.vue` — 接线 `extract-here` / `extract-to`。
- `src/components/disk/archive-action-dialog.vue` — 补 loading/禁用态透传。
- `src/locales/langs/zh-cn.ts`、`src/locales/langs/en-us.ts` — `page.disk.extract.*`。

---

## Task B1: 后端 — 审计常量 + path 化请求模型 + handler 改签名

**Files:**
- Modify: `backend/model/disk/disk_file_audit.go:7-25`（新增常量）
- Modify: `backend/model/disk/archive.go:13-17`（重写 `ExtractArchiveRequest`）
- Modify: `backend/api/v1/disk/disk_archive.go:68-92`（handler）
- Test: `backend/service/disk/disk_archive_test.go`（新建，仅本任务的 binding 测试）

**Interfaces:**
- Produces: `diskModel.ExtractArchiveRequest{FileID, DestPath string; IntoSubfolder bool}`；`ArchiveService.ExtractArchive(fileID, destPath string, intoSubfolder bool, userID int64, roleCode string) error`（本任务先放一个 stub 实现，B3 替换）；常量 `disk.FileOpExtract = "extract"`。

- [ ] **Step 1: 新增审计常量**

`backend/model/disk/disk_file_audit.go` 的操作常量块（`FileOpTransfer = "transfer"` 之后）追加：

```go
	FileOpExtract      = "extract" // 解压压缩包
```

- [ ] **Step 2: 重写请求模型**

`backend/model/disk/archive.go` 用下面内容替换 `ExtractArchiveRequest`（保留 `ArchiveEntry` 不动）：

```go
// ExtractArchiveRequest 解压请求
type ExtractArchiveRequest struct {
	FileID        string `json:"fileId" binding:"required"`           // 压缩包文件ID
	DestPath      string `json:"destPath" binding:"required"`         // 目标目录相对路径(/ 或 /docs)
	IntoSubfolder bool   `json:"intoSubfolder"`                       // true=在 destPath 下建以压缩包命名的子目录
}
```

- [ ] **Step 3: 写失败测试（请求绑定 + handler 入参）**

新建 `backend/service/disk/disk_archive_test.go`：

```go
package disk

import (
	"testing"

	diskModel "github.com/hllkk/devopsAdmin/model/disk"
)

func TestExtractArchiveRequest_Binding(t *testing.T) {
	// 校验 path 化后的请求结构字段语义
	req := diskModel.ExtractArchiveRequest{FileID: "123", DestPath: "/", IntoSubfolder: true}
	if req.FileID != "123" || req.DestPath != "/" || !req.IntoSubfolder {
		t.Fatalf("请求字段语义错误: %+v", req)
	}
	// 旧的 DestFolderID 字段必须已移除（编译期保证：此处不再引用）
}
```

- [ ] **Step 4: 运行测试确认通过（模型已改）**

Run: `cd backend && go test ./service/disk/ -run TestExtractArchiveRequest_Binding -v`
Expected: PASS

- [ ] **Step 5: 改 handler 签名 + Swagger（暂用 stub service）**

`backend/api/v1/disk/disk_archive.go` 用下面内容替换整个 `ExtractArchive` 函数（68-92 行）：

```go
// ExtractArchive 解压归档到目标目录
// @Tags Disk
// @Summary 解压压缩包到指定目录(支持解压到当前目录/解压到...)
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body disk.ExtractArchiveRequest true "解压参数"
// @Success 200 {object} response.Response
// @Router /archive/extract [post]
func (a *ArchiveApi) ExtractArchive(c *gin.Context) {
	var req diskModel.ExtractArchiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}

	userID := utils.GetUserID(c)
	if userID == 0 {
		response.FailWithMessage("未登录", c)
		return
	}
	claims, _ := utils.GetClaims(c)
	roleCode := claims.RoleCode

	if err := ArchiveService.ExtractArchive(req.FileID, req.DestPath, req.IntoSubfolder, userID, roleCode); err != nil {
		global.OPS_LOG.Error("ExtractArchive failed", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("解压成功", c)
}
```

`backend/service/disk/disk_archive.go` 用下面 stub 替换现有 `ExtractArchive`（135-185 行整段），B3 再替换为真实实现：

```go
// ExtractArchive 解压归档到目标目录(B1 stub,B3 替换为完整流水线)
func (s *ArchiveService) ExtractArchive(fileID, destPath string, intoSubfolder bool, userID int64, roleCode string) error {
	return fmt.Errorf("解压功能尚未实现")
}
```

- [ ] **Step 6: 编译确认**

Run: `cd backend && go build ./...`
Expected: 无错误（若 service 文件有未使用 import 如 `path/filepath` 暂时引用，保持；B3 会重新用到）。

- [ ] **Step 7: 提交**

```bash
cd backend
git add model/disk/disk_file_audit.go model/disk/archive.go api/v1/disk/disk_archive.go service/disk/disk_archive.go service/disk/disk_archive_test.go
git commit -m "feat(disk): 解压请求改为 path 化并接入审计常量(stub)"
```

---

## Task B2: 后端 — 冲突重命名 + 条目规划纯函数（TDD）

**Files:**
- Modify: `backend/service/disk/disk_archive.go`（新增 `uniqueNameFor`/`plannedEntry`/`planExtraction`/`folderExistsByPath`/`computeEffectiveDest`）
- Test: `backend/service/disk/disk_archive_test.go`

**Interfaces:**
- Produces:
  - `uniqueNameFor(name string, taken map[string]bool) string` — 给定已被占用名集合，返回不冲突的名（追加 `(1)/(2)`，保留扩展名）。
  - `type plannedEntry struct { RelFolder, Name string; IsFolder bool; Size int64 }` — 一条待登记条目（RelFolder 为相对 effectiveDest 的目录路径，如 `/` 或 `/sub`）。
  - `planExtraction(stageDir, effectiveDest string, userID int64) ([]plannedEntry, int64, error)` — 遍历暂存树 + 查 DB 解析冲突，返回有序条目（目录在前）与总大小。
  - `computeEffectiveDest(userID int64, destPath string, intoSubfolder bool, archiveBaseName string) (string, error)`。
  - `folderExistsByPath(userID int64, fullPath string) bool`。

- [ ] **Step 1: 写失败测试 — uniqueNameFor**

追加到 `backend/service/disk/disk_archive_test.go`：

```go
func TestUniqueNameFor(t *testing.T) {
	tests := []struct {
		name string
		in   string
		taken map[string]bool
		want string
	}{
		{"无冲突", "a.txt", map[string]bool{}, "a.txt"},
		{"同名追加序号", "a.txt", map[string]bool{"a.txt": true}, "a.txt (1)"},
		{"序号递增", "a.txt", map[string]bool{"a.txt": true, "a.txt (1)": true}, "a.txt (2)"},
		{"目录名无扩展", "proj", map[string]bool{"proj": true}, "proj (1)"},
		{"多扩展名仅分离最后一段", "a.tar.gz", map[string]bool{"a.tar.gz": true}, "a.tar.gz (1)"},
	}
	for _, tt := range tests {
		got := uniqueNameFor(tt.in, tt.taken)
		if got != tt.want {
			t.Errorf("%s: got %q want %q", tt.name, got, tt.want)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend && go test ./service/disk/ -run TestUniqueNameFor -v`
Expected: FAIL（`uniqueNameFor undefined`）

- [ ] **Step 3: 实现 uniqueNameFor**

追加到 `backend/service/disk/disk_archive.go`：

```go
// uniqueNameFor 在已占用名集合 taken 中为 name 生成不冲突的名(追加 (1)/(2)…,保留扩展名)
func uniqueNameFor(name string, taken map[string]bool) string {
	if !taken[name] {
		return name
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if !taken[candidate] {
			return candidate
		}
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd backend && go test ./service/disk/ -run TestUniqueNameFor -v`
Expected: PASS

- [ ] **Step 5: 写失败测试 — planExtraction(本地暂存树)**

追加到测试文件（用真实暂存目录，验证规划+冲突解析+目录排序+总大小）：

```go
import (
	"os"
	"path/filepath"
)

func TestPlanExtraction_SortsAndSums(t *testing.T) {
	setupTestDB(t) // 仅用 DB 做冲突查询,此处目标目录为空 → 无冲突
	stage := t.TempDir()
	// 构造: stage/a.txt(10) + stage/sub/b.txt(20)
	if err := os.MkdirAll(filepath.Join(stage, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "a.txt"), make([]byte, 10), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "sub", "b.txt"), make([]byte, 20), 0o644); err != nil {
		t.Fatal(err)
	}

	plans, total, err := planExtraction(stage, "/dst", 1)
	if err != nil {
		t.Fatalf("planExtraction 失败: %v", err)
	}
	if total != 30 {
		t.Errorf("总大小: got %d want 30", total)
	}
	// 目录条目必须排在同层文件之前
	first := plans[0]
	if !first.IsFolder || first.Name != "sub" {
		t.Errorf("首条应为目录 sub, got %+v", first)
	}
	// 校验相对目录路径
	wantFolders := map[string]bool{"/": true, "/dst/sub": true}
	gotFolders := map[string]bool{}
	for _, p := range plans {
		gotFolders[p.RelFolder] = true
	}
	for k := range wantFolders {
		if !gotFolders[k] {
			t.Errorf("缺少相对目录 %s", k)
		}
	}
}

func TestPlanExtraction_ConflictAutoRenames(t *testing.T) {
	db := setupTestDB(t)
	// 目标 /dst 下已存在 a.txt → 规划时应解析为 a.txt (1)
	db.Create(&diskModel.File{OPS_MODEL: global.OPS_MODEL{ID: 1}, UserID: 1, Name: "dst", Path: "/", IsFolder: true})
	db.Create(&diskModel.File{OPS_MODEL: global.OPS_MODEL{ID: 2}, UserID: 1, Name: "a.txt", Path: "/dst", IsFolder: false})

	stage := t.TempDir()
	if err := os.WriteFile(filepath.Join(stage, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	plans, _, err := planExtraction(stage, "/dst", 1)
	if err != nil {
		t.Fatalf("planExtraction 失败: %v", err)
	}
	if len(plans) != 1 || plans[0].Name != "a.txt (1)" {
		t.Fatalf("冲突重命名错误: %+v", plans)
	}
}
```

测试文件 import 块顶部需补：`"github.com/hllkk/devopsAdmin/global"`、`diskModel "github.com/hllkk/devopsAdmin/model/disk"`（`diskModel` 已有；补 `global`）。

- [ ] **Step 6: 运行测试确认失败**

Run: `cd backend && go test ./service/disk/ -run TestPlanExtraction -v`
Expected: FAIL（`planExtraction undefined`）

- [ ] **Step 7: 实现 plannedEntry + folderExistsByPath + planExtraction**

追加到 `backend/service/disk/disk_archive.go`：

```go
// plannedEntry 一条待登记的解压条目
type plannedEntry struct {
	StageRelPath string // 暂存目录内的原始相对路径(用于定位物理文件,Name 可能已被重命名)
	RelFolder    string // 相对 effectiveDest 的目录路径(/ 或 /sub)
	Name         string // 最终文件名(可能已冲突重命名)
	IsFolder     bool
	Size         int64
}

// folderExistsByPath 判断某用户下是否存在某 full path 的文件夹记录(/ 视为根,恒存在)
func folderExistsByPath(userID int64, fullPath string) bool {
	if fullPath == "/" || fullPath == "" {
		return true
	}
	parentPath := utils.NormalizePathWithFilepath(filepath.Dir(strings.TrimSuffix(fullPath, "/")))
	name := filepath.Base(strings.TrimSuffix(fullPath, "/"))
	var count int64
	global.OPS_DB.Model(&diskModel.File{}).
		Where("user_id = ? AND path = ? AND name = ? AND is_folder = ?", userID, parentPath, name, true).
		Count(&count)
	return count > 0
}

// takenNamesIn 查询某用户某目录下已存在的文件/文件夹名集合
func takenNamesIn(userID int64, folderPath string) map[string]bool {
	taken := make(map[string]bool)
	var rows []diskModel.File
	global.OPS_DB.Select("name").Where("user_id = ? AND path = ?", userID, folderPath).Find(&rows)
	for _, r := range rows {
		taken[r.Name] = true
	}
	return taken
}

// planExtraction 遍历暂存目录,解析冲突,返回有序条目(目录在前)与总大小
func planExtraction(stageDir, effectiveDest string, userID int64) ([]plannedEntry, int64, error) {
	type rawEntry struct {
		stageRel  string // 暂存目录内原始相对路径
		relFolder string
		name      string
		isFolder  bool
		size      int64
	}
	var raws []rawEntry
	var total int64

	// 每个目录独立维护"已占用名"(DB 现有 + 本轮已规划),保证同目录多处冲突不撞车
	takenByFolder := make(map[string]map[string]bool)
	getTaken := func(folder string) map[string]bool {
		if m, ok := takenByFolder[folder]; ok {
			return m
		}
		m := takenNamesIn(userID, folder)
		takenByFolder[folder] = m
		return m
	}

	walkErr := filepath.Walk(stageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == stageDir {
			return nil
		}
		rel, err := filepath.Rel(stageDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !utils.IsValidArchiveEntry(rel) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		dirRel := "/" + filepath.ToSlash(filepath.Dir(rel))
		if dirRel == "/." {
			dirRel = "/"
		}
		absFolder := utils.NormalizePathWithFilepath(effectiveDest + dirRel)
		raws = append(raws, rawEntry{rel, absFolder, info.Name(), info.IsDir(), info.Size()})
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	if walkErr != nil {
		return nil, 0, fmt.Errorf("遍历暂存目录失败: %w", walkErr)
	}

	// 先解析目录(按路径短→长,确保父目录先占名),再解析文件
	sort.SliceStable(raws, func(i, j int) bool {
		if raws[i].isFolder != raws[j].isFolder {
			return raws[i].isFolder // 目录在前
		}
		return len(raws[i].relFolder) < len(raws[j].relFolder)
	})

	plans := make([]plannedEntry, 0, len(raws))
	for _, r := range raws {
		taken := getTaken(r.relFolder)
		name := uniqueNameFor(r.name, taken)
		taken[name] = true
		rel := strings.TrimPrefix(r.relFolder, effectiveDest)
		if rel == "" {
			rel = "/"
		}
		plans = append(plans, plannedEntry{StageRelPath: r.stageRel, RelFolder: rel, Name: name, IsFolder: r.isFolder, Size: r.size})
	}
	return plans, total, nil
}
```

> 注：`utils.IsValidArchiveEntry`、`utils.NormalizePathWithFilepath` 已存在；`sort`、`os`、`filepath`、`strings` 已在文件 import 中（若缺则补）。

- [ ] **Step 8: 运行测试确认通过**

Run: `cd backend && go test ./service/disk/ -run TestPlanExtraction -v`
Expected: PASS

- [ ] **Step 9: 实现 computeEffectiveDest + 写测试**

追加实现到 `disk_archive.go`：

```go
// computeEffectiveDest 计算最终目标目录: intoSubfolder=true 时在 destPath 下建以压缩包名命名的子目录(冲突自动重命名)
func computeEffectiveDest(userID int64, destPath string, intoSubfolder bool, archiveBaseName string) (string, error) {
	destPath = utils.NormalizePathWithFilepath(destPath)
	if err := utils.ValidatePath(destPath); err != nil {
		return "", fmt.Errorf("目标路径非法: %w", err)
	}
	if !folderExistsByPath(userID, destPath) {
		return "", fmt.Errorf("目标目录不存在")
	}
	if !intoSubfolder {
		return destPath, nil
	}
	if err := utils.ValidateFolderName(archiveBaseName); err != nil {
		return "", fmt.Errorf("压缩包名非法: %w", err)
	}
	name := uniqueNameFor(archiveBaseName, takenNamesIn(userID, destPath))
	if destPath == "/" {
		return "/" + name, nil
	}
	return utils.NormalizePathWithFilepath(destPath + "/" + name), nil
}
```

追加测试：

```go
func TestComputeEffectiveDest(t *testing.T) {
	setupTestDB(t)
	// 直解模式
	got, err := computeEffectiveDest(1, "/", false, "x")
	if err != nil || got != "/" {
		t.Fatalf("直解根目录: got %q err %v", got, err)
	}
	// 子目录模式(根目录无冲突)
	got, err = computeEffectiveDest(1, "/", true, "myarchive")
	if err != nil || got != "/myarchive" {
		t.Fatalf("子目录模式: got %q err %v", got, err)
	}
}
```

Run: `cd backend && go test ./service/disk/ -run TestComputeEffectiveDest -v`
Expected: PASS

- [ ] **Step 10: 提交**

```bash
cd backend
git add service/disk/disk_archive.go service/disk/disk_archive_test.go
git commit -m "feat(disk): 解压冲突重命名与条目规划纯函数(含测试)"
```

---

## Task B3: 后端 — 重写 ExtractArchive 流水线（暂存→上传/移动→入库事务）

**Files:**
- Modify: `backend/service/disk/disk_archive.go`（替换 B1 的 stub；新增 `commitExtraction`）
- Test: `backend/service/disk/disk_archive_test.go`（集成测试）

**Interfaces:**
- Consumes: B2 的 `planExtraction`/`computeEffectiveDest`/`plannedEntry`；`utils` 解压与校验；`GetStorageProvider`/`UploadFromReader`；`QuotaService`；`FileService.ensureFolderRecords`；`FileAuditService`。
- Produces: `ArchiveService.ExtractArchive` 真实实现（供 handler 调用）。

- [ ] **Step 1: 写失败集成测试 — 本地存储端到端解压**

追加到测试文件（用 `archive/zip` 构造真实压缩包，本地存储，校验入库一致性）：

```go
import (
	"archive/zip"
	"github.com/hllkk/devopsAdmin/config"
)

// makeZip 构造一个含 a.txt + sub/b.txt 的 zip
func makeZip(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
	add := func(name, body string) {
		zf, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := zf.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	add("a.txt", "hello")
	add("sub/b.txt", "world")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestExtractArchive_LocalEndToEnd(t *testing.T) {
	db := setupTestDB(t)
	root := t.TempDir()
	global.OPS_CONFIG = config.Server{System: config.System{}, Local: config.Local{RootPath: root}}

	// 用户 1 的源压缩包(物理文件)
	srcDir := filepath.Join(root, "1")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(srcDir, "pkg.zip")
	makeZip(t, zipPath)
	db.Create(&diskModel.File{OPS_MODEL: global.OPS_MODEL{ID: 100}, UserID: 1, Name: "pkg.zip", Path: "/", IsFolder: false, Suffix: "zip", StorageType: "local", StoragePath: zipPath})

	svc := &ArchiveService{}
	// 解压到当前目录(根目录) → 应建子目录 /pkg
	err := svc.ExtractArchive("100", "/", true, 1, "COMMON")
	if err != nil {
		t.Fatalf("ExtractArchive 失败: %v", err)
	}

	// 校验入库: /pkg 目录 + /pkg/a.txt + /pkg/sub + /pkg/sub/b.txt
	var rows []diskModel.File
	db.Where("user_id = ? AND path LIKE ?", 1, "/pkg%").Find(&rows)
ByName := map[string]diskModel.File{}
	for _, r := range rows {
		ByName[r.Path+"/"+r.Name] = r
	}
	for _, want := range []string{"/pkg", "/pkg/a.txt", "/pkg/sub", "/pkg/sub/b.txt"} {
		if _, ok := ByName[want]; !ok {
			t.Errorf("缺少入库记录 %s", want)
		}
	}
	// 物理文件应存在
	if _, err := os.Stat(filepath.Join(root, "1", "pkg", "a.txt")); err != nil {
		t.Errorf("物理文件缺失 pkg/a.txt: %v", err)
	}
}
```

> 说明：本测试假定 `GetFileForPreviewWithPermission` 能按 ID 取到源文件（sqlite 中已建记录）。若该方法对 `shareId=""` 分支依赖更多字段，可在测试中按需补字段；以能跑通为准。

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend && go test ./service/disk/ -run TestExtractArchive_LocalEndToEnd -v`
Expected: FAIL（`解压功能尚未实现`）

- [ ] **Step 3: 实现 commitExtraction（事务：建目录→上传/移动→写记录）**

追加到 `backend/service/disk/disk_archive.go`：

```go
// commitExtraction 落地物理文件并事务批量入库:先上传/移动所有文件,再开事务写记录;任一阶段失败清理已落地产物。
func (s *ArchiveService) commitExtraction(userID int64, plans []plannedEntry, stageDir, effectiveDest string) (int64, error) {
	storageType := global.OPS_CONFIG.System.OssType
	if storageType == "" {
		storageType = "local"
	}
	provider := GetStorageProvider(StorageType(storageType))

	var artifacts []string // 已落地产物(本地物理路径 或 OSS 对象 key),失败时清理
	cleanup := func() {
		for _, a := range artifacts {
			_ = provider.DeleteObject(a) // 本地=os.Remove;rustfs=RemoveObject,统一清理
		}
	}

	// 阶段1:上传/移动所有文件条目(事务外,大文件流式上传不宜占事务)
	var totalSize int64
	for _, p := range plans {
		if p.IsFolder {
			continue
		}
		key, err := s.uploadOrMoveFile(userID, storageType, provider, p, stageDir, effectiveDest)
		if err != nil {
			cleanup()
			return 0, fmt.Errorf("落地文件失败 %s: %w", p.Name, err)
		}
		artifacts = append(artifacts, key)
		totalSize += p.Size
	}

	// 阶段2:事务批量写记录(目录在前)
	txErr := global.OPS_DB.Transaction(func(tx *gorm.DB) error {
		fs := &FileService{}
		// 先确保 effectiveDest 自身记录存在(intoSubfolder 新建子目录 / 扁平压缩包无子目录时必需;已存在则幂等)
		if effectiveDest != "/" && effectiveDest != "" {
			if err := fs.ensureFolderRecords(userID, effectiveDest); err != nil {
				return fmt.Errorf("建目标目录记录失败 %s: %w", effectiveDest, err)
			}
		}
		for _, p := range plans {
			folder := utils.NormalizePathWithFilepath(effectiveDest + p.RelFolder)
			if p.IsFolder && folder != "/" && folder != "" {
				if err := fs.ensureFolderRecords(userID, folder); err != nil {
					return fmt.Errorf("建目录记录失败 %s: %w", folder, err)
				}
			}
			rec := diskModel.File{
				UserID: userID, Name: p.Name, Path: folder, IsFolder: p.IsFolder,
				ContentType: utils.GetContentType(p.Name), Suffix: utils.GetFileSuffix(p.Name),
			}
			if p.IsFolder {
				rec.ContentType = "directory"
			} else {
				rec.Size = p.Size
				rec.StorageType = storageType
			}
			if err := tx.Create(&rec).Error; err != nil {
				return fmt.Errorf("写入记录失败 %s: %w", folder+"/"+p.Name, err)
			}
		}
		return nil
	})
	if txErr != nil {
		cleanup()
		return 0, txErr
	}
	return totalSize, nil
}

// uploadOrMoveFile 落地单个文件到最终存储,返回产物路径/对象 key。源取自 plan.StageRelPath(暂存原始名),终名用 plan.Name(可能已重命名)
func (s *ArchiveService) uploadOrMoveFile(userID int64, storageType string, provider StorageProvider, plan plannedEntry, stageDir, effectiveDest string) (string, error) {
	srcPath := filepath.Join(stageDir, filepath.FromSlash(plan.StageRelPath))
	folder := utils.NormalizePathWithFilepath(effectiveDest + plan.RelFolder)
	userBase, err := provider.BuildPath(userID, folder)
	if err != nil {
		return "", fmt.Errorf("构建存储路径失败: %w", err)
	}
	if storageType == "local" {
		if err := os.MkdirAll(userBase, 0o755); err != nil {
			return "", fmt.Errorf("创建目录失败: %w", err)
		}
		dest := filepath.Join(userBase, plan.Name)
		if err := os.Rename(srcPath, dest); err != nil {
			return "", fmt.Errorf("移动文件失败: %w", err)
		}
		return dest, nil
	}
	key := strings.TrimSuffix(userBase, "/") + "/" + plan.Name
	in, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("打开暂存文件失败: %w", err)
	}
	defer in.Close()
	if err := UploadFromReader(context.Background(), key, in, plan.Size, "application/octet-stream"); err != nil {
		return "", fmt.Errorf("上传失败: %w", err)
	}
	return key, nil
}
```

> 说明：`commitExtraction` 与 `uploadOrMoveFile` 为完整实现（非骨架）。顺序固定为「先上传/移动全部文件 → 再事务批量写记录 → 任一阶段失败用 `provider.DeleteObject` 清理已落地产物」，保证物理与 DB 最终一致。回滚清理对 local(`os.Remove`) 与 rustfs(`RemoveObject`) 统一走 `provider.DeleteObject`。源文件定位用 `plan.StageRelPath`（暂存原始名），终名用 `plan.Name`（可能已重命名）。

- [ ] **Step 4: 重写 ExtractArchive 编排（替换 stub）**

用下面内容替换 B1 的 stub `ExtractArchive`：

```go
// ExtractArchive 解压归档到目标目录: 暂存→校验→配额→上传/移动→入库事务→清理
func (s *ArchiveService) ExtractArchive(fileID, destPath string, intoSubfolder bool, userID int64, roleCode string) error {
	ps := &PreviewService{}
	downloadPerm := common.OperationPermissionDownload
	file, storageKey, storageType, err := ps.GetFileForPreviewWithPermission(fileID, userID, "", "", &downloadPerm)
	if err != nil {
		return fmt.Errorf("获取压缩文件失败: %w", err)
	}

	archivePath, srcCleanup, err := s.resolveLocalPath(storageType, storageKey)
	if err != nil {
		return fmt.Errorf("解析压缩文件路径失败: %w", err)
	}
	defer srcCleanup()

	compType, err := utils.DetectArchiveType(archivePath)
	if err != nil {
		return fmt.Errorf("检测压缩类型失败: %w", err)
	}
	if compType == "unknown" {
		return fmt.Errorf("不支持的压缩格式")
	}

	// 唯一暂存目录
	stageDir, err := os.MkdirTemp("", "extract-*")
	if err != nil {
		return fmt.Errorf("创建暂存目录失败: %w", err)
	}
	defer os.RemoveAll(stageDir)

	switch compType {
	case "rar", "rar5":
		err = utils.ExtractArchiveUnar(archivePath, stageDir)
	default:
		err = utils.ExtractArchive7z(archivePath, stageDir)
	}
	if err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}
	if err := utils.PostExtractValidation(stageDir); err != nil {
		return fmt.Errorf("解压后安全验证失败: %w", err)
	}

	archiveBaseName := strings.TrimSuffix(file.Name, filepath.Ext(file.Name))
	effectiveDest, err := computeEffectiveDest(userID, destPath, intoSubfolder, archiveBaseName)
	if err != nil {
		return err
	}

	plans, totalSize, err := planExtraction(stageDir, effectiveDest, userID)
	if err != nil {
		return err
	}
	if len(plans) == 0 {
		return fmt.Errorf("压缩包为空或全部条目被安全过滤")
	}

	// 配额预检
	quotaCheck, err := QuotaService.CheckQuota(userID, roleCode, totalSize)
	if err != nil {
		return fmt.Errorf("配额校验失败: %w", err)
	}
	if !quotaCheck.Allowed {
		return fmt.Errorf("存储空间不足: %s", quotaCheck.Reason)
	}

	committed, err := s.commitExtraction(userID, plans, stageDir, effectiveDest)
	if err != nil {
		return fmt.Errorf("入库失败: %w", err)
	}
	if err := QuotaService.UpdateUsedSpace(userID, committed); err != nil {
		global.OPS_LOG.Warn("更新已用空间失败", zap.Int64("delta", committed), zap.Error(err))
	}

	(&FileAuditService{}).Record(&diskModel.FileAuditLog{
		UserID: userID, OperationType: diskModel.FileOpExtract, FileID: file.ID,
		FileName: file.Name, IsDir: false, FileSize: committed,
		TargetFilePath: effectiveDest, Source: diskModel.FileAuditSourceWeb,
		Status: diskModel.FileAuditStatusSuccess,
	})
	return nil
}
```

补 import（若缺）：`"context"`、`"go.uber.org/zap"`、`"gorm.io/gorm"`、`common`、`diskModel`。

- [ ] **Step 5: 运行集成测试确认通过**

Run: `cd backend && go test ./service/disk/ -run TestExtractArchive -v`
Expected: PASS。若 `GetFileForPreviewWithPermission` 在 sqlite 下因字段缺失报错，按报错在测试中补全源文件记录字段后重跑。

- [ ] **Step 6: 写失败测试 — 失败回滚无孤儿（配额超额场景）**

追加测试（设用户配额为极小值，断言无 DB 记录、无物理文件）：

```go
func TestExtractArchive_RollbackOnQuotaExceeded(t *testing.T) {
	db := setupTestDB(t)
	root := t.TempDir()
	global.OPS_CONFIG = config.Server{System: config.System{OssType: "local"}, Local: config.Local{RootPath: root}}
	// 用户配额极小 → CheckQuota 拒绝。需让 QuotaService 在 sqlite 下返回 Allowed=false。
	// 若 CheckQuota 依赖 SysSetting/SysUser 配额行,在此插入相应行使剩余空间 < 解压大小。
	// (按 disk_quota.go 实际查询补种子数据;此处占位说明意图)
	db.Create(&diskModel.File{OPS_MODEL: global.OPS_MODEL{ID: 200}, UserID: 1, Name: "big.zip", Path: "/", IsFolder: false, Suffix: "zip", StorageType: "local", StoragePath: filepath.Join(root, "1", "big.zip")})
	srcDir := filepath.Join(root, "1")
	_ = os.MkdirAll(srcDir, 0o755)
	makeZip(t, filepath.Join(srcDir, "big.zip"))

	// 视 QuotaService 实现补种子后,断言:
	svc := &ArchiveService{}
	_ = svc.ExtractArchive("200", "/", true, 1, "COMMON")
	var n int64
	db.Model(&diskModel.File{}).Where("user_id = ? AND path LIKE ?", 1, "/big%").Count(&n)
	// 解压应被配额拦截,无入库记录
	if n != 0 {
		t.Errorf("配额超额时应无入库记录,实际 %d", n)
	}
}
```

> 注：本测试需配合 `QuotaService` 在 sqlite 下的真实查询补种子数据（配额行）。实现者按 `service/disk/disk_quota.go` 的查询逻辑补全种子，使 `CheckQuota` 返回 `Allowed=false`。若配额依赖无法在 sqlite 简单构造，则将该测试标记 `t.Skip("配额依赖需独立 fixture")` 并在计划备注，不阻塞主流程。

- [ ] **Step 7: 编译 + 全量测试 + vet**

Run: `cd backend && go build ./... && go test ./service/disk/ ./utils/ -race && go vet ./service/disk/...`
Expected: 全部 PASS，无 vet 警告。删除任何编译产物（`rm -f devopsAdmin`）。

- [ ] **Step 8: 提交**

```bash
cd backend
git add service/disk/disk_archive.go service/disk/disk_archive_test.go
git commit -m "feat(disk): 解压流水线(暂存→上传/移动→入库事务)+安全与配额防护"
```

---

## Task B4: 后端 — Swagger 重新生成 + 收尾

**Files:**
- Modify: `backend/docs/`（`swag init` 产物，已 gitignore，不提交）
- Verify: `backend/api/v1/disk/disk_archive.go` 注解完整

- [ ] **Step 1: 运行 swag init**

Run: `cd backend && swag init`
Expected: 成功生成 `docs/docs.go`/`swagger.json`/`swagger.yaml`；`POST /archive/extract` 的 body schema 为 `disk.ExtractArchiveRequest`（含 `destPath`/`intoSubfolder`）。

- [ ] **Step 2: 确认无未清理产物**

Run: `cd backend && git status`
Expected: 仅显示本次改动的源码；`docs/` 下 swagger 产物因 gitignore 不在待提交列表；无 `devopsAdmin` 二进制。

- [ ] **Step 3: 提交（若有源码注解微调）**

```bash
cd backend
git add -A  # 仅 api 注解若被微调;docs swagger 产物被 gitignore 不会进入
git commit -m "docs(disk): 同步解压接口 Swagger 注解" --allow-empty || echo "无源码改动,跳过"
```

---

## Task F1: 前端 — 重写 archive API + 类型

**Files:**
- Modify: `frontend/src/service/api/disk/archive.ts`

**Interfaces:**
- Produces: `ExtractArchiveParams { fileId: CommonType.IdType; destPath: string; intoSubfolder: boolean }`、`fetchExtractArchive(params: ExtractArchiveParams)`。删除旧 `(fileId, destFolderId)` 签名。

- [ ] **Step 1: 重写 archive.ts**

用下面内容替换 `frontend/src/service/api/disk/archive.ts` 的 `fetchExtractArchive`（38-45 行）与顶部 import：

```ts
import { request } from '@/service/request';
import type { CommonType } from '@/typings/common';

export interface ArchiveEntry {
  name: string;
  path: string;
  isFolder: boolean;
  suffix: string;
  size: number;
  children?: ArchiveEntry[];
}

/** 大体积归档文件（ISO等）列出内容可能耗时较长 */
const ARCHIVE_LIST_TIMEOUT = 2 * 60 * 1000;

/** 解压大归档文件耗时更长 */
const ARCHIVE_EXTRACT_TIMEOUT = 10 * 60 * 1000;

/** 列出归档文件顶层内容 */
export function fetchListArchive(fileId: string) {
  return request<ArchiveEntry[]>({
    url: `/archive/list/${fileId}`,
    method: 'get',
    timeout: ARCHIVE_LIST_TIMEOUT
  });
}

/** 列出归档内子目录内容 */
export function fetchListSubArchive(fileId: string, path: string) {
  return request<ArchiveEntry[]>({
    url: '/archive/list-sub',
    method: 'get',
    params: { fileId, path },
    timeout: ARCHIVE_LIST_TIMEOUT
  });
}

/** 解压归档参数（与后端 ExtractArchiveRequest 一一对应） */
export interface ExtractArchiveParams {
  fileId: CommonType.IdType;
  destPath: string;
  intoSubfolder: boolean;
}

/** 解压归档到目标目录 */
export function fetchExtractArchive(params: ExtractArchiveParams) {
  return request<void>({
    url: '/archive/extract',
    method: 'post',
    data: params,
    timeout: ARCHIVE_EXTRACT_TIMEOUT
  });
}
```

- [ ] **Step 2: typecheck**

Run: `cd frontend && pnpm typecheck`
Expected: 无错误（旧 `fetchExtractArchive(fileId, destFolderId)` 无调用方，删除不影响）。

- [ ] **Step 3: 提交**

```bash
cd frontend
git add src/service/api/disk/archive.ts
git commit -m "feat(disk): 解压 API 改为对象参数并新增 ExtractArchiveParams 类型"
```

---

## Task F2: 前端 — 国际化

**Files:**
- Modify: `frontend/src/locales/langs/zh-cn.ts`、`frontend/src/locales/langs/en-us.ts`

- [ ] **Step 1: 新增 page.disk.extract 键（zh-cn）**

在 `zh-cn.ts` 的 `page.disk` 下（与 `moveCopy` 同级，约 1551 行附近）插入：

```ts
      extract: {
        title: '解压到',
        currentDir: '当前目录',
        noFolders: '暂无子文件夹',
        targetLabel: '目标路径',
        extracting: '解压中，请稍候...',
        success: '解压成功',
        failed: '解压失败'
      },
```

- [ ] **Step 2: en-us 同步**

在 `en-us.ts` 对应位置插入：

```ts
      extract: {
        title: 'Extract to',
        currentDir: 'Current folder',
        noFolders: 'No subfolders',
        targetLabel: 'Target path',
        extracting: 'Extracting, please wait...',
        success: 'Extraction succeeded',
        failed: 'Extraction failed'
      },
```

- [ ] **Step 3: typecheck**

Run: `cd frontend && pnpm typecheck`
Expected: 无错误。

- [ ] **Step 4: 提交**

```bash
cd frontend
git add src/locales/langs/zh-cn.ts src/locales/langs/en-us.ts
git commit -m "feat(disk): 新增解压相关国际化(extract)"
```

---

## Task F3: 前端 — 新建目标目录选择器 extract-to-dialog.vue

**Files:**
- Create: `frontend/src/views/disk/modules/extract-to-dialog.vue`

**Interfaces:**
- Consumes: `fetchGetFolderList(path?)`（`src/service/api/disk/file.ts`）；`Api.Disk.FolderItem`；`$t`。
- Produces: 组件 `ExtractToDialog`，props `{ visible: boolean; fileName: string }`，emits `update:visible`、`confirm(destPath: string)`。

- [ ] **Step 1: 创建组件**

`frontend/src/views/disk/modules/extract-to-dialog.vue`：

```vue
<script setup lang="ts">
import { ref, watch } from 'vue';
import { $t } from '@/locales';
import { fetchGetFolderList } from '@/service/api/disk';

defineOptions({
  name: 'ExtractToDialog'
});

interface Props {
  visible: boolean;
  fileName: string;
}
interface Emits {
  (e: 'update:visible', value: boolean): void;
  (e: 'confirm', destPath: string): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const loading = ref(false);
const folders = ref<Api.Disk.FolderItem[]>([]);
const currentBrowsePath = ref('/');
const breadcrumb = ref<{ name: string; path: string }[]>([{ name: $t('page.disk.moveCopy.currentDir'), path: '/' }]);
const selectedPath = ref<string | null>(null);

watch(
  () => props.visible,
  visible => {
    if (visible) {
      currentBrowsePath.value = '/';
      breadcrumb.value = [{ name: $t('page.disk.moveCopy.currentDir'), path: '/' }];
      selectedPath.value = null;
      loadFolders('/');
    }
  }
);

async function loadFolders(path: string) {
  loading.value = true;
  currentBrowsePath.value = path;
  const { data, error } = await fetchGetFolderList(path);
  folders.value = !error && data ? data.list || [] : [];
  loading.value = false;
}

function enterFolder(folder: Api.Disk.FolderItem) {
  breadcrumb.value.push({ name: folder.name, path: folder.path });
  selectedPath.value = null;
  loadFolders(folder.path);
}

function clickBreadcrumb(index: number) {
  const target = breadcrumb.value[index];
  breadcrumb.value = breadcrumb.value.slice(0, index + 1);
  selectedPath.value = null;
  loadFolders(target.path);
}

function toggleSelect(path: string) {
  selectedPath.value = selectedPath.value === path ? null : path;
}

function handleConfirm() {
  const targetPath = selectedPath.value || currentBrowsePath.value;
  if (!targetPath) return;
  emit('confirm', targetPath);
}

function handleClose() {
  emit('update:visible', false);
}
</script>

<template>
  <NModal
    :show="visible"
    preset="card"
    :title="$t('page.disk.extract.title')"
    style="width: 90%; max-width: 560px"
    :mask-closable="false"
    :bordered="false"
    @update:show="handleClose"
  >
    <div class="flex flex-col gap-12px">
      <div class="text-14px">
        <span class="opacity-60">{{ $t('page.disk.moveCopy.sourceLabel') }}: </span>
        <span class="font-medium">{{ fileName }}</span>
      </div>

      <NBreadcrumb separator="/">
        <NBreadcrumbItem
          v-for="(item, index) in breadcrumb"
          :key="item.path"
          @click="clickBreadcrumb(index)"
        >
          <span :class="{ 'cursor-pointer hover:text-primary': index < breadcrumb.length - 1 }">
            {{ item.name }}
          </span>
        </NBreadcrumbItem>
      </NBreadcrumb>

      <div
        class="flex items-center gap-8px p-8px rounded cursor-pointer transition-colors"
        :class="selectedPath === currentBrowsePath ? 'bg-primary/10 text-primary' : 'hover:bg-gray-100 dark:hover:bg-gray-800'"
        @click="toggleSelect(currentBrowsePath)"
      >
        <SvgIcon icon="mdi:folder" :size="20" class="text-amber-500" />
        <span class="text-14px font-medium">. ({{ $t('page.disk.extract.currentDir') }})</span>
      </div>

      <NScrollbar style="max-height: 300px">
        <NSpin :show="loading">
          <div v-if="folders.length === 0 && !loading" class="py-24px text-center opacity-50">
            {{ $t('page.disk.extract.noFolders') }}
          </div>
          <div class="flex flex-col gap-4px">
            <div
              v-for="folder in folders"
              :key="folder.id"
              class="flex items-center gap-8px p-8px rounded cursor-pointer transition-colors"
              :class="selectedPath === folder.path ? 'bg-primary/10 text-primary' : 'hover:bg-gray-100 dark:hover:bg-gray-800'"
              @click="toggleSelect(folder.path)"
              @dblclick="enterFolder(folder)"
            >
              <SvgIcon icon="mdi:folder" :size="20" class="text-amber-500" />
              <span class="flex-1 text-14px">{{ folder.name }}</span>
              <NButton quaternary size="tiny" @click.stop="enterFolder(folder)">
                <template #icon>
                  <SvgIcon icon="mdi:chevron-right" :size="16" />
                </template>
              </NButton>
            </div>
          </div>
        </NSpin>
      </NScrollbar>

      <div v-if="selectedPath" class="text-13px opacity-70">
        {{ $t('page.disk.extract.targetLabel') }}: {{ selectedPath }}
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-8px">
        <NButton @click="handleClose">{{ $t('common.cancel') }}</NButton>
        <NButton
          type="primary"
          :disabled="!selectedPath && currentBrowsePath === '/'"
          @click="handleConfirm"
        >
          {{ $t('common.confirm') }}
        </NButton>
      </div>
    </template>
  </NModal>
</template>
```

- [ ] **Step 2: typecheck + lint**

Run: `cd frontend && pnpm typecheck && pnpm lint`
Expected: 无错误。

- [ ] **Step 3: 提交**

```bash
cd frontend
git add src/views/disk/modules/extract-to-dialog.vue
git commit -m "feat(disk): 新建解压目标目录选择器 extract-to-dialog"
```

---

## Task F4: 前端 — 接线 disk/index.vue（解压到当前目录 / 解压到...）

**Files:**
- Modify: `frontend/src/views/disk/index.vue`（import、处理函数、模板事件、新对话框）
- Modify: `frontend/src/components/disk/archive-action-dialog.vue`（loading 透传，可选）

**Interfaces:**
- Consumes: F1 `fetchExtractArchive`、F2 i18n、F3 `ExtractToDialog`；`diskStore.getCurrentPathString()`；`getFileList()`（本文件 86 行）。
- Produces: `@extract-here` / `@extract-to` 实际逻辑。

- [ ] **Step 1: 在 index.vue 增加 import 与状态**

在 `archive.ts` 相关 import 区追加：

```ts
import { fetchExtractArchive } from '@/service/api/disk/archive';
import ExtractToDialog from './modules/extract-to-dialog.vue';
```

在 preview 相关 ref 附近（`preview.showArchiveAction` 同区域）增加：

```ts
const extractLoading = ref(false);
const showExtractTo = ref(false);
```

（若 `ref` 已 import 则复用。）

- [ ] **Step 2: 增加解压处理函数**

在 `<script setup>` 内合适位置（`getFileList` 附近）增加：

```ts
async function runExtract(destPath: string, intoSubfolder: boolean) {
  const file = preview.archiveFile;
  if (!file) return;
  const fileId = file.fileId;
  if (fileId === undefined || fileId === null || fileId === '') return;

  extractLoading.value = true;
  window.$message?.loading($t('page.disk.extract.extracting'), { duration: 0 });
  const { error } = await fetchExtractArchive({ fileId, destPath, intoSubfolder });
  extractLoading.value = false;
  window.$message?.destroyAll?.();

  if (!error) {
    window.$message?.success($t('page.disk.extract.success'));
    preview.showArchiveAction = false;
    preview.showArchivePreview = false;
    showExtractTo.value = false;
    getFileList();
  } else {
    window.$message?.error($t('page.disk.extract.failed'));
  }
}

function handleExtractHere() {
  preview.showArchiveAction = false;
  runExtract(diskStore.getCurrentPathString(), true);
}

function handleExtractToOpen() {
  preview.showArchiveAction = false;
  showExtractTo.value = true;
}

function handleExtractToConfirm(destPath: string) {
  showExtractTo.value = false;
  runExtract(destPath, false);
}
```

- [ ] **Step 3: 改模板事件 + 挂载 ExtractToDialog**

替换 `disk/index.vue` 704-710 行的 `ArchiveActionDialog` 与事件为：

```vue
    <ArchiveActionDialog
      v-model:visible="preview.showArchiveAction"
      :file-name="preview.archiveFile?.fileName || preview.archiveFile?.name || ''"
      :extract-loading="extractLoading"
      @preview="preview.showArchivePreview = true; preview.showArchiveAction = false"
      @extract-here="handleExtractHere"
      @extract-to="handleExtractToOpen"
    />
    <ExtractToDialog
      v-model:visible="showExtractTo"
      :file-name="preview.archiveFile?.fileName || preview.archiveFile?.name || ''"
      @confirm="handleExtractToConfirm"
    />
```

- [ ] **Step 4: archive-action-dialog.vue 透传 loading/禁用**

修改 `frontend/src/components/disk/archive-action-dialog.vue`：Props 增加 `extractLoading?: boolean`；两个解压按钮加 `:loading="extractLoading"` 与 `:disabled="extractLoading"`：

```vue
interface Props {
  visible: boolean;
  fileName: string;
  extractLoading?: boolean;
}
```

模板内：

```vue
      <NButton :loading="extractLoading" :disabled="extractLoading" @click="emit('extractTo')">
        {{ $t('page.disk.contextMenu.extractTo') }}
      </NButton>
      <NButton :loading="extractLoading" :disabled="extractLoading" @click="emit('extractHere')">
        {{ $t('page.disk.contextMenu.extractHere') }}
      </NButton>
```

- [ ] **Step 5: typecheck + lint**

Run: `cd frontend && pnpm typecheck && pnpm lint`
Expected: 无错误。

- [ ] **Step 6: 手测清单**

- 上传一个 zip → 右键/预览弹窗 →「解压到当前目录」→ 出现以压缩包命名的子目录，内含文件。
- 「解压到...」→ 选择目录 → 确认 → 内容解压进所选目录。
- 同名再解压一次 → 自动重命名（`(1)`）。
- 大压缩包 → Loading 提示，按钮禁用。
- 解压后当前目录列表自动刷新。

- [ ] **Step 7: 提交**

```bash
cd frontend
git add src/views/disk/index.vue src/components/disk/archive-action-dialog.vue
git commit -m "feat(disk): 接线解压到当前目录/解压到...(含 loading 与列表刷新)"
```

---

## 收尾：跨模块一致性核对

- [ ] **前后端字段核对**：后端 `ExtractArchiveRequest{fileId, destPath, intoSubfolder}` ↔ 前端 `ExtractArchiveParams{fileId, destPath, intoSubfolder}`，JSON tag 一致。
- [ ] **后端**：`cd backend && go build ./... && go test ./service/disk/ ./utils/ -race && go vet ./...`；删除二进制。
- [ ] **前端**：`cd frontend && pnpm typecheck && pnpm lint`。
- [ ] **根仓库**：按团队工作流同步前后端子模块指针（main 聚合快照），不在根仓走 feature→main PR。
