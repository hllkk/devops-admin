# 网盘模块·后端地基（阶段0：目录树 + 只读列表真实后端）

> 网盘上传功能整体设计见 `docs/disk-upload-design.md`；迁移分期与进度见 AI 协作记忆 `disk-migration-progress`。
> 本条记录阶段0 落地的后端地基（目录树模型 + `/file-meta` 只读接口 + 鉴权接线）。

## 背景

网盘模块前端第1期（只读文件列表）此前走 `USE_MOCK=true` 假数据，后端 `/file-meta/*` 为零。
阶段0 补齐**目录树地基**（既是第1/2 期列表/CRUD 的基础，也是第3 期上传的前置依赖），并让前端列表接到真实后端。

## 落地内容（2026-07-30）

**数据模型** `model/disk/disk_file.go`：`DiskFile`（表 `disk_files`），基座 `OPS_AUDIT_MODEL` + 业务主键 `FileId`（DB 自增）。
- 目录树字段：`UserId/ParentId/Name/IsDirectory/Path/Size/ExtendName/ContentType/IsFavorite/Status`
- 第3 期上传预留字段（一次性建齐，免后续加列）：`Md5/QuickHash/StrongHash/StorageType/StoragePath/RefCount`
- 索引：`user_id/parent_id/path/md5/quick_hash`

**四层切片**（对齐项目 `Router→API→Service→Model` + `enter.go` 聚合）：
- `service/disk/disk_file.go`：`GetFileList`（按 JWT `userId` 隔离 + `parent_id` 列直接子节点 + 文件夹优先排序 + `queryType`/`keyword` 过滤 + `PageInfo.LimitOffset` 分页）、`ResolvePath`（路径段单查还原面包屑）
- `api/v1/disk/disk_file.go`、`router/disk/disk_file.go`（`/file-meta/list`、`/file-meta/path-resolve`）
- 三个 `enter.go` 接线：`service`(`DiskServiceGroup`) / `api/v1`(`DiskApiGroup`) / `router`(`Disk`)

**建表**：`initialize/gorm_biz.go` 的 `bizModel()` 加 `disk.DiskFile{}`（业务表落点，`RegisterTables()` 会调用）。

## 关键决策

- **鉴权模型**：网盘 `/file-meta/*` 是**个人自有数据**操作，挂在 `initialize/router.go` 新建的**专用 `diskGroup`**（仅 `JWTAuth + OperationRecord`），**不挂 PrivateGroup 的 Casbin/DataScope**——角色级门控与部门数据权限对个人文件不适用；`service` 按 JWT `userId` 强制隔离防 IDOR。
- **userId 取法**：`utils.GetUserID(c)`（JWT），不信前端传参。
- **响应契约**：list 返回 jmal 风格 `{list,total,page,size}` + `BackendFileItem` 字段（第1 期前端契约已定型、vue-tsc 已过），非项目通用 `{rows,total,pageNum,pageSize}`；Swagger 已标注实际形状。
- **请求参数**：`page`→`pageNum` 对齐 `request.PageInfo` 绑定。

## 验证

静态验证通过：`go build ./...` / `go vet` / `gofmt -l` 全绿，前端 `vue-tsc --noEmit` EXIT=0。
**未做运行时验证**（需起 DB/Redis 栈登录打 `/file-meta/list`）；空库下列表返回空属正常。

## 后续

- 第2 期：文件/目录 CRUD（新建/重命名/移动/复制/删除→回收站）
- 第3 期：上传体系（分片/秒传/断点续传/文件夹/大文件），设计见 `docs/disk-upload-design.md`
- `/disk/quota` 后端待补（需 `sys_user` 配额字段），`quota.ts` 仍 MOCK
