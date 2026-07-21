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

## 待办（本次未做）

- 后端 /system/setting 聚合接口（GET/PUT）尚未实现，前端 fetchGetSetting/fetchUpdateSetting 当前会 404；后端实现时 service 需整合 SysGeneralConfig + SysSecurityConfig → 前端 { general, security }，注意：账户默认/日志清理字段属 SysGeneralConfig、验证码字段属 SysSecurityConfig。
- 后端 sys_security_config.go 注释"Captcha*/Limit*/PwdExpire* 前端不直接消费"已过时（前端现已消费），注释待订正。

## 关联

- 后端 model：[[system-model-rebuild]]（SysGeneralConfig/SysSecurityConfig）
- 验证码链路：[[go-captcha-login]]
