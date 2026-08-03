# 网盘模块·上传后端（第3期 3-1）

> 整体设计 `docs/disk-upload-design.md`；迁移进度见 AI 协作记忆 `disk-migration-progress`；阶段0/第2期见 `disk-backend-foundation.md`/`disk-crud.md`。
> 本条记录第3期上传体系**后端**落地（前端 uploader-engine 待做）。

## 数据模型（建表注册在 `initialize/gorm.go` RegisterTables，用户偏好的惯例）

- `disk_upload_sessions`：上传会话（`DiskUploadSession`，OPS_AUDIT_MODEL + UploadId PK）。字段：user_id/identifier(=quickHash)/file_name/relative_path/total_size/chunk_size/total_chunks/current_directory/status(uploading/merging/completed/failed)/quick_hash/strong_hash/merged_md5/file_id/storage_key。identifier+user 作会话去重键。
- `disk_upload_chunks`：分片记录（`DiskUploadChunk`），唯一索引 (upload_id, chunk_number)，`ON CONFLICT DO NOTHING` 幂等收片。0-based chunkNumber（与 utils 一致）。

## 分片物理存储

`utils/upload/disk_chunk.go`：`DiskChunkDir(sessionID)=chunkRoot()/disk/{id}`（与 media 隔离）；`SaveDiskChunk`/`MergeDiskChunks`(流式合并边算 MD5)/`RemoveDiskUploadDir`。复用 chunk.go 的 chunkRoot()。

## 接口（挂专用 DiskGroup：JWTAuth + OperationRecord）

| 方法 | 路径 | 用途 |
|---|---|---|
| GET | /file-meta/upload | 秒传/续传检测 |
| POST | /file-meta/upload | 上传分片（multipart：uploadId/chunkNumber/chunkHash/file）|
| POST | /file-meta/merge | 合并（原子抢占→流式合并算 MD5→OSS 推送→建 disk_files→清理）|
| DELETE | /file-meta/upload | 取消并清理 |

## 语义（镜像 media_upload.go，适配 disk）

- **秒传**：同用户已有同 `quick_hash+strong_hash` 的正常 disk_files → `pass=true`（同用户场景；跨用户实测复用留细化）。
- **续传**：DB 会话+分片表，返回 `resume[]`（已收分片序号）；多实例/重启可恢复（D2，不学 remote/jmal 纯内存）。
- **分片校验**：每片算 MD5 与 chunkHash 比对，不符拒绝；幂等收片。
- **合并**：原子抢占 `uploading→merging`（防并发双合并）→ `MergeDiskChunks` 流式合并算完整 MD5 → `BuildFileHeader`+`NewOss().UploadFile` 推 OSS（**local/rustfs 由 System.OssType 配置驱动，3-1 即覆盖两种**）→ 建 disk_files（md5=合并 MD5、quick_hash/strong_hash、storage_type=OssType、storage_path=key、parent 由 currentDirectory 解析）→ 回填会话 + 清分片。
- **文件夹上传**：`resolveFolderForUpload` 按 relativePath 段懒建子目录（前端文件夹上传时透传 relativePath）。
- 文件 ContentType 未嗅探（留 TODO，可从首片 magic bytes）。

## 复用与隔离

- 复用 `utils/upload`（OSS 抽象 8 种存储、BuildFileHeader）、media_upload.go 的范式。
- 分片目录与 media 隔离（`/disk/` 前缀）；disk 与 media 是独立功能（D1）。

## 验证

`go build`/`go vet`/`gofmt` 全绿。**未运行时验证**（需前端 uploader-engine 计算 quickHash/strongHash + 分片上传才能端到端）。

## 后续

- **前端 3-3**：uploader-engine（spark-md5 Web Worker 算 quickHash+strongHash）+ chunk-manager（动态分片/并发/重试）+ transfer-panel + toolbar 上传入口；与后端 check/upload/merge 对接。
- 配额（sys_user.take_up_space 原子预占/对账）、跨用户秒传实测（首尾采样常量时间比对）、ContentType 嗅探、SSE 合并进度。
