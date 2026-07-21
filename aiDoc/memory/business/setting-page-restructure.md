# 系统设置页面重构（常规瘦身 + 安全配置 8 tab）

> 2026-07-20 · 纯前端改动 · 对齐后端 SysSecurityConfig 已实现的安全字段

## 背景

「系统设置」页原先左侧两栏菜单（常规/安全）：常规配置塞了验证码、账户默认、日志清理等安全相关项；安全配置只对齐了后端 SysSecurityConfig 的 Password*/LoginFailLock*/IpValidation* 三段，而 Captcha*/Limit*/PwdExpire* 三段后端已实现、前端未消费。

## 改动（纯前端，后端 model 与 /system/setting 聚合接口均未动）

- **常规配置瘦身**：general-setting.vue 只留 systemName/systemDescription/logoUrl/faviconUrl 四项基本信息。
- **安全配置改 8 个 NTabs**：验证码 / 密码复杂度 / 限流 / 失败锁定 / 密码过期 / 访问控制 / 账户默认 / 日志清理。前 6 个 tab 操作 securityConfig，后 2 个（账户默认/日志清理）操作 generalConfig。
- **类型**（web/src/typings/api/system.api.d.ts）：
  - GeneralSettingConfig 移除 verifyCode*（验证码统一到 security.Captcha*），保留 userDefault*/logRetention*（后端仍在 SysGeneralConfig，仅页面渲染迁移）。
  - SecuritySettingConfig 新增 Captcha*(8) / Limit*(3) / PwdExpire*(2)，对齐后端六段。
- **双 model**：security-setting.vue 用 `defineModel('generalConfig')` + `defineModel('securityConfig')`；index.vue 传 `v-model:general-config` + `v-model:security-config`。账户默认/日志清理 tab 绑 generalConfig，其余绑 securityConfig。
- **默认值**：SECURITY_DEFAULTS 对齐后端 DefaultSecurityConfig / gorm default（passwordRequire* 全 false、captchaTimeout 3600、limitWindow 60、pwdExpireDays 90 等）。
- **i18n**：zh-cn + en-us 补 Captcha*/Limit*/PwdExpire* 字段、8 个 tab 标题、单位（秒/次）；同步 app.d.ts 的 setting schema（captchaType 加 image，新增 27 个 key）。typecheck 通过。

## 边界决策（用户 AskUserQuestion 确认）

- 账户默认 / 日志清理：**各自独立 tab**（不并入其他 tab）。
- IP 校验：**新增「访问控制」独立 tab**（不并入限流/失败锁定）。
- 验证码 tab 用后端 Captcha* 段（非 general 的 verifyCode* 平移）。

## 后端聚合接口落地（2026-07-20，方案 A）

- 新增 GeneralConfigService（Get/Set/Current/LoadAll + atomic 缓存 + DefaultGeneralConfig 兜底首行），对齐 SecurityConfigService 形态；core/server.go 启动加载。
- 新增薄 SettingService 聚合层（Get 读两表拼 {general,security}、Set 按段非空分发两表各自刷缓存；非跨表事务）。
- 新增 SettingApi（GetSetting/UpdateSetting）+ SettingRouter（挂 PrivateGroup，GET/PUT /system/setting）；注册到三处 enter.go + initialize/router.go 调用；request DTO model/system/request/sys_setting.go（SettingConfig）。
- 订正 sys_security_config.go 注释（Captcha*/Limit*/PwdExpire* 前端现已消费）。go build / go vet 通过，前端 fetchGetSetting/fetchUpdateSetting 现可对接。

## 仍未做

- /system/setting/public（登录页脱敏 PublicSetting）：验证码字段需整合 Captcha* ↔ verifyCode*，单独设计，本次不做。

## 字段对齐修复（2026-07-20）

- SysGeneralConfig 移除 6 个 verifyCode* 字段（EnableVerifyCode/VerifyCodeType/VerifyCodeLen/VerifyCodeExp/VerifyCodeTokenExp/VerifyInaccuracy）+ DefaultGeneralConfig 对应赋值；消除 GeneralConfigService.Set 的零值覆盖 bug（前端只传 8 字段，旧 model 多余字段被 Save 用零值覆盖）。
- 现常规配置前后端 8 字段、安全配置 24 字段均完全对齐（json tag 一一对应）。go build / go vet 通过。
- 旧库 verifyCode* 列成孤儿列（GORM AutoMigrate 不删列，model 不再映射），无害，可后续手动 DROP COLUMN。

## 关联

- 后端 model：[[system-model-rebuild]]（SysGeneralConfig/SysSecurityConfig）
- 验证码链路：[[go-captcha-login]]
