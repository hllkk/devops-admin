# 网盘文件上传功能 设计方案

> 范围：网盘模块的文件上传体系（秒传 / 断点续传 / 文件夹上传 / 大文件上传）。
> 状态：设计稿（2026-07-30）。对应网盘迁移**第 3 期**，前置依赖**目录树模型**（第 1/2 期基础，当前缺）。
> 蓝本：`/home/remote/devops-admin`（前后端同栈，主参考）；`/home/remote/jmal-cloud`（功能对照 + 避坑）。

---

## 1. 已拍板的决策

| # | 决策 | 选择 |
|---|---|---|
| D1 | 后端底座 | **新建 `service/disk`**，照搬 remote `disk_file.go`（目录树 + 上传合一）；现有 `service/media/media_upload.go` 作为独立"媒体库"功能保留不动 |
| D2 | 续传状态 | **DB 会话表 + 分片表**（原子状态机），不学 remote 的 Ristretto 纯内存 / jmal 的 Caffeine 纯内存 |
| D3 | 上传链路 | **后端收片模式**（浏览器→后端分片→后端合并→`PutObject`/`ComposeObject` 入 RustFS）；S3 预签名直传列为未来优化 |
| D4 | 下载链路 | 已定（见 `rustfs-public-download-before-netdisk`）：私有文件走**后端代理下载 + Range 透传**，不走 `/oss/` 公开反代 |
| D5 | 秒传指纹 | **真实内容哈希**：quickHash(采样 MD5) + strongHash(采样 SHA-256) 快速初筛，合并时算**完整 MD5** 入库作权威指纹；绝不学 jmal 拼接串 |

---

## 2. 总体架构

```
浏览器                                  devops-admin 后端(Go)                    RustFS(S3兼容)
─────────                              ─────────────────────                   ──────────────
uploader-engine.ts                      router/disk  →  api/v1/disk  →  service/disk
 ├ hash-worker(WebWorker:MD5/SHA256)      ├ /file-meta/upload-config             │
 ├ chunk-manager(动态分片)                ├ /file-meta/upload       (GET check)  │
 ├ instant-check(秒传/续传判定)           ├ /file-meta/upload       (POST 分片)  │  PutObject(file/xxx)
 └ use-uploader(任务/并发/重试)           ├ /file-meta/merge        (合并)       │  ComposeObject(分片→整文件)
                                          ├ /file-meta/upload       (DELETE 取消)│
transfer-panel.vue(传输面板)              ├ /file-meta/verify-cross-user(跨用户秒传实测)
toolbar.vue(上传入口)                     ├ /file-meta/list /path-resolve (目录树, 第1/2期)
                                          └ /disk/quota /quota/check            │
                                                                                 ↑
                                          global.OPS_DB(disk_files/sessions/chunks)
                                          global.OPS_CACHE(累计上传限流计数)
                                          global.OPS_REDIS(SSE hub / 可选)
                                          utils/upload.NewOss()(local / rustfs[minio])
                                          middleware: OperationRecord / UploadSizeLimit
                                          SSE: merge_progress 推送(复用 notice-sse)
```

四层同域（遵循项目约定）：`web/src/views/_disk` ↔ `web/src/service/api/disk` ↔ `web/src/store/modules/disk` ↔ `web/src/typings/api/disk.api.d.ts`；i18n 三处同步（`zh-cn` / `en-us` / `app.d.ts` schema）。

---

## 3. 数据模型（表结构，基于 remote 蓝本 + 项目命名约定推导）

> 命名沿用现有 media 表风格（`media_uploads` → `disk_*`）。最终字段以实现时核对 remote `model/disk/disk_file.go` 为准。

### 3.1 `disk_files`（目录树节点 —— 第 1/2 期基础，本设计一并定义）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigint PK | 雪花 ID |
| parent_id | bigint | 父目录 ID，根=0 |
| user_id | bigint | 所属用户 |
| name | varchar(255) | 文件/文件夹名 |
| is_directory | bool | 是否目录 |
| path | varchar(512) | 全路径（如 `/docs/a.txt`），用于面包屑/路径解析 |
| size | bigint | 字节数（目录=0 或子树聚合） |
| md5 | char(32) | 完整内容 MD5（合并后算，权威指纹） |
| quick_hash | char(32) | 采样 MD5（首尾 2MB + size），秒传初筛 |
| strong_hash | varchar(64) | 采样 SHA-256，防碰撞 |
| storage_type | varchar(16) | `local` / `rustfs` |
| storage_path | varchar(512) | 对象 key / 本地相对路径（网盘统一 `file/{userId}/{...}`） |
| ref_count | int | 物理引用计数（跨用户秒传复用物理对象，删到 0 才删 OSS） |
| status | tinyint | 正常 / 回收站 / 删除中 |
| created_at / updated_at / deleted_at | timestamp | GORM 软删除 |

索引：`(user_id, parent_id, name)`、`(quick_hash, strong_hash)`（跨用户秒传查）、`md5`、`path`。

### 3.2 `disk_upload_sessions`（断点续传会话 —— D2 核心，替代纯内存缓存）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigint PK | |
| identifier | char(32) | = quickHash，上传会话标识 |
| user_id | bigint | |
| file_name | varchar(255) | |
| relative_path | varchar(512) | 文件夹上传的相对路径 |
| total_size | bigint | |
| total_chunks | int | |
| chunk_size | bigint | |
| current_directory | varchar(512) | 目标父目录 |
| status | varchar(16) | `uploading` / `merging` / `completed` / `failed` |
| quick_hash / strong_hash | char(32)/varchar(64) | |
| merged_md5 | char(32) | 合并后实测（权威） |
| storage_path | varchar(512) | 合并落点 |
| created_at / updated_at | timestamp | |

唯一约束：`(identifier, user_id)`。状态机 `uploading→merging→completed/failed` 用条件 UPDATE 原子抢占（复用 `media_upload.go` 已验证模式）。**重启不丢、多实例共享**。

### 3.3 `disk_upload_chunks`（分片记录）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigint PK | |
| session_id | bigint FK | |
| chunk_number | int | 1-based |
| chunk_hash | char(32) | 单片 MD5（可选校验） |
| size | bigint | |
| created_at | timestamp | |

唯一索引 `idx_upload_chunk(session_id, chunk_number)`，`ON CONFLICT DO NOTHING` 幂等收片（复用 media 模式）。续传时 `SELECT chunk_number WHERE session_id=?` 即得已传列表。

### 3.4 配额

复用 `sys_user.take_up_space` 字段（remote 同款）。原子预占：`UPDATE sys_user SET take_up_space = take_up_space + ? WHERE id=? AND take_up_space + ? <= quota`（防 TOCTOU）。SUPER/ADMIN 不限。

---

## 4. 后端分层与接口清单

### 4.1 新增/接线点（本项目缺失项）

- `server/model/disk/`：`disk_file.go`、`disk_upload_session.go`、`disk_upload_chunk.go`、`request/*.go`
- `server/router/disk/enter.go`：实现 `InitDiskFileRouter`（当前 `media/enter.go` 是空壳的反面教材，别照抄）
- `server/api/v1/disk/`：`disk_file.go`（handler 层，本项目 `api/v1/` 下目前只有 `system/`）
- `server/service/disk/`：`disk_file.go`（上传/秒传/合并/续传）、`disk_quota.go`、`disk_provider.go`（存储类型分发）
- **接线**：
  - `server/initialize/gorm.go` AutoMigrate 注册三张表（现有 media 四张表在此注册，照加）
  - `server/service/enter.go` `ServiceGroup` 挂 `DiskServiceGroup`（当前只挂了 System）
  - `server/initialize/router.go` 注册 `InitDiskFileRouter`（media 路由被注释的反例，别重蹈）
  - 中间件：`OperationRecord()`（审计）+ `UploadSizeLimitMiddleware()`（单分片上限），挂 `/file-meta` 组

### 4.2 接口清单

| 方法 | 路径 | 用途 |
|---|---|---|
| GET | `/file-meta/upload-config` | 返回分片策略（chunkSize 阈值、并发、限制），前端可覆盖默认 |
| GET | `/file-meta/upload` | 秒传/续传检测：入参 identifier/quickHash/strongHash/fileName/totalSize/relativePath/currentDirectory；出参 `pass`(秒传)/`resume[]`(已传分片)/`merge`(全到齐可直合并)/`crossUserVerify`(跨用户待实测) |
| POST | `/file-meta/upload` | 收分片/整文件（multipart，含 chunkNumber/totalChunks/chunkHash）；整文件=chunkNumber 0/totalChunks 1 |
| POST | `/file-meta/verify-cross-user-instant-upload` | 跨用户秒传：前端上传首尾 2MB 采样，后端实测哈希常量时间比对 |
| POST | `/file-meta/merge` | 合并：原子抢占 `uploading→merging` → 流式合并算完整 MD5 → `PutObject`/`ComposeObject` → 建 `disk_files` 记录 → 扣配额对账 → 清分片；SSE 推 `merge_progress` |
| DELETE | `/file-meta/upload` | 取消并清理分片/会话 |
| POST | `/file-meta/check-chunks` | （预留）批量分片去重检查，按 chunkHash 跳过已存在 |
| POST | `/disk/quota/check` | 配额预检 |
| GET | `/file-meta/list`、`/file-meta/path-resolve`、`/disk/quota` | 目录树/配额（第 1/2 期，前端 MOCK 已对接） |

---

## 5. 前端模块拆分（照搬 remote `hooks/business/upload/`）

### 5.1 上传引擎（新增，约 6 文件）

- `web/src/hooks/business/upload/uploader-engine.ts`：任务状态机 `pending→hashing/checking→uploading→merging→completed/paused/failed`；文件级并发（默认 3）+ 分片级并发（默认 6，设备感知降级）；指数退避重试（分片 3 次/合并 2 次）；AbortController 取消
- `chunk-manager.ts`：动态分片（<100M→10M / <1G→20M / <5G→50M / ≥5G→100M），设备内存自适应阈值
- `hash-worker.ts`：Web Worker 算 quickHash/strongHash/fullMD5/chunkHash（spark-md5 + crypto.subtle）
- `instant-check.ts`：Worker 调度单例
- `use-uploader.ts`：业务封装（触发上传/文件夹、冲突弹窗、store 同步）
- 类型：`web/src/typings/upload.d.ts`

### 5.2 视图与状态

- `web/src/views/_disk/disk/modules/transfer-panel.vue`：传输面板（任务列表 + 暂停/继续/取消/重试 + 文件夹聚合）
- `web/src/views/_disk/disk/modules/toolbar.vue`：**新增上传入口**（选文件 / 选文件夹，webkitdirectory）
- `web/src/store/modules/disk/index.ts`：加 `transferList` 状态
- `web/src/service/api/disk/upload.ts`：**新建**（勿复用指向不存在后端的 `/resource/oss/upload`）；含 fetchUploadConfig/fetchCheckFile/fetchUploadChunk/fetchMergeChunks/fetchCancelUploadChunks/fetchVerifyCrossUserInstantUpload
- `web/src/typings/api/disk.api.d.ts`：补上传相关类型
- i18n：`page.disk` 上传文案三处同步

### 5.3 依赖

`web/package.json` 新增 `spark-md5`（SHA-256 用浏览器原生 crypto.subtle，无需额外依赖）。

---

## 6. 四功能实现要点

### 6.1 秒传（真实内容哈希）
- 前端 Worker 算 quickHash = `MD5(首2MB + size[8字节小端] + 尾2MB)` + strongHash = `SHA-256(同采样缓冲)`
- `GET /file-meta/upload` 初筛：
  - **同用户命中**（`user_id+path+quick_hash+name` 且 storage_path 非空）→ `pass=true` 直接完成
  - **跨用户命中**（全局 `quick_hash+strong_hash`）→ 返回 `crossUserVerify`，前端传首尾 2MB 实测，后端 `subtle.ConstantTimeCompare` 比对通过才复用物理对象（`ref_count++`，信任锚在服务端）
- 合并时算完整 MD5 入库，作权威指纹

### 6.2 断点续传（DB 会话 + 分片表，D2）
- 首次上传：建 `disk_upload_sessions`(status=uploading)，收片幂等写 `disk_upload_chunks`
- 续传：`GET /file-meta/upload` → 后端 `SELECT chunk_number FROM disk_upload_chunks WHERE session_id=?` 返回 `resume[]`；前端只传缺失分片
- 合并：条件 UPDATE `status uploading→merging` 原子抢占（防并发双合并）；流式合并算 MD5 → 落 OSS → 建 disk_files → 清 chunks/session
- **页面刷新**：前端凭 quickHash 重查会话即可恢复（remote 纯内存做不到，本项目 DB 做得到 ✅）

### 6.3 文件夹上传（修 remote 空目录丢失）
- `webkitdirectory` + `directory` 属性取 `webkitRelativePath`，全程透传 `relative_path`
- 后端 merge 阶段按 relative_path 懒建目录节点（`ensureFolderRecords`）
- **修 remote 缺陷**：不跳过 size=0 条目中"目录语义"的项；或上传前显式发送目录结构（借鉴 jmal `upload-folder` 先建目录），保留空文件夹

### 6.4 大文件上传（hash-while-upload）
- **跳过上传前整文件 MD5 预算**：直接用 quickHash 当会话 identifier 开传，每个分片单独算 chunkHash 校验，**合并时后端算完整 MD5**
- 动态分片 + 设备感知并发 + 指数退避（1s/2s/4s + jitter）；4xx（除 408/429）快速失败
- 合并进度经 SSE 推 `merge_progress`（复用现有 notice-sse hub）

---

## 7. 上传时序

```
选文件/文件夹 → 建 task
  → Worker 算 quickHash+strongHash（<100ms）
  → GET /file-meta/upload (check)
       ├ pass=true            → 直接完成（同用户秒传）
       ├ crossUserVerify      → POST 首尾2MB实测 → 通过则复用（跨用户秒传）
       ├ merge=true           → POST /file-meta/merge（全到齐）
       └ resume=[已传分片]    → 剔除已传，并发传缺失分片
  → 循环 POST /file-meta/upload（分片，幂等）
  → 全到齐 → POST /file-meta/merge
       后端: 原子抢占 merging → 流式合并算MD5 → PutObject/ComposeObject
            → 建 disk_files → 扣配额对账(实测size修正) → 清分片 → SSE merge_progress
  → completed
```

---

## 8. 存储与安全

- **RustFS 落点**：网盘统一 `file/{userId}/{...}` 前缀；生产已收敛为 `uploads/` 公开、其余私有（见 `rustfs-public-download-before-netdisk`）。`file/` 天然私有 ✅
- **下载**：后端代理 `GetObject` + Range 透传鉴权（D4），不走 `/oss/` 公开反代
- **配额对账**：原子条件 UPDATE 预占；落库后 `os.Stat` 实测修正客户端自报 size（低报补扣/高报退还）
- **完整性校验**：合并算 MD5 与 identifier/quickHash 比对，不符删文件报错（避 jmal 静默写坏文件）
- **防路径穿越**：identifier 强制 32 位十六进制；文件名/路径 `SanitizePathComponents`；magic bytes 嗅探防伪装
- **文件名编码**：下载头用 RFC5987 `filename*=UTF-8''<percent-encoded>`（避 jmal ISO-8859-1 乱码）
- **限流**：累计上传大小用 `OPS_CACHE` 计数（key `disk:upload_bytes:{userId}:{identifier}`，TTL 24h）

---

## 9. 实施步骤（分阶段）

> 注：上传是第 3 期，但**目录树模型 `disk_files` 是第 1/2 期的基础**，须先落地。建议顺序：

- **阶段 0｜目录树地基**（第 1/2 期前置）：`disk_files` 模型 + AutoMigrate + `/file-meta/list`、`/path-resolve`、`/disk/quota` 后端实现 → 前端 `USE_MOCK=false` 切真实接口。**这是上传的前置**。
- **阶段 1｜后端上传骨架**：sessions/chunks 表 + service/disk 上传链路（check/upload/merge/cancel）+ handler + 路由注册 + 中间件。本地存储先跑通。
- **阶段 2｜RustFS 接入**：storage_type=rustfs 分支（PutObject/ComposeObject），`file/` 前缀落点，代理下载。
- **阶段 3｜前端引擎**：移植 uploader-engine/chunk-manager/hash-worker/use-uploader + transfer-panel + toolbar 入口 + api/upload.ts + 依赖 spark-md5。
- **阶段 4｜秒传/跨用户/配额对账**：双指纹 + 跨用户实测 + 配额三本账。
- **阶段 5｜文件夹与体验**：webkitdirectory + 空目录修复 + SSE 合并进度 + 重试调优。
- 每阶段结束跑 `vue-tsc`（前端）+ `go build`（后端）验证。

---

## 10. 风险与开放问题

| 风险 | 应对 |
|---|---|
| remote 后端 `disk_file.go` 5163 行，移植量大 | 分阶段、按本方案接口边界裁剪，不照搬全量；先 local 后 rustfs |
| DB 续传每分片一次写 | 已有幂等唯一索引，可接受；超大文件分片大(50-100M)则写次数少 |
| 跨用户秒传本地存储需真复制（ref_count + 物理空间） | remote 明确牺牲去重换配额正确；本项目 RustFS 用同对象多引用(`ref_count`)可省物理空间，待实现时定 |
| spark-md5 大文件全量 MD5 仍慢 | hash-while-upload 已规避（只在合并后端算） |
| 与既有 `media_upload.go` 职责边界 | D1 已定：media=独立媒体库，disk=网盘目录树，互不合并 |
| 表字段为推导值 | 实现阶段核对 remote `model/disk/disk_file.go` 后定稿 |

---

## 11. 验收清单

- [ ] 单文件上传（小文件整传 + 大文件分片）成功落 RustFS `file/` 前缀
- [ ] 秒传：同用户重复上传即时完成；跨用户上传相同文件经实测后复用
- [ ] 断点续传：上传中断（关页/断网）后重选同文件，仅传缺失分片；服务重启后续传仍可恢复
- [ ] 文件夹上传：目录结构完整保留，含空文件夹
- [ ] 配额：超限拒绝；落库 size 与实测一致
- [ ] 完整性：分片损坏/缺失被检测，不产坏文件
- [ ] 下载：私有文件走后端代理 + Range，非 `/oss/` 公开
- [ ] 前端 `vue-tsc` 通过、i18n 三处同步、`USE_MOCK=false` 切真实接口
