# 网盘模块·文件 CRUD 后端（第2期）

> 网盘整体设计 `docs/disk-upload-design.md`；阶段0 地基见 `disk-backend-foundation.md`；迁移进度见 AI 协作记忆 `disk-migration-progress`。
> 本条记录第2期 文件 CRUD 的**前后端**落地（后端 + 前端 UI 均已完成，go build/vet/gofmt + vue-tsc 通过，未运行时验证）。

## 范围

文件/目录 CRUD：新建文件夹 / 重命名 / 移动 / 复制 / 删除（→回收站）。
**不含**：restore + 回收站列表（属第6期）；上传（第3期）。

## 接口（均挂专用 DiskGroup：JWTAuth + OperationRecord）

| 方法 | 路径 | 用途 |
|---|---|---|
| POST | /file-meta/mkdir | 新建文件夹 |
| POST | /file-meta/rename | 重命名（文件夹含后代 path 级联）|
| PUT  | /file-meta/move | 移动（批量；循环校验；文件夹级联后代 path）|
| POST | /file-meta/copy | 复制（文件夹递归深拷贝子树）|
| POST | /file-meta/delete | 删除→回收站（文件夹级联后代 status=2）|

请求体：`model/disk/request` 的 `MkdirReq/RenameReq/MoveReq/CopyReq/DeleteReq`。FileIds 用 `[]string`（对齐前端 IdType），service 内 `parseFileIds` 转 `[]int64`。userId 一律 JWT 取（防 IDOR）。

## 关键语义（适配本项目模型：path=自身全路径 + parent_id 树 + status 回收站 + ref_count）

- **路径级联**（rename/move 文件夹）：事务内一条 SQL 重写整棵子树 path ——
  `UPDATE disk_files SET path = CONCAT(新前缀, SUBSTR(path, 旧长度+1)) WHERE user_id=? AND path LIKE 旧前缀/% AND deleted_at IS NULL`
  （借鉴 remote，适配"path=自身全路径"；raw Exec 需手加 `deleted_at IS NULL`，GORM 自动条件不作用于 Exec）。
- **同名冲突**：统一**报错**（非自动改名），简洁一致；mkdir/rename/move/copy 均查同级 `status=1` 同名。
- **删除→回收站**：`status=2`（保留行，不 GORM 软删除）；文件夹级联后代 `WHERE path LIKE 前缀/%` 一并 status=2。无独立 `disk_trash` 表（比 remote 简单，因不动物理存储）。
- **复制**：纯元数据复制（新 file_id）；文件夹按 path 升序递归深拷贝，`newIdByNewPath` 映射回填后代 parent_id。依赖不变式：status=1 后代其祖先必 status=1（因删除级联）。
- **循环校验**（move）：目标不能是自身或自身后代（`TargetPath==srcPath || HasPrefix(TargetPath, srcPath+"/")`）。
- **第2期纯元数据**：当前无真实上传文件（storage_path 空）；物理存储重命名/ref_count 共享留 TODO 给第3期上传落地。
- **GORM finisher 污染规避**：所有读（Take/Count/Find）用 fresh `global.OPS_DB.WithContext(ctx)`，事务 `tx` 只承载写（Updates/Exec/Create）。

## 事务

rename/move/copy/delete 均包在 `global.OPS_DB.Transaction` 内；批量 move/copy/delete 在单事务内循环。无行级锁（同级同名完整性靠先查后插，理论有并发竞态；未来可加 `(user_id,parent_id,name)` 部分唯一索引 `WHERE deleted_at IS NULL AND status=1` 加固）。

## 验证

`go build ./...` / `go vet` / `gofmt -l` 全绿。**未运行时验证**（需重启后端注册新路由；逻辑正确性待前端 CRUD UI 浏览器点测或 Go 集成测试）。

## 前端 UI（A1+A2，vue-tsc 通过）

- 工具栏「新建文件夹」按钮（toolbar.vue）。
- 文件项 `⋯` 操作菜单（hover 显示，file-card.vue / file-list.vue）：重命名 / 移动 / 复制 / 删除；事件冒泡 file-grid→index。
- `name-input-modal`：新建文件夹 + 重命名共用（index.vue 内联 NModal）。
- `move-copy-modal.vue`：NTree 目录树选择目标（前端前置「根目录」节点 key='/'，子节点 key=path，选中即 targetPath）；onMounted watch visible 拉 `GET /file-meta/folder-tree`。
- 删除：`useDialog.warning` 二次确认；操作后统一 `getFileList()` 刷新。
- i18n：`page.disk.action/modal/msg` 三处同步（zh-cn / en-us / app.d.ts schema）。
- 单文件操作（从 ⋯ 菜单）；批量（多选）未做。

## 后续

- 第3期上传落地后，rename/move/copy/delete 需补物理存储处理（storage_path 重命名/复制、ref_count 增减、彻底删时 DeleteObject）。
- 运行时浏览器点测（需重启后端注册 CRUD + folder-tree 路由）。
- 批量移动/复制（基于 selectedFiles）、回收站 restore/列表（第6期）。
