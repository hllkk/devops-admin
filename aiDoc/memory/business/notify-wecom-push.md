# 通知推送（企微应用消息 + 群机器人 + 晨报）

- 日期：2026-09-01
- 状态：已实现（go build/vet/test + 前端 typecheck/eslint 全过，待运行时联调企微真实凭证）
- 关联：[[notice-sse-realtime-push]]（站内通知+SSE 地基）、[[wecom-qrcode-login]]（企微凭证/sys_social 映射来源）、[[ai-gateway-budget-control]]（预算告警接入渠道）、[[ai-gateway-provider-balance]]（TokenPlan 余量数据源）

## 需求

1. **TokenPlan 工作日晨报**：工作日每天早上定时通知"阿里云 TokenPlan 已使用 XX%，N 天后重置（M月D日）。如已超量可临时切换到 MIMO 或其他个人自定义模型"，发给有权限的部门或用户（部门群或应用消息）。
2. **预算告警外发**：预算软限预警/硬限超限时，除现有站内通知外，增加企微应用消息通知到对应负责人/用户/部门。

## 调研结论（参照 /home/SoyDisk 同源实现）

可搬运资产：
- `WecomClient.SendMessage`（message/send 应用消息）：textcard（URL 空降级纯文本）、token 失效(40014/42001)清缓存重试一次、官方内容去重 1800s、单批 ≤1000（touser `|` 连接）、invaliduser 部分无效不整体失败。devops-admin 的 utils/wecom.go 同源但缺此段，直接移植。
- `pushToWecom` 范式：本地 userIds → join `sys_social(source=wecom)` 取企微 userid → 未绑定跳过+Warn 计数 → 截断上限 → 分批发送 → 失败仅日志 best-effort。
- 通知配置中心：本项目已移植 `sys_notify_config`（邮件+webhook+测试按钮，`notify-setting.vue`），缺企微段——本次扩展而非新建。
- 企微凭证复用 `sys_auth_config` 企微段（扫码登录在用），不另存。

SoyDisk 没有而本次新增：群机器人 webhook（markdown 进群）、定时内容型晨报。

## 用户决策（2026-09-01）

1. **群通知方式**：群机器人 + 应用消息都做（群机器人进部门群，应用消息精准到人）。
2. **晨报条件**：每工作日固定发（不设使用率门槛）。
3. **硬限通知范围**：scope 全体成员 + 负责人 + 超管（软限维持现状：负责人+超管）。
4. **前端落点**：系统设置「通知设置」Tab 内扩展（零新菜单/零 casbin 改动）。

## 设计

### 架构约束（关键）

`service/system` 已 import `service/gateway`（sys_user_manage.go）→ **gateway service 不得反向 import system service**。因此"场景组装"与"渠道发送"分离：

- 组装（纯 gateway 数据 → 通知草稿）：gateway service 纯函数，不 import system service（可 import system model）。
- 发送（三渠道）：`system.NotifySendService`，由 API 层 / initialize.timer 闭包调用（ClearDB 闭包注入同模式）。

### 渠道模型

`sys_notify_config` 扩展字段（沿用单行表+内存缓存热更新）：

```go
// 企微应用消息(凭证复用 sys_auth_config 企微段)
WecomPushEnabled  bool   // 应用消息渠道开关
WecomPushRedirectBase string // 消息跳转基础地址(空则 textcard 降级纯文本)
WecomPushMaxTargets int  // 单次人数上限(默认1000,超出截断)
// 企微群机器人
WecomBotEnabled bool
WecomBotWebhook string   // 群机器人 webhook URL
// 事件开关(控制该事件是否走外部渠道;站内通知不受控)
PushMorningReportEnabled bool // 晨报外部推送(默认false)
PushBudgetAlertEnabled   bool // 预算告警外部推送(默认true)
```

### 晨报策略表 `sys_notify_policy`（新，scene 注册制）

```go
SceneKey   string `gorm:"uniqueIndex;size:64"` // 第一期仅 token_plan_morning
Enabled    bool   `gorm:"default:false"`
TargetType string `gorm:"size:16"` // all/depts/users(depts 含子部门)
TargetIds  datatypes.JSON // [deptId/userId...]
Params     datatypes.JSON // 场景参数预留
```

经设置聚合 API（GET/PUT /system/setting）随 notify 段一起读写，不另立端点。

### 三渠道发送 `service/system/sys_notify_send.go`

- `SendInApp(title, content, targetType, userIds, deptIds)`：包 NoticeService.CreateNotice（定向+SSE+已读）。
- `SendWecomApp(userIds, title, desc, url)`：sys_social 映射 → 截断 → 分批 SendMessage。
- `SendWecomBot(markdown)`：POST 群机器人 webhook（msgtype=markdown）。
- 目标展开：depts → `DataScopeService.ExpandDeptIDs`（含子部门）→ 查成员。

### 场景A：TokenPlan 晨报

- `gateway.MorningReportService.BuildMorningReport(ctx)`：读 `gateway_provider_balance`（bailian token_plan，坐席+共享包）GROUP BY provider 汇总 total/surplus，cycleEnd 取 MAX 为重置日；输出标题/正文（markdown + 纯文本两版）。数据旁路只读，不进成本链路。
- `task.Register("TokenPlanMorningReport")`（initialize/timer.go 闭包）：读 policy → 关则跳过 → Build 组文案 → NotifySend 发送（站内+应用消息给目标、群机器人进群）。
- **发送时间可配置**（2026-09-01 补）：`sys_notify_policy.send_time`（HH:mm，默认 08:33）在通知设置页配置，保存时 `NotifyPolicyService.Upsert → syncMorningSchedule` 同步改写 `sys_timed_tasks` 的 spec（`m h * * 1-5`）并经 TimedTaskService 热重载；定时任务行不存在时自动创建（**已有库配置即生效，无需面板手工建任务**）。时间未变不动调度；调度同步失败策略不回滚（面板可兜底）。
- cron 种子：`33 8 * * 1-5`（工作日 08:33，与默认 send_time 对齐，排在 8:17 余量同步后拿当天快照）。
- 文案：
  ```
  【AI 平台晨报】阿里云 TokenPlan
  已使用 73.2%（剩余 2.14M / 总 8.0M Credits）
  9 月 10 日重置（9 天后）
  如已超量，可临时切换到 MIMO 或其他个人自定义模型。
  ```

### 场景B：预算告警接入渠道

- 目标扩展（service/gateway/budget_rule.go CheckBudgetAlerts）：软限=负责人+超管（现状）；**硬限=scope 全体成员+负责人+超管**。
- 组装纯函数 `BudgetAlertNotices(results) []NoticeDraft`（gateway service，纯数据）；API 层与 timer 闭包共用，杜绝两处重复。
- 定时路径补发（修现有 gap）：timer 的 CheckBudgetAlerts 闭包拿到 results 后同样走 NoticeDraft → NotifySend（此前定时任务拿到结果后丢弃不发，仅手动 API 触发才发）。
- 渠道：站内（现状保留）+ 企微应用消息（事件开关 PushBudgetAlertEnabled）。群机器人第一期不接告警（后续可扩展）。

### 告警防轰炸

预算告警已有 `gateway_budget_alert` 周期唯一键去重（同周期同类型只告警一次），外部渠道共用该节流；晨报天然每天一次 + 企微官方内容去重 1800s 双保险。

### utils/wecom.go 扩展（移植）

`SendMessage(ctx, touser, WecomTextCard)` + `WecomMessageSendBatch=1000` + `SendBotMessage(ctx, webhookURL, markdown)`（群机器人）。不改动现有登录链路。

### 前端

- `notify-setting.vue` 扩展三个子 Tab/区块：企微推送（应用消息开关/跳转地址/上限）、群机器人（开关+webhook）、晨报策略（开关+目标部门树/用户选择器，复用预算规则 drawer 的选择器模式）+ 两个测试按钮（test-wecom-app 选人实测 / test-wecom-bot 用表单 URL 实测）。
- 新端点：`POST /system/setting/notify/test-wecom-app`、`POST /system/setting/notify/test-wecom-bot`（未保存表单值可测，SoyDisk 同款体验）。
- i18n 三处同步（zh-cn / en-us / app.d.ts）。

## 落点清单

后端新建：
- `model/system/sys_notify_policy.go`、`source/system/sys_notify_policy.go`（建表注册，无种子）
- `service/system/sys_notify_send.go`

后端修改：
- `utils/wecom.go`（SendMessage/SendBotMessage）、`model/system/sys_notify_config.go`（扩字段）
- `service/gateway/budget_rule.go`（硬限目标扩展+NoticeDraft 纯函数）、`service/gateway/morning_report.go`（新建，组装）
- `initialize/timer.go`（晨报注册+预算闭包发送）、`source/system/timed_task.go`（晨报种子）
- `api/v1/system/sys_setting.go`+request（policy 聚合段+两测试端点）、`router/system/sys_setting.go`
- `api/v1/gateway/budget_rule.go`（sendBudgetNotifications 改走 NoticeDraft+NotifySend）

前端：`notify-setting.vue`、`service/api/system/setting.ts`、typings、i18n×3。

## 待办

- [ ] 运行时验证：企微凭证+应用可见范围+群机器人 webhook 真实联调（发测试消息）
- [ ] 群机器人接预算告警、邮件渠道接晨报（按需二期）
- [ ] 晨报多供应商扩展（当前仅百炼 token_plan）
