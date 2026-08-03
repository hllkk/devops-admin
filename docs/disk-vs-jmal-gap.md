# 网盘模块 vs jmal-cloud:差距与值得参考的设计点

> 范围:以成品网盘 jmal-cloud 为功能对照,盘点 devops-admin 网盘当前实现度、缺失的必要设计、可直接借鉴的范式与必须避开的坑。
> 状态:对照分析稿(2026-08-02)。基于对当前 `server/disk`+`web/_disk` 与 `/home/remote/jmal-cloud` 前后端的实读,结合 `docs/disk-upload-design.md`、`aiDoc/modules/business-modules.md`、`aiDoc/memory/business/disk-*` 沉淀。
> 定位参考:`business-modules.md` 第13行——企业内部网盘,管理员上传大型安装包/交付物,对外链接+提取码匿名分享,对内账号/部门共享+OnlyOffice 协同。**判断"必要 vs 增强"以此定位为准**,不照搬 jmal 的全功能面。

---

## 1. 对比基准

- **当前已落地**:6 期迁移的第 1-3 期——目录树/面包屑(`server/service/disk/disk_file.go` GetFileList/ResolvePath/GetFolderTree)、CRUD(Mkdir/Rename/Move/Copy/MoveToTrash)、上传 MVP(Check 秒传+续传 / SaveChunk 流式 / Merge 原子抢占+流式合并算完整 MD5 / Cancel / 文件夹 relativePath 懒建)、配额(原子预占防 TOCTOU / 释放 / 超管豁免)、用户隔离(JWT 取 userId 防 IDOR)、会话 GC(CleanupStaleChunks 定时清理)。
- **jmal 完整面**:文件管理 / 回收站 / 上传下载 / 分享 / 预览 / 配额 / Lucene 全文搜索 / 收藏·最近·标签 / 版本历史 / OnlyOffice / 媒体库(转码) / SSE 通知 / RBAC / 多 OSS 抽象 / 加密。

---

## 2. 对比矩阵

| # | 维度 | 当前状态 | jmal 要点(后端 / 前端) | 必要性 |
|---|---|---|---|---|
| 1 | 列表/目录树/面包屑 | ✅ 对齐 | MongoDB 文档+path 扁平;`ShowFile.vue` 单组件+mixins 复用 | — |
| 2 | 文件 CRUD | ✅ 对齐 | 批量/跨目录/解压入口 | — |
| 3 | 回收站 | ⚠️ 仅 MoveToTrash(`disk_file.go:721-763`) | `Trash` 集合,恢复/彻底删/清空/`hidden` 防重复 | **P0** |
| 4 | 上传(分片/秒传/续传/文件夹) | ✅ **优于 jmal** | Caffeine 内存+ReentrantLock+OSS 直传 | — |
| 5 | 下载 | ❌ **后端无接口** | 打包 Zip / OSS 预签名 307 直连 / 本地静态 Range / `isAllowDownload` | **P0** |
| 6 | 分享 | ❌ 仅 DTO 预留字段 | shortId 短链/提取码/过期/4 档权限(DOWNLOAD·UPLOAD·DELETE·PUT)/子分享继承/分享 token JWT/挂载目录 | **P0** |
| 7 | 预览 | ❌ 仅缩略图 | 文本流式/图片/视频 HLS(FFmpeg)/音频/媒体封面/PDF/Office/CAD/压缩包解压 | **P0** |
| 8 | 配额 | ✅ **优于 jmal** | 实时 aggregation 统计(缺陷) | — |
| 9 | 搜索 | ⚠️ 仅文件名模糊(`disk_file.go:104-106`) | Lucene 全文+NGram+高级筛选(时间/大小/类型)+搜索历史+索引重建 | P1 筛选/全文增强 |
| 10 | 收藏/最近/标签 | ⚠️ `is_favorite` 字段预留(`model/disk/disk_file.go:32`) | 收藏页/最近页/标签页独立路由+TagDO 多对多+颜色+排序 | P1 收藏·最近/标签增强 |
| 11 | 版本历史/锁定 | ❌ 无 | GridFS 版本+列表/恢复/删除+OSS 对象锁定+Office 编辑历史 | **P1** |
| 12 | OnlyOffice 协同 | ⚠️ 配置预留(`sys_disk_config.go:24-28`) | `OfficeController`+`CallbackHandler` track 回调+JWT+分享文档编辑 | **P1** |
| 13 | 媒体库(音/视/图) | ❌ 无 | 视频转码 H264/H265+HLS 切片 m3u8+ts+vtt 字幕+EXIF+封面提取 | P2 增强(重资产) |
| 14 | 通知/消息中心 | ❌ 无 | SSE `SseController`+文件增删通知+传输进度+节流 300ms+128 连接+心跳 | P1 可复用 notice-sse |
| 15 | 权限/部门共享 | ⚠️ 仅用户隔离 | RBAC+`@Permission` AOP+分享操作权限细分 | P1 内部共享必要 |
| 16 | 会话 GC/定时清理 | ✅ 对齐 | Caffeine 自动过期+分片临时清理+定时清分享 | — |
| 17 | 多 OSS/存储抽象 | ⚠️ local/minio,无预签名/Multipart | `IOssService`:本地/阿里/腾讯/AWS/MinIO+预签名+Multipart+对象版本+跨 Bucket | P2 预签名+Multipart 必要/多后端增强 |
| 18 | 文件名编码/路径穿越 | ✅ **优于 jmal** | URL 编码+`../` 检测+255 字符 | — |
| 19 | 拖拽/右键/快捷键/批量 | ⚠️ 仅 hover 操作菜单 | 拖拽上传+拖拽移动+二级右键菜单+快捷键(空格/F2/Del/Ctrl+A·C·V·X·D·P)+批量 | P2 体验 |
| 20 | 移动端 | ⚠️ 强制列表+合并按钮 | 独立移动端路由+Vant+触摸手势+专用上传页 | P2 增强 |
| 21 | 主题/暗色/i18n | ✅ 对齐 | 亮暗+跟随系统+VueI18n+CSS 变量 | — |
| 22 | 传输任务管理 | ✅ 对齐 | 球体+面板+任务列表+SSE 进度 | —(缺 SSE 合并进度) |

> jmal 路径前缀:`/home/remote/jmal-cloud/jmal-cloud-server/`(后端)、`/home/remote/jmal-cloud/jmal-cloud-view/src/`(前端)。

---

## 3. 必要设计分级

### P0 — 核心缺失,网盘不可商用前必须补

1. **下载链路**(穿插硬前置,先于/伴随第5期)
   - 现状:后端零下载接口。生产 RustFS 当前**匿名公开下载**(记忆 `rustfs-public-download-before-netdisk`),网盘私有文件上线前硬性前置改预签名鉴权。设计稿 `disk-upload-design.md` D4 已定「后端代理 `GetObject`+Range 透传」。
   - 借鉴 jmal:`oss/IOssService.java:246` `getPresignedObjectUrl(objectName, expiryTime, isDownload)` 预签名范式;打包下载 Zip;本地静态 Range(零拷贝最优)。
   - 避坑:先收敛 `/oss/` 公开反代,私有文件必须鉴权后才下发预签名 URL 或走代理。

2. **分享体系**(第5期)
   - `business-modules.md` 已规划:外链(短链+提取码+有效期+下载次数)+ 内部共享(账号/部门,只读/编辑两档)。
   - 借鉴 jmal:`service/impl/ShareServiceImpl.java:64-290` shortId 短链、4 位提取码、4 档操作权限、分享 token JWT 鉴权、子分享继承、挂载目录(把分享目录挂到我的网盘)。
   - 避坑:jmal 权限粒度仅模块级无文件 ACL;内部共享要做细粒度,需先确认是否引入 Casbin(见 P1#9)。

3. **预览体系**(第4期)
   - `business-modules.md` 规划 txt/pdf/Office。前端当前仅缩略图。
   - 借鉴 jmal:文本用 `StreamingResponseBody` 流式读(避大文件 OOM)、图片缩略图、PDF/Office、压缩包解压预览。
   - 慎抄:视频 HLS(FFmpeg 转码)是重资产,对"安装包/交付物"定位收益低。建议**先做图片/PDF/文本/Office,视频先 Range 直连,不做 HLS 转码**,转码后置或不做。

### P1 — 规划内或配套必要

4. **回收站完整能力**:补恢复/清空/列表 + 独立页。避坑:jmal 恢复时未校验原路径同名,需加冲突检查(同名加后缀或提示)。
5. **版本历史**:`business-modules.md` 规划"覆盖上传→旧版进历史,留 10 份,回滚"。避坑:jmal 用 GridFS 存版本**无生命周期管理**(存储膨胀);本项目"留 10 份"正好补此坑,实现时务必带保留数量+过期清理策略。
6. **OnlyOffice 协同**:配置已预留,缺接口。借鉴 jmal `office/callbacks/CallbackHandler.java` track 回调保存 + JWT 签名 + 与文件操作层共用一个"编辑权限"开关(贴合 `business-modules.md` 设计)。
7. **通知/消息中心**:项目已有 `utils/sse/hub.go`(notice-sse 已落地),分享通知/合并进度可直接复用。借鉴 jmal 节流 300ms 防洪泛 + 单用户连接上限 + 心跳保活。
8. **收藏/最近访问**:第6期规划。`isFavorite` 字段已预留,补接口+独立页即可。
9. **内部共享权限档**:`business-modules.md` 规划只读/编辑两档。**注意**:记忆 `no-casbin-unless-explicitly-asked`——当前网盘专用组仅 JWTAuth+OperationRecord 不挂 Casbin;内部共享若要细粒度,需先与用户确认是否引入 Casbin/ACL,别自动套。

### P2 — 增强/体验,按需取舍(守 `security-perf-analysis-no-over-engineering`)

10. **OSS 抽象补预签名+Multipart**:记忆 `disk-migration-progress` 已点明当前 `utils/upload` 无 Presigned/Multipart API(故上传选后端收片)。补这两能力有明确收益(下载预签名直连 + 大文件合并走 ComposeObject),但**做"多后端(阿里/腾讯/AWS)适配"属过度设计**,当前只用 RustFS,不抄 jmal 多 OSS。
11. **搜索高级筛选**:文件名模糊已有,补"类型/大小/时间"筛选有收益;**Lucene 全文属增强**,对二进制安装包内容搜索收益低,不引入 Lucene 重依赖。
12. **交互:右键菜单+批量操作+拖拽上传**:当前 hover 菜单已有,补右键二级菜单 + 多选批量删/移/复制 + 拖拽上传是基本体验。借鉴 jmal `VContextmenu`+`shortcutKey`+`fileDrag`。
13. **媒体库/移动端/标签**:视频转码 HLS、独立移动端路由、标签系统——对内部安装包网盘属锦上添花,优先级最低,后置。

### 已优于 jmal、无需补(避开 jmal 坑)

- **传输链路**:DB 续传入库(重启不丢、多实例共享)> jmal Caffeine 纯内存;三重哈希(quickHash+strongHash+midHash)真实秒传 > jmal 拼接串假秒传;合并重算完整 MD5 + 原子状态机防双合并 > jmal 合并不校验+ReentrantLock 单机;流式落盘 > jmal 整片入内存。
- **配额**:原子条件 UPDATE 预占防 TOCTOU > jmal 实时 aggregation(海量文件性能瓶颈)。
- **文件名编码**:RFC5987 `filename*=UTF-8''...` > jmal ISO-8859-1 乱码风险。
- **路径穿越**:`validateFileName`+`..` 拦截+分片边界常量校验,与 jmal 对齐。

---

## 4. jmal 最值得直接借鉴的范式(带避坑)

| 范式 | jmal 位置 | 借鉴点 | 避坑 |
|---|---|---|---|
| 预签名直连下载 | `oss/IOssService.java:246` | `getPresignedObjectUrl(objectName, expiryTime, isDownload)`,RustFS 用 minio-go `PresignedGetObject` 实现 | 生产先收敛 `/oss/` 公开反代 |
| 分享短链+提取码+权限档 | `service/impl/ShareServiceImpl.java:64-290` | shortId、4 位提取码、4 档权限、分享 token JWT、挂载目录 | 补文件级 ACL,别停在模块级 |
| OnlyOffice track 回调 | `office/callbacks/CallbackHandler.java` | 回调保存+JWT+共用编辑开关 | — |
| SSE 节流推送 | `controller/rest/sse/SseController.java:19-100` | 300ms 节流+单用户连接上限+心跳 | 复用项目 `utils/sse/hub.go`,不重写 |
| 列表单组件复用 | `jmal-cloud-view/.../ShowFile/ShowFile.vue` | 首页/收藏/最近/标签/回收站/类型库复用同一列表组件 | Vue3 用 composable(`hooks/business/disk/`)对等拆分,别照抄 Vue2 mixins |

---

## 5. 必须避开的 jmal 设计缺陷

| # | jmal 缺陷 | 当前对策 |
|---|---|---|
| 1 | 本地秒传名存实亡(前端 identifier=大小-文件名,后端 md5=拼接串,不命中) | 已用真实内容哈希(quickHash+strongHash+midHash),合并算完整 MD5 入库 |
| 2 | 无内容寻址/引用计数,跨用户跨目录重复存 | 已有 `ref_count` 物理引用计数,跨位置秒传复用 |
| 3 | 合并不重算哈希、不校验完整性,静默写坏文件 | 合并已重算完整 MD5 与 identifier 比对 |
| 4 | 续传状态纯内存 Caffeine,多实例不同步、重启丢 | 已用 DB 会话表+分片表,重启不丢、多实例共享 |
| 5 | 严格顺序合并,缺片滞留致残缺文件 | 原子状态机 + 完整性校验兜底 |
| 6 | 版本历史 GridFS 无生命周期,存储膨胀 | 实现时带"留 10 份"+过期清理(本项目规划已定) |
| 7 | 回收站恢复未校验同名,覆盖风险 | 实现时加恢复前冲突检查 |
| 8 | 配额实时 aggregation 性能瓶颈 | 已用 `take_up_space` 增量字段+原子预占 |
| 9 | 权限粒度模块级,无文件 ACL | 内部共享需文件级授权(待定 Casbin,先问用户) |
| 10 | 文件名 ISO-8859-1 乱码 | 已用 RFC5987 |

---

## 6. 与现有迁移计划(6 期)的衔接

| 迁移期 | 状态 | 本对照对应项 |
|---|---|---|
| 第1期 列表/目录树 | ✅ | 维度 1 |
| 第2期 CRUD | ✅ | 维度 2(回收站仅移入→第6期补完整) |
| 第3期 上传 | ✅(优于 jmal) | 维度 4 |
| 第4期 预览 | 待启动 | P0-3、维度 7 |
| 第5期 分享 | 待启动 | P0-2、维度 6 |
| 第6期 配额/收藏/最近/回收站 | 待启动 | P0-1 回收站完整、P1 收藏/最近、维度 3/8/10 |
| 穿插硬前置 | 待启动 | P0-1 下载链路(不属期编号,先于/伴随第5期) |
| 配套 | 待启动 | P1 版本历史/OnlyOffice/通知/内部共享 |

每期进设计前,按 AGENT.MD 工作流规则用 `AskUserQuestion` 确认是否启用 superpowers 重流程。
