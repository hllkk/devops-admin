# 网盘模块拆分清单 (disk-split-inventory)

> 盘点时间:2026-08-03。用于将网盘模块独立成新仓库的迁移/删除依据。
> 方案:两仓库物理隔离(fork 保留 git 历史)。老仓库 devops-admin 删网盘,新仓库独占网盘。
> A 类 = 网盘独有(可整体迁移/删除);B 类 = 与基座/其他模块耦合(需按行拆分)。

## 总览结论

网盘模块**高度内聚但与基座多处耦合**。三个改变拆分形态的关键发现:

1. **OnlyOffice 是「纸面功能」**:仅在 `sys_disk_config` 模型 + 前端 `disk-setting.vue` + i18n 中存在配置入口,**无任何后端协同编辑代码、无部署容器**。协同编辑功能实质为零,独立产品要重做。
2. **RustFS / minio 是全站共享基础设施**(头像/媒体/网盘共用),**不是网盘独有**。网盘独立后要自带对象存储;`minio_oss.go` 的网盘专属方法(`UploadStream`/`DownloadStream`)随网盘迁走,基础方法两边各留。
3. **`sys_users.TakeUpSpace` 隐蔽耦合**:网盘配额字段寄生在共享 sys_users 表(`model/system/sys_user.go` L50),新老仓库都要处理。

---

## 一、前端 web/

### 1.1 A 类「网盘独有」

| 文件路径 | 说明 |
|---|---|
| `web/src/views/_disk/disk/index.vue` | 网盘主页(DiskHome) |
| `web/src/views/_disk/disk/modules/*.vue` | breadcrumb / context-menu / drop-zone / file-card / file-empty / file-grid / file-icon / file-list / file-type-menu / move-copy-modal / toolbar / transfer-panel / transfer-sphere |
| `web/src/layouts/disk-layout/index.vue` | 网盘专属布局外壳 |
| `web/src/service/api/disk/index.ts` | 网盘 API barrel |
| `web/src/service/api/disk/file.ts` | 网盘文件 API(列表/路径/上传/合并/秒传/移动/复制/删除/打包下载) |
| `web/src/service/api/disk/quota.ts` | 网盘配额 API |
| `web/src/typings/api/disk.api.d.ts` | 网盘全部 TS 类型(377 行) |
| `web/src/store/modules/disk/index.ts` | 网盘 Pinia store |
| `web/src/hooks/business/disk/use-disk-create.ts` | 行内新建 hook |
| `web/src/hooks/business/disk/use-disk-upload.ts` | 上传引擎 hook(分片/秒传/续传/并发/重试) |
| `web/src/utils/disk.ts` | 网盘专用工具 |
| `web/src/utils/disk_hash.ts` | 秒传采样哈希 |
| `web/src/utils/disk_hash-worker.ts` | 哈希 Web Worker |
| `web/src/views/_admin/system/setting/modules/disk-setting.vue` | 系统设置里的「网盘配置」面板 |

### 1.2 B 类「耦合」(前端)

| 文件 | 网盘部分 | 拆分动作 |
|---|---|---|
| `constants/module.ts` | `'disk'` 联合类型(L16)、`DEFAULT_MODULE='disk'`(L19)、`ALL_MODULES`(L22)、`MODULE_CONFIG.disk`(L31) | 老仓库:删 disk 项,DEFAULT_MODULE 改回 admin |
| `constants/business.ts` | `menuLayoutRecord['2']='网盘布局'`(L57) | 老仓库删该项 |
| `enum/index.ts` | `SetupStoreId.Disk='disk-store'`(L10) | 老仓库删 |
| `locales/langs/zh-cn.ts` | route.disk(L266)、module.disk(L274)、page.disk 整段(L330-476)、system.setting.disk 段(L716/821-857) | 三处按行剔除(见迁移要点 5) |
| `locales/langs/en-us.ts` | 与 zh-cn 对称(L270/278/334-480/720/825-857) | 同上 |
| `typings/app.d.ts` | i18n schema 多处 disk 声明(L507/519/907/970/1011-1045) | 按行删 |
| `router/elegant/imports.ts` | DiskLayout 导入(L11/17/43) | 自动生成,删 _disk 后重跑 elegant-router |
| `router/elegant/routes.ts` | disk 路由块(L54-64) | 同上 |
| `router/elegant/transform.ts` | `"disk":"/disk"`(L170) | 同上 |
| `router/routes/index.ts` | `tagLayoutMeta()`(L34-52) | 函数服务全局,删 disk 分支保留函数 |
| `layouts/auto-layout/index.vue` | DiskLayout 引用 + currentModule==='disk' 分支(L5/14-19) | 删 disk 分支 |
| `store/modules/theme/index.ts` | `effectiveLayoutMode` disk 强制 vertical-mix(L37-44) | 删 disk 分支 |
| `typings/router.d.ts` | `RouteMeta.useDiskLayout`(L86-92) | 删字段 |
| `typings/elegant-router.d.ts` | RouteLayout/路由名联合含 "disk"(L12/24/82/131) | 自动生成 |
| `typings/api/system.api.d.ts` | module 联合含 'disk'(L286-287)、`DiskSettingConfig`(L611-645)、`SettingConfig.disk`(L684) | 老仓库删 |
| `views/_admin/system/setting/index.vue` | DiskSetting 引用 + disk tab + 读写逻辑(L9/13/23/45/52/78-81/108/120/129/167/198) | 老仓库删 disk tab |
| `views/_admin/system/menu/*` | 「网盘布局」菜单布局选项 | 删 menuLayoutRecord['2'] |
| 其余注释级耦合 | `router/index.ts`(L23)、`base-layout`(L44)、`global-header`(L24)、`global-menu`(L39)、`theme/shared`(L272)、`route store`(L101)、`menu-tree`(L32)、`storage.d.ts`(L58)、`auth.d.ts`(L70) | 多为注释,可留可清 |

> 路由分类:`disk` 路由无 `meta.constant`,归 authRoutes;实际可见性由后端 `/route/getUserRoutes` 下发(menu seed 驱动),前端 disk 路由只做 component 解析桩。

---

## 二、后端 server/

### 2.1 A 类「网盘独有」

| 文件路径 | 说明 |
|---|---|
| `server/model/disk/disk_file.go` | DiskFile 模型(table disk_files) |
| `server/model/disk/disk_upload.go` | DiskUploadSession / DiskUploadChunk + 状态常量 |
| `server/model/disk/request/disk_file.go` | 文件请求 DTO |
| `server/model/disk/request/disk_upload.go` | 上传请求 DTO |
| `server/model/disk/response/disk_file.go` | 文件响应体 |
| `server/model/disk/response/disk_upload.go` | 上传响应体 |
| `server/router/disk/enter.go` | 路由聚合 |
| `server/router/disk/disk_file.go` | `/file-meta/*` 路由注册 |
| `server/service/disk/enter.go` | 服务聚合 |
| `server/service/disk/disk_file.go` | 文件业务(CRUD/移动/复制/打包下载) |
| `server/service/disk/disk_upload.go` | 上传业务(秒传/分片/合并/续传/配额) |
| `server/service/disk/disk_quota.go` | 配额对账 |
| `server/service/disk/disk_upload_test.go` | 上传单测 |
| `server/api/v1/disk/enter.go` | API 聚合 + service 注入 |
| `server/api/v1/disk/disk_file.go` | `/file-meta/*` handler(含 zip 流式下载) |
| `server/model/system/sys_disk_config.go` | SysDiskConfig 模型 + DefaultDiskConfig() |
| `server/service/system/sys_disk_config.go` | DiskConfigService(Get/Set + 内存缓存) |
| `server/utils/upload/disk_chunk.go` | 网盘分片暂存(隔离于 media chunk.go) |

### 2.2 B 类「耦合」(后端)

| 文件 | 网盘部分 | 拆分动作 |
|---|---|---|
| `api/v1/enter.go` | import disk + `DiskApiGroup` 字段(L4/9) | 老仓库删字段+import |
| `router/enter.go` | import disk + `Disk` 字段(L4/11) | 同上 |
| `service/enter.go` | import disk + `DiskServiceGroup` 字段(L4/9) | 同上 |
| `service/system/enter.go` | `DiskConfigService` 成员(L13) | 老仓库删 |
| `initialize/router.go` | disk 路由组 + InitDiskFileRouter(L69/94/102-103/136-141) | 老仓库删整块 |
| `initialize/gorm.go` | SysDiskConfig + disk 三表 AutoMigrate(L67/76-78) | 老仓库删 |
| `initialize/other.go` | RUSTFS_ROOT_USER/PASSWORD 环境覆盖(L78-85) | 网盘迁走后,RustFS 仍服务全站,**老仓库保留** |
| `initialize/timer.go` | CleanStaleDiskChunks 定时任务(L8/32-33) | 老仓库删 |
| `core/server.go` | DiskConfigService.LoadAll(L39) | 老仓库删 |
| `config.yaml` | minio 段(L102-106)、oss-type:minio(L229) | **共享,两边都留** |
| `config/oss_minio.go` | Minio struct | **共享,两边都留** |
| `utils/upload/minio_oss.go` | UploadStream/DownloadStream/HeadObject(网盘专属,L84-86/116) | 网盘专属方法随网盘迁;基础 UploadFile/DeleteFile/ListFiles 两边留 |
| `utils/upload/upload.go` | OSS interface + NewOss() | **共享抽象,两边都留** |
| `utils/upload/chunk.go` | chunkRoot() 读 Media.ChunkDir | media 逻辑,两边留 |
| `source/system/sys_menu.go` | 网盘菜单 seed(L130-141) | 老仓库删 seed |
| `source/system/sys_setting.go` | SysDiskConfig/DiskFile 注册 + DefaultDiskConfig(L7/16/45/52/91-95) | 老仓库删 |
| `source/system/timed_task.go` | CleanStaleDiskChunks seed(L51) | 老仓库删 |
| `service/system/sys_setting.go` | disk Get/拼响应/分发保存(L12/26/38/59-60) | 老仓库删 disk 段 |
| `model/system/request/sys_setting.go` | `Disk *SysDiskConfig` 字段(L13) | 老仓库删 |
| `api/v1/system/sys_setting.go` | swagger disk(L37) | 老仓库删 |
| `model/system/sys_user.go` | **TakeUpSpace 字段(L50)** | 隐蔽耦合,见迁移要点 3 |
| `service/system/sys_route.go` | routeLayoutDisk 常量 + resolveLayout disk(L22/58/61/67-68) | 老仓库删 disk 布局解析 |
| `source/system/sys_role.go` | 注释提 disk(L48) | 注释级 |
| `model/system/sys_route.go` | Module 注释(L12) | 注释级 |
| `global/snowflake.go` | 注释(L17) | 注释级 |

> 网盘**不直接依赖** aws_s3/cloudflare_r2/aliyun_oss/tencent_cos/qiniu/huawei_obs/local 等 OSS provider(网盘固定走 minio/RustFS)。这些是兄弟实现,迁网盘不带,删网盘不影响。

---

## 三、部署 deploy/(全 B 类「全站共享」)

> OnlyOffice **无任何容器编排**。RustFS 是全站对象存储,**不能整体随网盘迁走**。

| 文件 | 网盘相关部分 | 处理 |
|---|---|---|
| `docker-dev/docker-compose.yml` | rustfs + rustfs_init(L61-115) | 共享,老仓库保留;新仓库自带一份 |
| `docker-prod/docker-compose.yml` | RUSTFS env/卷/依赖 + rustfs 服务(L64-65/73/86/148-203) | 同上 |
| `docker-prod/.env` / `.env.example` | RUSTFS_* 配置(L45-52) | 同上 |
| `docker-prod/config/config.yaml` | minio 段 + oss-type(L113-120/237) | 同上 |
| `docker-prod/Dockerfile.server` | RustFS 密码 env 覆盖注释 + ca-certificates(L10/42) | 同上 |
| `docker-prod/nginx/nginx.conf` | client_max_body_size(网盘大文件) + /oss/ 反代(L54/85-88) | /oss/ 共享保留;body_size 两边按需 |
| `docker-{dev,prod}/README.md` | RustFS 说明多处 | 老仓库保留;新仓库自带 |

---

## 四、文档

### 4.1 A 类「网盘独有」

- `docs/disk-upload-design.md` / `docs/disk-upload-vs-jmal.md` / `docs/disk-vs-jmal-gap.md`
- `aiDoc/memory/business/disk-backend-foundation.md` / `disk-crud.md` / `disk-upload-backend.md` / `rustfs-object-storage.md`

### 4.2 B 类「耦合」

- `aiDoc/modules/business-modules.md`:网盘节(L7/9/11-49/82)。**注意文档称网盘「未启动」与代码现状(已大量实现)严重偏差,拆分时一并校正**
- `aiDoc/memory/business/module-component-decouple.md` / `module-isolation-backend-driven.md`:disk 作为四模块用例
- `aiDoc/relations/development-workflow.md`:L8 提网盘

---

## 五、迁移要点

1. **OnlyOffice 纸面化**:仅配置入口,无协同代码/容器。配置项随网盘走,协同功能独立产品要重做。
2. **RustFS 共享**:`minio_oss.go` 的 `UploadStream`/`DownloadStream` 网盘专属(随迁),`Minio` struct + `UploadFile`/`DeleteFile` 服务全站(两边留)。config.yaml minio 段、deploy rustfs 容器、nginx /oss/ 反代**不随网盘迁走**;新仓库部署自带一份 RustFS。
3. **TakeUpSpace 寄生 sys_users**:网盘配额字段在共享用户表。新仓库承接(网盘自带 sys_users),老仓库清理(注意 DB 已有数据)。
4. **四模块机制深度耦合**:disk 是 admin/disk/server/gateway 之一,`DEFAULT_MODULE='disk'`(默认登录进网盘)。老仓库拆完默认模块改回 admin,清四模块联合类型里的 'disk'。
5. **i18n 三处散落**(zh-cn / en-us / app.d.ts):disk 内容是巨型文件里的多段落,按行拆(行号见 1.2 / 2.2 表)。
6. **菜单/配置 seed 混编**:`source/system/sys_menu.go` / `sys_setting.go` / `timed_task.go` 里网盘 seed 与其他模块混在同一初始化函数,按行剔除。

---

## 六、执行 checklist(仓库名定后启动)

- [ ] 新仓库:fork 当前仓库到 GitHub(fork 法保留历史)
- [ ] 新仓库瘦身:删 admin/server/gateway 业务模块,保留基座 + disk;DEFAULT_MODULE 保持 disk;验证独立编译运行
- [ ] 新仓库:RustFS/minio 基础设施自带一份;承接 TakeUpSpace
- [ ] 老仓库瘦身:对照 A 类整体删 + B 类按行清(DEFAULT_MODULE 改 admin、i18n 三处、seed、enter.go、initialize 注册)
- [ ] 老仓库:清理 sys_users.TakeUpSpace(评估 DB 数据)
- [ ] 两仓库文档/部署归置(校正 business-modules.md「未启动」偏差)
