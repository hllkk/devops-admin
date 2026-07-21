# 网盘配置（disk-setting）前后端实现计划

## 目标

在系统设置中新增「网盘配置」tab，前端构建 `disk-setting.vue` 组件，后端新建 `SysDiskConfig` 模型 + Service，接入现有的 `GET/PUT /system/setting` 聚合接口。

---

## 一、后端（server/）

### 1.1 新建 Model：`server/model/system/sys_disk_config.go`

**设计原则**：
- 沿用现有单行表模式（`id=1`，`OPS_MODEL` 基座）
- 字段对齐前端表单，避免嵌套结构（与 `SysLdapConfig`/`SysNotifyConfig` 风格一致）
- 剔除参考项目中过度细化的字段（视频转码/解压缩暂不纳入，后续作为独立插件配置），聚焦**一般网盘系统通用配置**

**字段设计**（共 15 字段，按功能分四段）：

```
基础存储配置：
  maxUploadSize      float64  // 最大上传大小（数值）
  maxUploadSizeUnit  string   // 上传大小单位："MB"/"GB"/"TB"，默认 "MB"
  storageQuota       float64  // 默认存储配额（数值）
  storageQuotaUnit   string   // 配额单位："MB"/"GB"/"TB"，默认 "GB"
  allowedExtensions  string   // 允许上传的扩展名（逗号分隔），空=允许全部
  blockedExtensions  string   // 禁止上传的扩展名（逗号分隔），优先级高于允许
  recycleBinRetentionDays int // 回收站自动清理天数，默认 30

展示配置：
  diskName  string  // 网盘名称
  diskLogo  string  // Logo URL

分享安全配置：
  shareLinkPasswordRequired  bool  // 分享链接需要密码，默认 false
  shareLinkPasswordMinLength int   // 分享密码最小长度，默认 4

OnlyOffice 协同编辑：
  onlyOfficeEnabled     bool   // 启用 OnlyOffice，默认 false
  onlyOfficeServerUrl   string // Document Server 地址
  onlyOfficeTokenSecret string // JWT 签名密钥
  onlyOfficeCallbackUrl string // 回调地址（OnlyOffice 容器可访问的后端地址）
```

**字段剔除理由**：

| 参考项目字段 | 剔除原因 |
|---|---|
| `uploadLinkPasswordRequired/MinLength` | 上传链接是独立功能，当前网盘模块未启动，预留过度 |
| `syncEnabled` | 同步功能未定义具体实现 |
| `videoTranscode*`（5 个字段） | 视频转码是独立插件能力，应走 `plugin/` 体系 |
| `archive*`（5 个字段） | 解压缩是运维期配置，与存储基础配置不同生命周期 |
| `trashRetentionDays` | 已在基础段保留（`recycleBinRetentionDays`），名称改为更通用的回收站语义 |

### 1.2 新建 Service：`server/service/system/sys_disk_config.go`

- 沿用 `GeneralConfigService` 一模一样的模式：`Get`/`Set`/`Current`/`LoadAll` + `atomic.Value` 缓存
- 复制 `sys_ldap_config.go` 结构，替换类型为 `SysDiskConfig`

### 1.3 修改 Request Model：`server/model/system/request/sys_setting.go`

- `SettingConfig` 新增 `Disk *system.SysDiskConfig json:"disk,omitempty"`

### 1.4 修改 SettingService：`server/service/system/sys_setting.go`

- `Get()`：增加从 `DiskConfigService` 读取 `disk` 段落
- `Set()`：增加 `req.Disk != nil` 时保存到 `DiskConfigService`

### 1.5 修改 API enter：`server/api/v1/system/enter.go`

- 新增 `diskConfigService` 变量引用

### 1.6 修改 Service enter：`server/service/system/enter.go`

- 新增 `DiskConfigService` 嵌入

### 1.7 修改启动加载：`server/core/server.go`

- 新增 `(&system.DiskConfigService{}).LoadAll(context.Background())`

---

## 二、前端（web/）

### 2.1 新建组件：`web/src/views/_admin/system/setting/modules/disk-setting.vue`

**设计原则**：
- 遵循 §9.9 系统配置页模式（标签页式，非列表三件套）
- 用 NaiveUI `NTabs` 分三个 tab：「基础配置」「个性化」「OnlyOffice」
- 使用 UnoCSS 原子类，不写内联样式
- 字段文案走 i18n key
- `maxUploadSize` 和 `storageQuota` 使用 `number input + select unit` 组合控件

**Tab 结构**：

1. **基础配置** tab：
   - 最大上传大小：`NInputNumber` + `NSelect`（MB/GB/TB）
   - 存储配额：`NInputNumber` + `NSelect`（MB/GB/TB）
   - 允许的文件类型：`NInput`（逗号分隔，带 tooltip 说明）
   - 禁止的文件类型：`NInput`（逗号分隔，带 tooltip 说明）
   - 回收站保留天数：`NInputNumber` + "天" 后缀

2. **个性化** tab：
   - 网盘名称：`NInput`
   - 网盘 Logo：`NInput`
   - 分享需要密码：`NSwitch`
   - 分享密码最小长度：`NInputNumber`（min=4, max=32）

3. **OnlyOffice** tab：
   - 启用 OnlyOffice：`NSwitch`
   - OnlyOffice 地址：`NInput`（带 tooltip）
   - Secret 密钥：`NInput`（type=password，带 tooltip）
   - 回调地址：`NInput`（带 tooltip）

### 2.2 修改 `index.vue`（系统设置主页）

- 新增 `DiskSetting` 组件导入
- `config` 对象新增 `disk` 类型与默认值
- 模板中新增 `v-if="activeKey === 'disk'"` 渲染 `DiskSetting`
- `loadConfig()` 和 `handleSave()` 中加入 `disk` 段处理

### 2.3 修改 API 类型：`web/src/typings/api/system.api.d.ts`

- 新增 `DiskSettingConfig` 类型（对齐后端字段）
- `Setting` 类型新增 `disk?: DiskSettingConfig`

### 2.4 修改 i18n：`web/src/locales/langs/zh-cn.ts` + `en-us.ts`

- 新增网盘配置相关表单标签的 i18n key（中文 + 英文）

---

## 三、实现顺序（7 步）

| 步骤 | 文件 | 内容 |
|---|---|---|
| 1 | `server/model/system/sys_disk_config.go` | 新建 Model + Default 函数 |
| 2 | `server/service/system/sys_disk_config.go` | 新建 Service（Get/Set/Current/LoadAll） |
| 3 | `server/service/system/sys_setting.go` + `enter.go` | 修改 SettingService + enter |
| 4 | `server/model/system/request/sys_setting.go` + `api/v1/system/enter.go` + `core/server.go` | 修改 request / api enter / 启动加载 |
| 5 | `web/src/typings/api/system.api.d.ts` | 新增 `DiskSettingConfig` 类型 |
| 6 | `web/src/locales/langs/zh-cn.ts` + `en-us.ts` | 新增 i18n key |
| 7 | `web/src/views/_admin/system/setting/modules/disk-setting.vue` + `index.vue` | 新建前端组件 + 接入主设置页 |

---

## 四、关键设计决策

1. **单位选择**：`maxUploadSize`/`storageQuota` 使用 `value(float64) + unit(string)` 双字段，而非统一存字节——便于管理员直观配置 10GB/500MB 而不做心算
2. **扁平化**：不嵌套 OnlyOffice 子结构，与现有 `SysLdapConfig`/`SysNotifyConfig` 风格一致
3. **剔除字段**：视频转码/解压缩/上传链接/同步——属于独立功能域，等对应模块启动后再以插件或独立配置表形式接入
4. **向前兼容**：`SettingConfig.Disk` 使用 `omitempty`，旧前端不传 `/system/setting` 的 disk 段时后端不做任何保存动作
