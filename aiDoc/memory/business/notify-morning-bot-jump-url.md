# 晨报群机器人消息加 AI 身份页跳转链接

- 日期：2026-09-16
- 状态：已实现（go build/vet/test + 前端 eslint/typecheck 通过）
- 关联：[[notify-wecom-push]]（晨报三渠道体系）、[[ai-gateway-provider-balance]]（TokenPlan 余量数据源）

## 需求

Token Plan 晨报消息加跳转地址，用户点击跳到本项目的 AI 身份页面（`/home`，默认落在「我的AI身份」Tab）。
用户澄清后的范围：**只针对群里收到的消息（企微群机器人渠道）**，站内通知与企微应用消息明确不考虑（企微应用消息维持跳 `/gateway`）。

同时确认：晨报内容用前端设置页模板渲染的能力此前已完整落地（`notify-setting.vue` 模板编辑 + `sys_notify_policy.params` + `BuildMorningReport` 渲染链），**无需改码**，本次仅做跳转。

## 设计

企微群机器人 markdown 原生支持 `[文字](URL)` 链接（须绝对地址）。跳转基础地址复用现有 `sys_notify_config.WecomPushRedirectBase`（通知设置→企微推送→消息跳转基础地址），不新增配置项；通过**模板变量**注入而非发送侧硬拼，用户自定义模板写了 `{{.JumpUrl}}` 才出链接，位置由模板控制。

- `MorningTemplateVars` 加 `JumpUrl string`；`BuildMorningReport(ctx)` → `BuildMorningReport(ctx, redirectBase string)`（唯一调用方 timer，timer 已持有 `notifyCfg`，直接传入复用内存缓存，不新增直查表）
- `morningJumpURL` helper：base 非空 `TrimSuffix + "/home"`；未配置返回空串，默认模板按 `{{if .JumpUrl}}` 整行省略，不发废链接（降级为现状）
- 默认 markdown 模板末尾追加 `{{if .JumpUrl}}\n[前往 AI 身份 ›]({{.JumpUrl}}){{end}}`（换行放 if 内，与 `ResetLine` 条件行同款，空值时输出与旧版完全一致）
- 前端：`notify-setting.vue` 可用变量列表加 `{{.JumpUrl}}` + i18n 三处同步（`notifyMorningTemplateVarJumpUrl`）

## 落点

- `server/service/gateway/morning_report.go`（Vars 字段/签名/默认模板/helper）
- `server/initialize/timer.go`（notifyCfg 上移 + 传参）
- `web/src/views/_admin/system/setting/modules/notify-setting.vue`、i18n×3

## 边界

- 跳转 base 未配置 → 无链接，行为与改动前一致
- 纯文本模板（站内/企微应用消息）不含该变量；站内通知无跳转能力是已知现状（sys_notice 无 URL 字段，铃铛点击仅标记已读）
- 群成员点链接走企微内置浏览器，未登录先跳登录页（正常）
