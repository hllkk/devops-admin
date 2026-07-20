# 通知公告 SSE 实时推送（cookie 鉴权 + 定向 + 已读跟踪）

- 日期：2026-07-20
- 状态：已实现（后端 build / 前端 vue-tsc 全绿）
- 前置：[[notice-management]]（CRUD 四接口已落地）

## 背景

notice CRUD 只解决"后台管理公告"。本块补"实时推送 + 定向投递 + 已读跟踪"：发布公告/通知后，在线用户经 SSE 立即收到铃铛弹窗；定向通知落 sys_notice_record 跟踪已读，离线用户上线拉取补齐。地基复用既有的 `server/utils/sse/hub.go`（GVA 同款通用 SSE 中枢，单进程内存版）。

## 决策（用户拍板）

1. **鉴权方案 B（httpOnly cookie）**：前端 sse.ts 去掉 token-in-query，EventSource `withCredentials` 自动携带 `x-token` cookie；后端 SSE handler 直接 `utils.GetUserID(c)`（GetToken 双取 header→cookie）。非方案 A（token-in-query）。
2. **投递范围**：公告（type=2）`Broadcast` 全员、不入 record；通知（type=1）定向，按**用户或部门（含子部门）**。
3. **已读跟踪**：要。定向通知预生成 sys_notice_record 未读行，支持未读计数/历史/标已读；公告不入 record（历史回看走 CRUD 管理页）。

## 实现（14 处）

后端（9）：
- `model/system/sys_notice_record.go` 新建：SysNoticeRecord（OPS_MODEL + NoticeId/UserId/ReadAt，复合唯一索引 idx_notice_user）
- `model/system/sys_notice.go`：加类型常量 NoticeTypeNotice='1'/NoticeTypeAnnouncement='2'
- `model/system/request/sys_notice.go`：NoticeOperateParams 加 TargetType/TargetUserIds/TargetDeptIds；新增 NoticeUnreadSearch/NoticeReadParams
- `source/system/sys_notice.go`：MigrateTable 加 AutoMigrate(SysNoticeRecord)
- `service/system/data_scope.go`：新增公开 ExpandDeptIDs（包一层 subtreeUnion，含子部门）
- `service/system/sys_notice.go`：CreateNotice 改造（按 targetType 展开目标→事务落主表+record→在线 PublishToUsers）；新增 publishNotice/expandTargetUserIDs/GetNoticeRecordList/MarkNoticeRead/NoticeRecordVO；3 处 data_scope:skip 旁路本人查询
- `api/v1/system/sys_notice.go`：Stream（GetUserID→uint→sse.Stream）/GetUnreadNotice/MarkNoticeRead + import sse
- `router/system/sys_notice.go`：unread/read 路由 + InitNoticeSSERouter
- `initialize/router.go`：专用 SSE 组（GinRecovery 后、AccessLog 前注册，绕过 captureWriter）

前端（4）：
- `web/src/utils/sse.ts`：去 localStg token 检查、去 query token、withCredentials:true、Data JSON 按 payload.type 分流（通知 success / 告警 error）
- `web/src/service/api/system/notice.ts`：fetchGetUnreadNotice/fetchMarkNoticeRead
- `web/src/store/modules/notice/index.ts`：NoticeItem 加 noticeId；addNotice 改 unshift 置顶；readNotice/readAll async 同步后端；新增 fetchUnread
- `web/src/layouts/modules/global-header/components/message-button.vue`：onMounted 调 fetchUnread（离线补齐）

## 关键设计点（避坑）

- **专用 SSE 组必须注册在 AccessLog 之前**：`middleware/access_log.go` 的 captureWriter 包装 c.Writer 并缓冲 resp body，会破坏 `c.Writer.(http.Flusher)` 断言 + 内存膨胀。gin 中间件只作用于注册后的路由，故 SSE 组在 `Router.Use(AccessLog())` 之前注册即绕过。
- **hub 用 uint / 用户 ID 是 int64**：所有 SSE 调用点 uint(id) 转换。
- **不建独立 alertStream 端点**：devops-admin 单 hub 架构，用户连 `/resource/sse` 即 Subscribe，定时任务 alertFailure 的 PublishToUsers 经同一连接送达。前端 useEventSource(url,[]) 只监听 message，故告警 Event.Name 留空走 message、payload.type 分流（见 [[role-id-datascope-legacy-fix]]）。
- **data_scope:skip**：notice 的本人查询（GetNoticeRecordList/MarkNoticeRead/expandTargetUserIDs）在 PrivateGroup（DataScope 中间件）下，须显式 Set("data_scope:skip", true) 旁路，否则被加范围条件查不全。

## 运行/部署注意

- **建表**：常规启动 RegisterTables 会 AutoMigrate sys_notice_record（老库重启即补建）。
- **.env**：`VITE_APP_SSE=Y`（默认 N，否则前端不连）。
- **cookie**：dev 走 vite proxy 同源自动带；生产跨域后端需 `Access-Control-Allow-Credentials: true` 且 `Allow-Origin` 非通配。
- **withCredentials**：vueuse useEventSource 透传 withCredentials 给 EventSource 需运行时验证（SSE 连接若 401 则改原生 EventSource）。
- **多实例**：hub.go 单进程内存版，跨实例广播需上层 Redis pub/sub 扇出（当前单实例可用）。

## 关联

- [[notice-management]]：CRUD 四接口基础
- [[role-id-datascope-legacy-fix]]：定时任务告警接收人 + 告警 payload.type 分流
