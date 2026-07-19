# 通知公告管理接口

- 日期：2026-07-19
- 状态：后端接口全套已落地（`go build ./...` / `go vet` / 路由注册测试通过）。承接 [[system-model-rebuild]]（07-17）的 `SysNotice` model。
- 关联前端：`web/src/service/api/system/notice.ts`、`web/src/views/_admin/system/notice/`、`web/src/typings/api/system.api.d.ts`（`Api.System.Notice/NoticeSearchParams/NoticeOperateParams`）
- 接口契约：`GET /system/notice/list`、`POST /system/notice`、`PUT /system/notice`、`DELETE /system/notice/{ids}`

## 背景

`[[system-model-rebuild]]`（07-17）重建 `SysNotice` model，但建表未注册、service/api/router 三层全缺，前端 `notice.ts`（四接口）与页面 `views/_admin/system/notice`（add/edit/delete）已就绪却后端不通。本次打通四层并补建表。

## 实现（5 新建 + 4 修改）

新建：
- `source/system/sys_notice.go`：initializer（建 `SysNotice` 表，无种子，`initOrderNotice = initOrderDict + 1` 链尾自注册）
- `model/system/request/sys_notice.go`：`NoticeSearch`（嵌入 `PageInfo` + noticeTitle/noticeType）+ `NoticeOperateParams`（noticeId/noticeTitle/noticeType/noticeContent/status）
- `service/system/sys_notice.go`：`NoticeService`（GetNoticeList 含 createByName 组装 / Create / Update / Delete）
- `api/v1/system/sys_notice.go`：`NoticeApi` ×4
- `router/system/sys_notice.go`：`NoticeRouter.InitNoticeRouter`

修改（注册）：
- `service/system/enter.go`：`ServiceGroup` 加 `NoticeService`
- `api/v1/system/enter.go`：`ApiGroup` 加 `NoticeApi` + `noticeService` 别名
- `router/system/enter.go`：`RouterGroup` 加 `NoticeRouter` + `noticeApi` 别名
- `initialize/router.go`：`InitNoticeRouter(PrivateGroup)` 接入

## 关键决策

1. **补建表（关键修复）**：`SysNotice` 此前全仓无 AutoMigrate 注册，list 会报「表不存在」。新建 `source/system/sys_notice.go` initializer 走项目单一真源注册制（`init()` 自注册，`register_init.go` 已空白导入 `source/system` 触发）；首启走 `RegisterTables`、`/initdb` 走 initializer 两条路径都会建 `sys_notice`。
2. **createByName 批量查组装（不改 model）**：`SysNotice.CreateByName` 标 `gorm:"-"`。`GetNoticeList` 先分页查，再收集 `createBy` 去重批量查 `sys_users`（`id IN ?`，SysUser 主键列 `id`）取 `user_name` 回填。项目无「join 带创建者名」先例，批量查零风险（不改已提交 model 的 gorm tag）；名称查询失败不阻断列表（尽力而为）。
3. **业务实体走软删除**：`DeleteNotice` 用普通 `Delete`（软删，与 dict/role 一致），非日志那种 `Unscoped` 物理删——通知公告是业务实体，软删可恢复、语义正确。
4. **casbin 未启用 → 无需碰 seed**：`initialize/router.go` 的 `CasbinHandler()` 仍注释，新路由登录即可访问；菜单 seed 已预埋 `route.system_notice` 与按钮 Perms（`system:notice:*`）。
5. **UpdateNotice 用 map**：显式覆盖全部可编辑字段（含空串），避免 GORM struct 更新遗漏零值字段。
6. **审计字段从 claims 注入**：Create/Update 用 `utils.GetUserID(c)` 取 `createBy`/`updateBy`（与 dict 一致）。

## 待办

- [ ] 通知公告导出 `/system/notice/export`（前端 `notice.ts` 暂无 export 调用，按需再加）
- [ ] （可选）草稿/发布状态机、富文本内容长度上限等业务增强（前端当前是简单 CRUD）

## 相关文件

- 新建：`server/source/system/sys_notice.go`、`server/model/system/request/sys_notice.go`、`server/service/system/sys_notice.go`、`server/api/v1/system/sys_notice.go`、`server/router/system/sys_notice.go`
- 改动：`server/service/system/enter.go`、`server/api/v1/system/enter.go`、`server/router/system/enter.go`、`server/initialize/router.go`
- 关联：[[system-model-rebuild]]（承接其 SysNotice model）[[menu-seed-routes-alignment]]（菜单 seed 预埋 notice 路由项/权限）[[dict-management]]（CRUD+审计注入范式）
