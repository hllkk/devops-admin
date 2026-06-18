# 压缩包解压功能设计（解压到当前目录 / 解压到...）

- 日期：2026-06-17
- 范围：DevOps Admin 网盘模块（前端 `frontend/` + 后端 `backend/`）
- 目标：实现「解压到当前目录」「解压到...」两个功能，补全后端文件登记链路，保证安全与性能，前后端类型一一对应、无冗余类型。

## 1. 背景与现状

压缩包预览/解压的引擎与弹窗骨架已存在，但关键链路是断的：

| 层 | 现状 | 位置 |
|---|---|---|
| 后端解压引擎 | 已完整：7z/unar 解压、GBK 编码、zip slip 防护、zip bomb 检测（≤50000 文件 / ≤50GB）、符号链接清理 | `backend/utils/archive.go` |
| 后端 API/Service | 路由 + handler 已存在 `POST /archive/extract`，但**致命缺口：只写磁盘，未建 `disk_files` 数据库记录** → 解压后文件在 UI 不可见；且用 `destFolderId`（根目录无 ID 不可用） | `backend/service/disk/disk_archive.go:135`、`backend/model/disk/archive.go` |
| 前端弹窗 | `archive-action-dialog.vue` 已有两个按钮，但在 `disk/index.vue:708-709` 两个事件**只关闭弹窗，零逻辑** | `frontend/src/views/disk/index.vue` |
| 前端 API | `fetchExtractArchive()` 已定义但**从未被调用** | `frontend/src/service/api/disk/archive.ts:38` |
| 文件夹选择器 | 可复用：`move-copy-dialog.vue`（面包屑 + 按 path 选择）+ `fetchGetFolderList(path)` | `frontend/src/views/disk/modules/` |

生产存储为 **rustfs（S3 兼容对象存储）**（`conf/config.yaml` → `oss-type: rustfs`），因此无法「直接解压进目标」——必须先本地暂存、再逐个上传到 rustfs、最后登记入库。

## 2. 已确认的决策

1. **解压语义**
   - 「解压到当前目录」：在当前目录下新建以压缩包名（去后缀）命名的子目录，内容解压进去。
   - 「解压到...」：直接解压到用户所选目标目录内，不再建子目录。
2. **同名冲突**：自动重命名（追加 `(1)/(2)` 后缀，保留两者），绝不覆盖。
3. **性能模式**：同步阻塞 + 前端全屏 Loading（沿用前端 10 分钟超时、后端 `archiveCmdTimeout=30min`）。
4. **v1 范围**：仅「我的网盘」（`disk/index.vue`）。`shared-with-me`（他人分享页）的解压留作后续迭代，其弹窗仅「预览」可用。

## 3. 方案选型

| 方案 | 结论 |
|---|---|
| **A. 暂存→校验→上传入库（事务）** ✅采用 | 对象存储无法直接写入，必须暂存；暂存让配额校验、冲突重命名、原子回滚都干净可控 |
| B. 直接解压到目标→事后登记 | ❌ 对 OSS 不成立；冲突/配额超标难以中途处理；回滚困难 |
| C. 逐条流式解压上传 | ❌ 现有 7z/unar 封装不支持单成员流式；冲突/配额决策无法中途做出；同步模式下过度工程 |

## 4. 后端设计

### 4.1 API 契约（统一用 `path`，与网盘模块移动/复制/建文件夹一致）

```
POST /archive/extract
Body: { fileId: string, destPath: string, intoSubfolder: boolean }
成功响应: { code, msg }  // 前端刷新当前目录
```

- `destPath`：目标目录相对路径（如 `/` 或 `/docs`），根目录 `/` 可用。
- `intoSubfolder`：`true` = 「解压到当前目录」（在 destPath 下建以压缩包名命名的子目录）；`false` = 「解压到...」。
- 冲突解析（含子目录名本身）全部在后端按 DB 真值处理，前端无需预判。

请求模型（重写 `model/disk/archive.go` 的 `ExtractArchiveRequest`）：

```go
type ExtractArchiveRequest struct {
    FileID        string `json:"fileId" binding:"required"`
    DestPath      string `json:"destPath" binding:"required"` // 目标目录相对路径
    IntoSubfolder bool   `json:"intoSubfolder"`               // 是否套一层以压缩包命名的子目录
}
```

handler 同步更新 Swagger 注解（`@Param data body disk.ExtractArchiveRequest`），修改后运行 `swag init`。

### 4.2 Service 流水线（重写 `service/disk/disk_archive.go` 的 `ExtractArchive`）

```
1. 权限：源文件取读/下载权限；destPath 取 UPLOAD(写) 权限；校验两者归属当前用户
2. 解析源 → 本地临时文件（复用 resolveLocalPath，rustfs 源→下载到 temp）
3. DetectArchiveType，拒绝 unknown / 不支持的格式
4. 在系统 temp 下创建唯一暂存目录 stageDir
5. 按类型解压到 stageDir（7z / unar），跑 PostExtractValidation（zip bomb / 符号链接 / .. 防护）
6. 计算有效目标：
   - if intoSubfolder: sub = 冲突解析后的压缩包名(去后缀，ValidateFolderName 净化)
                       effectiveDest = path.Join(destPath, sub)
   - else:             effectiveDest = destPath
7. 遍历 stageDir 树：
   - 逐条 IsValidArchiveEntry 校验；按 effectiveDest 计算每条相对 Path
   - 汇总 totalSize / fileCount（受 maxArchiveFileCount=50000 / maxArchiveTotalSize=50GB 约束）
   - 预解析每个叶子名冲突（查 DB user+path+name），生成重命名映射（追加 (1)/(2)…）
8. 配额预检：QuotaService.CheckQuota(userID, role, totalSize) —— 超额直接拒绝，零副作用
9. 提交（事务 + 失败回滚）：
   - ensureFolderRecords(userID, effectiveDest 及每个目录相对路径) 建目录记录
   - 对每个文件：流式上传到 rustfs（UploadFromReader）→ 得 storageKey → 组装 File 记录
   - 批量 Create File（目录在前、按 depth 排序），设置
     ParentID / Depth / ChildrenCount / Suffix / ContentType / Size / StoragePath / StorageType
   - tx 提交成功 → QuotaService.UpdateUsedSpace(userID, +totalSize)
10. 清理：defer 删除 stageDir + 源临时文件
11. 任一步失败：回滚 tx；清理已上传的 OSS 对象；删除 stageDir —— 保证不留孤儿数据
```

### 4.3 复用的现成积木

- `utils/archive.go`：解压 + 安全（`ExtractArchive7z`/`ExtractArchiveUnar`/`PostExtractValidation`/`IsValidArchiveEntry`/`DetectArchiveType`）。
- `FileService.ensureFolderRecords(userID, dirPath)`：自动建中间目录 DB 记录。
- `StorageProvider`：`BuildPath(userID, dir)`、`WriteFileContent`、`DownloadFile`；rustfs 专用 `UploadFromReader(ctx, objectKey, reader, size, contentType)` 流式上传。
- `QuotaService`：`CheckQuota(userID, role, size)`、`UpdateUsedSpace(userID, delta)`。
- `utils`：`ValidatePath`、`ValidateFolderName`、`NormalizePathWithFilepath`、`GetContentType`、`GetFileSuffix`。
- `FileAuditService.Record`：解压操作审计。

## 5. 安全措施

- **Zip Slip**：`IsValidArchiveEntry` 拒绝 `..`/绝对路径/危险字符；解压前后双校验，最终 Path 经 `ValidatePath` 归一化。
- **Zip Bomb**：复用 `PostExtractValidation`（≤50000 文件 / ≤50GB）+ 遍历阶段二次累计。
- **越权**：源文件读权限 + dest 目录写权限 + 两者归属当前用户（防止借解压写入他人目录）。
- **配额**：解压前预检，超额拒绝，杜绝刷爆存储。
- **文件名净化**：`ValidateFolderName`，防 `../`、控制字符、系统保留名。
- **资源回收**：超时（`archiveCmdTimeout=30min`）+ 临时文件 defer 清理；失败回滚清理已上传 OSS 对象。
- **审计**：`FileAuditService.Record` 记录解压（源文件、目标、文件数、总大小、操作人）。

## 6. 性能措施

- **流式上传**：大文件用 `UploadFromReader` 流式上传，不整体载入内存。
- **批量事务**：目录与文件记录一次事务批量插入，减少 DB 往返。
- **配额预检在前**：避免解压完才发现超额的无效 IO。
- **（可选）上传并发**：worker pool（如 4 并发）控制对 OSS 的压力——标注为可选项，v1 可先串行。
- 前端 10 分钟超时 + 全屏 Loading + 按钮 loading/禁用，防重复点击。

## 7. 前端设计

### 7.1 类型与 API（`frontend/src/service/api/disk/archive.ts`）

重写为对象参数，与后端一一对应；删除旧的 `(fileId, destFolderId)` 签名避免冗余：

```ts
import type { CommonType } from '@/typings/common';

export interface ExtractArchiveParams {
  fileId: CommonType.IdType;
  destPath: string;
  intoSubfolder: boolean;
}

export function fetchExtractArchive(params: ExtractArchiveParams) {
  return request<void>({
    url: '/archive/extract',
    method: 'post',
    data: params,
    timeout: ARCHIVE_EXTRACT_TIMEOUT
  });
}
```

（字段名 `fileId / destPath / intoSubfolder` 与后端 JSON tag 完全一致。）

### 7.2 `disk/index.vue` 接线（替换 708-709 的空逻辑）

- `@extract-here`：`destPath = diskStore.getCurrentPathString()`，`intoSubfolder = true`；调用 `fetchExtractArchive`，loading 中禁用按钮，成功 `$message.success` + 刷新文件列表。
- `@extract-to`：打开目标目录选择器（新建 `extract-to-dialog.vue`），确认后 `intoSubfolder = false` 调用。
- 解压过程中弹窗显示 loading；成功后关闭弹窗并刷新；失败 `$message.error` 提示后端返回的错误信息。

### 7.3 新建 `extract-to-dialog.vue`

基于 `move-copy-dialog.vue` 提炼的纯目录选择弹窗：去掉复制/移动模式，仅单选目录 → `emit('confirm', destPath)`。复用面包屑 + `fetchGetFolderList(path)`。

遵循前端规范：
- 组件编写顺序：导入 → defineOptions → Props/Emits → 状态 → 方法 → 模板。
- 需要在函数中渲染组件/图标时用 `lang="tsx"` + tsx 语法，不用 h 函数渲染组件。
- 响应式（`sm:` 断点）+ 暗黑模式（`dark:` 前缀）适配。
- 图标复用 `useSvgIcon` / `SvgIcon`。

## 8. 国际化

新增 `page.disk.extract.*` 键（解压中 / 解压成功 / 解压失败 / 选择目标目录 / 已自动重命名冲突项 等）。**先定义类型再写翻译**（遵循现有国际化规范），中英文同步。

## 9. 测试策略

- **后端**：扩展 `backend/utils/archive_test.go` 与新增 `service/disk/disk_archive_test.go` 覆盖：
  - zip slip 条目被拒绝；zip bomb 被拒绝（超文件数/超总大小）。
  - 配额超额被拒绝（零副作用：无 OSS 对象、无 DB 记录、无 stageDir 残留）。
  - 同名自动重命名（`(1)`/`(2)`）。
  - 子目录模式（`intoSubfolder=true`）与直解模式（`intoSubfolder=false`）。
  - rustfs 上传 + 入库一致性（文件数、Path、ParentID、Depth、Size、StoragePath）。
  - 失败回滚无孤儿（OSS 对象已清理、stageDir 已删除）。
- **前端**：`pnpm typecheck` + `pnpm lint` 零错误为提交前提；关键路径手测（解压到当前目录、解压到指定目录、冲突重命名、大包 loading）。
- **类型一致性核对**：`fileId / destPath / intoSubfolder` 前后端字段名与类型逐一比对。

## 10. 受影响文件清单

后端：
- `backend/model/disk/archive.go`（重写 `ExtractArchiveRequest`）
- `backend/api/v1/disk/disk_archive.go`（`ExtractArchive` handler + Swagger）
- `backend/service/disk/disk_archive.go`（重写 `ExtractArchive` service 流水线 + 冲突解析/上传入库助手）
- `backend/utils/archive_test.go`、新增 `backend/service/disk/disk_archive_test.go`
- `docs/`（`swag init` 重新生成 swagger）

前端：
- `frontend/src/service/api/disk/archive.ts`（重写 `fetchExtractArchive` + `ExtractArchiveParams`）
- `frontend/src/views/disk/index.vue`（接线 `extract-here` / `extract-to`）
- `frontend/src/views/disk/modules/extract-to-dialog.vue`（新建目标目录选择器）
- `frontend/src/components/disk/archive-action-dialog.vue`（按需补 loading/禁用态）
- 国际化文件（`page.disk.extract.*`）

## 11. 范围边界与风险

- **v1 仅「我的网盘」**：`shared-with-me` 解压（涉及把他人文件解压进自己网盘的权限模型）留作后续。
- **暂不含**：异步任务/进度条（已确认同步模式）。
- **风险**：rustfs 大包同步上传耗时——通过流式上传 + 合理超时 + Loading 缓解；如后续反馈不足再迭代异步。
