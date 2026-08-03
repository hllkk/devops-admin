# 网盘上传链路 vs jmal-cloud：机制深度对比

> 范围：聚焦「上传」链路（分片 / 秒传 / 续传 / 合并 / 配额 / 前端引擎），做**机制级**深度对比，不是功能有无清单。
> 对照对象：当前 `server/service/disk/disk_upload.go` + `web/src/hooks/business/disk/use-disk-upload.ts`（含 `utils/disk-hash*.ts`、`utils/upload/disk_chunk.go`、`service/disk/disk_quota.go`）vs `/home/remote/jmal-cloud`（前端 `jmal-cloud-view` / 后端 `jmal-cloud-server`）。
> 依据：2026-08-03 双边代码实读核对。当前代码已核至 HEAD（devops 分支）；jmal 由 subagent 带 file:line 复核，**纠正了旧记忆 `netdisk-upload-download-jmal-analysis` 的若干细节**（见 §2.1 注）。
> 关系：[`disk-vs-jmal-gap.md`](./disk-vs-jmal-gap.md) 是全功能面矩阵（功能有无 + P0/P1/P2 分级），本文是其「维度 4 上传」的机制展开，供后续上传专项优化（边传边合并 / OSS Presigned / SSE 进度等）做设计输入。
> 定位（判「必要 vs 增强」以此为准）：企业内部网盘，管理员上传大型安装包 / 交付物（含 9GB+ 系统镜像 ISO），对外链接分享。

---

## 1. 当前上传完整链路

### 1.1 前端（`web/src/hooks/business/disk/use-disk-upload.ts` + `utils/disk-hash*.ts`）

模块级单例引擎，toolbar 与 transfer-panel 共享同一份 `tasks`：

```
uploadFiles(files, currentDirectory, dirs?)
  ├─ ensureConcurrentConfig()        读 sys_disk_config（并发/重试），仅加载一次
  ├─ 文件夹：fetchEnsureFolders()    预建空目录（webkitEntries 递归，best-effort）
  ├─ 按 webkitRelativePath 顶层段分组 → 文件夹聚合条目(isFolderAgg)
  └─ enqueueUpload() → 文件级并发池(maxConcurrentUploads) 受控调度
        └─ runUpload()
             ├─ computeSampleHashes()  Web Worker 池算 quickHash+strongHash+midHash
             ├─ fetchCheckUpload()     → 秒传(pass) / 续传(resume[]) / 新传
             ├─ pass → fetchMergeUpload(instant=true)   不传分片直接落库
             ├─ 否则：分片级并发池(min4, max(CPU-2,4)) 上传缺失分片
             │        ├─ chunkMd5() 走 Worker 池（自增 id 路由，并发安全）
             │        ├─ 指数退避 + jitter 重试 N 次（CancelledSentinel 可打断）
             │        └─ 字节级进度(chunkLoaded Map) + EMA 速度(α=0.25) + 100ms 节流
             └─ fetchMergeUpload()  合并
  控制：pause / resume / retry / cancel / pauseAll / resumeAll + beforeunload 防误关
```

哈希算法（`utils/disk-hash.ts`）：`quickHash=MD5(首2MB+size+尾2MB)`、`strongHash=SHA-256 同采样`、`midHash=中间2MB MD5（>4MB 文件）`。分片大小 `getChunkSize` 按文件大小分级（10/20/50MB）再按 `navigator.deviceMemory` 缩放（<2GB 降一档）。哈希与分片 MD5 全部调度到 Worker 池（`hardwareConcurrency` 上限 8，轮询 + 自增 id 关联请求/响应，防并发错乱）。

### 1.2 后端（`server/service/disk/disk_upload.go` + `utils/upload/disk_chunk.go` + `service/disk/disk_quota.go`）

| 接口 | 方法 | 机制要点 |
|---|---|---|
| `GET /file-meta/upload` | `Check` (`disk_upload.go:60`) | 安全上限校验（10000 片 / 64MB / 配置上限）→ 配额预检 → **三重哈希秒传**（quickHash+strongHash 命中 + midHash 二次校验，不符降级走正常上传）→ DB 取/建会话 → 返回 `resume[]` |
| `POST /file-meta/upload` | `SaveChunkStream` (`disk_upload.go:154`) | userId 隔离（防 IDOR）+ 状态/序号边界校验 + **流式落盘** `SaveDiskChunkStream`（`io.Copy(MultiWriter(out,md5),reader)`，内存恒定）+ chunkHash 校验 + 幂等 `ON CONFLICT DO NOTHING` |
| `POST /file-meta/merge` | `Merge` / `mergeInstant` (`disk_upload.go:200`/`380`) | 原子抢占 `uploading→merging`（防并发双合并）→ 分片齐全校验 → 流式合并算完整 MD5 → **size 完整性校验** → 推 OSS → `resolveFolderForUpload` 懒建目录 → 同名拒绝 → 配额对账 → 建 `disk_files` → 清理 |
| `DELETE /file-meta/upload` | `Cancel` (`disk_upload.go:339`) | 清分片 + 会话 |
| 定时 | `CleanupStaleChunks` (`disk_upload.go:464`) | 24h TTL GC（uploading/merging 标 failed 留痕） |

handler `UploadChunk`（`api/v1/disk/disk_file.go:280`）用 `c.FormFile + fileHeader.Open` 取 reader 传给 `SaveChunkStream`，非整片 `ReadAll`。

### 1.3 数据模型

- `disk_upload_sessions`：会话（`identifier+user` 去重键，状态机 uploading/merging/completed/failed，存 quick_hash/strong_hash/merged_md5/file_id/storage_key）
- `disk_upload_chunks`：分片（唯一索引 `upload_id+chunk_number` 幂等收片）
- `disk_files`：四指纹 `md5/quick_hash/strong_hash/mid_hash` + `storage_type/storage_path` + `ref_count` 引用计数 + `is_favorite` 预留
- `sys_users.take_up_space`：配额增量字段（`reserveUserSpace` 原子条件 UPDATE 预占防 TOCTOU）

---

## 2. 优缺点对比（详细列举）

### 2.1 当前优于 jmal

| # | 维度 | 当前 | jmal（已核实） |
|---|---|---|---|
| 1 | **秒传** | 真内容哈希：quickHash+strongHash **+ midHash 防采样碰撞**，命中率高、不会张冠李戴 | **名存实亡**：前端 `identifier=size-文件名`（`uploader.js:192` 未覆盖 `generateUniqueIdentifier`），后端入库 md5=`size+relativePath+fileName`（`CommonUserFileService.java:337`），两者格式不同 → `existsByUserIdAndPathAndMd5` 几乎不命中；OSS 路径也只用 `doesObjectExist` 判对象存在、eTag 不做指纹 |
| 2 | **去重 / 引用计数** | `ref_count` 物理引用计数，跨位置秒传复用同一 OSS 对象 | **完全没有**，跨用户/跨目录相同文件各存一份 |
| 3 | **合并完整性** | 原子状态机防双合并 + 分片 `chunkHash` 校验 + 合并后 **size 校验** + 重算完整 MD5 入库 | `mergeFile` 只 `move`，**不校验哈希/大小/分片数**；严格顺序合并，缺片致残缺且无兜底（`MultipartUpload.java:241-272`） |
| 4 | **断点续传状态** | DB 会话表+分片表，**重启不丢、多实例共享** | 五块 Caffeine 全 `newBuilder().build()`（`CaffeineUtil.java:136-152`），**无 TTL/无容量/无持久化**，重启靠扫磁盘自愈、跨实例无法续传 |
| 5 | **配额** | `take_up_space` 原子条件 UPDATE 预占，防 TOCTOU，Check 预检 + Merge 兜底双保险 | 实时 aggregation（`MessageService.occupiedSpace`）+ 1 秒缓存，**非原子、滞后**，用户可在推送周期内超量上传 |
| 6 | **流式落盘** | `io.Copy` 边写边算 MD5，内存恒定 | `FileUtil.writeFromStream` 整片入 |
| 7 | **大小上限** | 双轨制：`sys_disk_config` 配置页 + 硬编码常量（10000 片 / 64MB），防洪泛 | **无任何单文件大小上限**（仅 quota 总量） |
| 8 | **前端引擎** | Web Worker 池（不阻塞 UI，自增 id 路由并发安全）+ **文件级池 × 分片级池双层并发** + 指数退避重试 + 暂停/继续/取消/重试全生命周期 + EMA 速度 + 字节级进度 + 文件夹聚合 | 内嵌 simple-uploader.js 源码（固定 3 并发、立即重试无退避、jQuery 操作 DOM 浮球 `globalUploader.vue` 1188 行） |
| 9 | **OSS 收尾** | merge 成功即建 `disk_files` 主动入库 | S3 直传 `completeMultipartUpload`（`OssController.java:110`）**只调 ossService 不入库**，靠下次列表刷新被动同步（滞后） |
| 10 | **文件名编码** | RFC5987 `filename*=UTF-8''...` | 旧式 ISO-8859-1 转码（`FileServiceImpl.java:703`），乱码风险 |
| 11 | **路径穿越** | `validateFileName` + `..` 拦截 + 分片边界常量（与 move/copy 对齐） | `decodeAndCheckPath` + `startsWith(userRoot)`，对齐 |

> 注（旧记忆纠正）：复核确认 jmal **OSS 中转上传入库的 md5 也是假值 identifier**（`WebOssService.getFileDocumentByOssPath:419` 把 `upload.getIdentifier()` 当 eTag 入库）；**S3 直传 complete 不入库**（被动同步）；**配额检查非原子滞后**。这三点旧记忆 `netdisk-upload-download-jmal-analysis` 未明确记录。

> 结论：**正确性、可靠性、大文件健壮性上当前已系统性优于 jmal**。jmal 在这些维度是反面教材（伪秒传、不校验、纯内存状态、非原子配额），当前都已避开。

### 2.2 当前相对 jmal 的不足 / 自身薄弱点（诚实清单）

| # | 问题 | 说明 | jmal 对照 |
|---|---|---|---|
| 1 | **合并是末尾一次性合并** | `MergeDiskChunks`（`disk_chunk.go:56`）顺序读所有 `.part` 写入 `merged.bin`，非边传边合并。GB 级文件有合并 IO 峰值 + 临时盘峰值（≈原文件×4，9GB ISO 需 ~30GB） | 边传边合并 + `FileChannel.transferTo` 零拷贝，IO 峰值摊平 |
| 2 | **handler 非真流式** | `c.FormFile` 让 gin 先 `ParseMultipartForm`（默认内存 32MB 超出写临时文件），分片先落 gin 临时文件再 copy 到 `.part`，**多一次磁盘 IO** | 直接 `file.getInputStream()` 落盘 |
| 3 | **merge 推 OSS 经临时文件** | `BuildFileHeader` 用 `os.CreateTemp`，非直接把 `merged.bin` 句柄流式喂给 OSS（`PutObject` 接 `io.Reader` 即可） | — |
| 4 | **OSS 抽象无 Multipart/Presigned** | `utils/upload` 只有 `UploadFile`，故上传只能服务器中转收片；下载预签名直连也做不了 | 完整 `getPresignedObjectUrl`/`UploadPart`/`ListParts`/`completeMultipartUpload` |
| 5 | **无 SSE 合并进度** | 大文件合并几分钟，前端无进度反馈（停在 99%） | 有 SSE 推送（虽是 jQuery 浮球） |
| 6 | **下载链路完全缺失** | 后端零下载接口；生产 RustFS 当前匿名公开下载，私有文件上线前硬性需改预签名（关联 `rustfs-public-download-before-netdisk`） | 打包 Zip + 307 预签名直连 + 本地静态 Range |
| 7 | **完整 MD5 未做端到端比对** | 合并算出的完整 MD5 只入库；前端不算完整 MD5（只算采样），无法做「合并 MD5 vs 声明 MD5」比对 | jmal 同样没有（当前有 chunkHash+size 兜底，**不算严重**） |
| 8 | **跨用户秒传未实现** | Check 只查 `user_id=?`，router 无 `verify-cross-user`（设计稿 D5/6.1 写了未落地） | jmal 同样无跨用户秒传 |
| 9 | **ref_count 减路径缺失** | `MoveToTrash` 只改 status+释放配额，**不动 ref_count**，物理对象永不释放（第 6 期回收站彻底删除补 `--`） | 无引用计数，无此问题也无此能力 |
| 10 | **permanentErrors 未区分** | 所有错误一律重试 N 次，415/500/501 类不可恢复错误浪费 | 区分 `[404,415,500,501]` 直接失败（`uploader.js:95`） |

---

## 3. 可从 jmal 借鉴的点（务实排序，标注价值 / 成本）

> 守 `security-perf-analysis-no-over-engineering`：先确认项目现有机制，不叠过度设计层。借鉴只取「服务器减压」与「大文件体验」两类范式，明确排除 jmal 的工程化包袱。

| 优先级 | 借鉴点 | 价值 | 成本 | 说明 |
|---|---|---|---|---|
| 🟢 P0 配套 | **OSS 抽象补 Presigned** | 高 | 中 | 关联下载硬前置（非上传本身，但上传链路姊妹缺口）。用 minio-go `PresignedGetObject` 实现 jmal `getPresignedObjectUrl(objectName,expiry,isDownload)`（`IOssService.java:246`）。**先收敛生产 `/oss/` 公开反代**，私有文件必须鉴权后才下发预签名。**不做多后端适配（阿里/腾讯/AWS），只 RustFS，避免过度设计** |
| 🟡 P1 大文件 | **边传边合并** | 高（仅 GB 级） | 中 | 9GB ISO 场景临时盘 ~30GB→~10GB、合并 IO 峰值摊平。Go 用 `os.OpenFile(O_APPEND)` + `io.Copy` + 乱序分片排队（参照 jmal `unWrittenCache`）。**实现复杂**（乱序/并发抢占），当前已有原子状态机+size 校验兜底，可作可选优化；常规文件收益有限 |
| 🟡 P1 大文件 | **merge 直喂 OSS 流式入口** | 高（仅 GB 级） | 低 | 给 `utils/upload` 加 `UploadStream(ctx, io.Reader, size)`，merge 直接把 `merged.bin` 句柄喂给 `PutObject`，省掉 `BuildFileHeader` 的 `os.CreateTemp` 临时文件（省一倍磁盘） |
| 🟠 P2 体验 | **SSE 合并进度** | 中 | 中 | 大文件合并期间推送进度。复用项目 `utils/sse/hub.go`，借鉴 jmal 300ms 节流防洪泛（`MessageService.pushMessageSync`） |
| 🟠 P2 小改进 | **permanentErrors 区分** | 中 | 极低 | 前端 `fetchUploadChunk` 返回 415/500/501 等直接标错不重试，几行代码 |
| 🟠 P2 真流式收片 | **handler 改 multipart.Reader** | 低-中 | 低 | 用 `c.Request.MultipartReader()` 边读 part 边喂 service，省掉 gin 临时文件那一次 IO。50MB 分片场景收益不大，可后置 |
| ⚪ 不借鉴 | simple-uploader 库 / jQuery 浮球 / 动态分片 SSE 推送 / 多 OSS 适配 | — | — | 当前引擎更现代（Worker 池+双层并发+EMA），`getChunkSize` 已按文件大小+设备内存动态且配置化，SSE 推送属重复；多 OSS 对单一 RustFS 是过度设计 |

---

## 4. 结论与节奏

- **当前上传链路在核心质量（真秒传、引用计数去重、合并完整性校验、DB 续传、原子配额、流式落盘、现代前端引擎）上已系统性领先 jmal**；jmal 在这些维度是反面教材，当前都已避开。
- **jmal 真正值得借鉴的是两类「服务器减压」范式**：① OSS 预签名直传/直连（省服务器中转带宽，成本高且需先解决鉴权）；② 边传边合并 + 流式推 OSS（降大文件临时盘峰值）。两者主要服务 **GB 级大文件** 场景，与「安装包/交付物」定位高度契合。
- 建议节奏：
  1. **先随下载链路（第 5 期穿插硬前置）补 OSS Presigned**（P0 配套，关联 RustFS 匿名下载隐患）。
  2. **边传边合并 + merge 流式推 OSS** 作为大文件专项优化择机做（对 9GB ISO 收益最大）。
  3. permanentErrors / SSE 进度作为低成本体验增强随手做。
  4. 其余（simple-uploader、jQuery UI、多 OSS、SSE 推分片大小）**明确不抄**。

---

## 附：jmal 关键文件索引（外部仓库 `/home/remote/jmal-cloud`）

- 前端：`jmal-cloud-view/src/components/SimpleUploader/globalUploader.vue`（总调度壳）、`S3DirectUploader.js`（S3 直传器）、`utils/simple-uploader/{uploader,file,chunk}.js`（内嵌开源库）
- 后端：`jmal-cloud-server/.../controller/rest/FileController.java`、`oss/web/OssController.java`、`service/impl/{FileServiceImpl,MultipartUpload,CommonUserFileService}.java`、`oss/web/WebOssService.java`、`oss/s3/AwsS3Service.java`、`util/CaffeineUtil.java`、`service/impl/MessageService.java`
