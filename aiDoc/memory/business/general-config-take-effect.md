# 常规配置字段落地生效 + 字段归属重整

> 关联：[[setting-page-restructure]]（前一轮系统设置页面重构，遗留两个问题由本轮解决：general 字段渲染到安全页的归属错位、general 整表字段全悬空）

## 需求

系统设置两张单行配置表落地程度严重不对称：安全配置 `sys_security_config` 字段全被登录链路消费（真闭环），常规配置 `sys_general_config` 整表零消费——`GeneralConfigService.Current()` 无调用方，8 字段全悬空（systemName/logoUrl/faviconUrl 走 i18n/env 写死；userDefaultPassword 建号强制手填；userDefaultRole 语义不清无消费方；两个日志保留天数 ClearTable 硬编码且不覆盖 sys_login_log）。用户要求常规配置像安全配置那样落地生效，并重整 general 表「账户默认/日志清理」字段被前端渲染到安全页的归属错位。

## 实现（2026-07-21）

**后端落地生效**
- A1 日志清理：`task/clearTable.go` `ClearTable` 增 `ClearOptions{OperationLogRetentionDays,LoginLogRetentionDays}`，sys_operation_records/sys_login_log 用配置天数（≤0 跳过），jwt_blacklists/sys_timed_task_logs 保持硬编码。新增 sys_login_log 清理项，CompareField 用 `create_time`（OPS_AUDIT_MODEL 锁定列名，非 created_at）。`initialize/timer.go` ClearDB 闭包从 `GeneralConfigService.Current()` 取值传入——**避免 task 反向 import service/system 成环**（service/system 已 import task）。
- A2 已撤回：用户确认建号/重置密码均为必填项不应有缺省密码。`UserDefaultPassword`/`UserDefaultRole` 字段已从 `SysGeneralConfig` model、`DefaultGeneralConfig`、前端 `GeneralSettingConfig` 类型、`GENERAL_DEFAULTS` 常量、`general-setting.vue` 页面中删除。
- A3 public 接口：新增 `GET /system/setting/public`（PublicGroup，免鉴权脱敏：系统信息 4 字段 + 验证码段），`model/system/request/sys_setting.go::PublicSetting` + `SettingService.GetPublic`（读 Current 内存缓存，DB 未就绪返回默认）+ `SettingApi.GetPublicSetting` + `InitSettingRouter` 改双 group。

**前端字段归位**（B）
- `general-setting.vue` 加回日志清理字段（NDivider 分段），系统信息 4 字段保留；`security-setting.vue` 删 account/log tab + generalConfig 绑定，回归纯 6 tab；`index.vue` SecuritySetting 只传 security-config。

**前端系统信息接入**（C）
- `system.api.d.ts` PublicSetting 类型重写（删 enableVerifyCode*/enableWecom* 等过时字段，加 captcha*）。
- 新建 `store/modules/system/index.ts`：init() 拉 public（幂等，beforeEach fire-and-forget 触发），systemName 经 `i18n.global.mergeLocaleMessage` 覆盖 system.title 联动登录页+全局 logo（不改模板），faviconUrl 改 link[rel=icon] href。
- `system-logo.vue` 内部读 store.setting.logoUrl（有值渲染 img 否则原 SVG，3 处调用点不动）。

## 最终 SysGeneralConfig 字段（6 个，皆有消费方）

| 字段 | 消费方 |
|---|---|
| SystemName | system store → mergeLocaleMessage 覆盖 i18n system.title → 登录页+全局 logo 标题 |
| SystemDescription | public 接口返回（前端可展示） |
| LogoUrl | system-logo.vue 读 store.setting.logoUrl |
| FaviconUrl | system store → document.querySelector link[rel=icon].href |
| LoginLogRetentionDays | task.ClearTable → sys_login_log 清理 |
| OperationLogRetentionDays | task.ClearTable → sys_operation_records 清理 |

## 状态

已落地（go build/vet 通过，task 无循环依赖；pnpm typecheck/lint 0 error）。手动点触验证待用户跑应用：日志清理（改天数+插旧数据触发 ClearDB）、public 接口免鉴权、登录页标题/logo/favicon 随配置变化。
