# 登录日志 / 操作日志 读路径接口

- 日期：2026-07-19
- 状态：读路径全套接口已落地（`go build ./...` / `go vet` / 绑定单测 / 路由注册测试通过）。承接 [[system-log-models]]（model）与 [[system-oper-log-middleware]]（写路径），日志监控两页打通。
- 关联前端：`web/src/service/api/log/{login-log,oper-log}.ts`、`web/src/views/_admin/log/{loginlog,operlog}/`、`web/src/typings/api/log.api.d.ts`
- 接口契约（前端已统一到 `/log`）：登录日志 `GET /log/loginlog/list`、`DELETE /log/loginlog/{ids}`、`DELETE /log/loginlog/clean`、`GET /log/loginlog/unlock/{username}`；操作日志 `GET /log/operlog/list`、`DELETE /log/operlog/{ids}`、`DELETE /log/operlog/clean`

## 背景

`[[system-log-models]]`（07-17）落地 model，`[[system-oper-log-middleware]]`（07-18）落地写路径（中间件 + 异步落表），读路径（list/批量删除/清空/解锁）一直缺，前端两页是死的。用户将操作日志前端路径由 `/monitor/operlog/*` 统一到 `/log/operlog/*`（与登录日志 `/log/loginlog/*` 对齐，防割裂），本次按统一后的契约补齐后端读路径。

## 实现（7 文件 + 注册点）

新建：
- `server/model/system/request/sys_login_log.go`：`LoginLogSearch`（嵌入 `PageInfo` + userName/ipaddr/status）
- `server/model/system/request/sys_oper_log.go`：`OperLogSearch`（嵌入 `PageInfo` + title/businessType/operName/operIp/status + BeginTime/EndTime）
- `server/api/v1/system/sys_login_log.go`：`LoginLogApi` ×4
- `server/api/v1/system/sys_oper_log.go`：`OperLogApi` ×3
- `server/router/system/sys_login_log.go`、`sys_oper_log.go`：路由挂在 PrivateGroup

补 service 读方法：
- `service/system/sys_login_log.go`：`GetLoginLogList`/`DeleteLoginLog`/`CleanLoginLog`/`UnlockLoginLog`（转调 `ClearLoginFail`）
- `service/system/sys_oper_log.go`：`GetOperLogList`/`DeleteOperLog`/`CleanOperLog`

注册：
- `api/v1/system/enter.go`：`ApiGroup` + `loginLogService`/`operLogService` 别名
- `router/system/enter.go`：`RouterGroup` + `loginLogApi`/`operLogApi` 别名
- `initialize/router.go`：`InitLoginLogRouter`/`InitOperLogRouter` 挂 PrivateGroup

## 关键决策

1. **物理删除（Unscoped）**：两表嵌入 `OPS_AUDIT_MODEL` 带 `DeletedAt` 软删除；日志为 append-only 审计数据无恢复需求，删除/clean 走 `Unscoped()` 物理删，避免软删垃圾行累积（兑现 [[system-oper-log-middleware]] 待办）。
2. **时间范围用 `c.Query` 显式取**：前端 `qs.stringify` 将 `params:{beginTime,endTime}` 序列化为 `params[beginTime]/params[endTime]`（bracket），实测 gin struct binding **不支持** bracket 嵌套映射到子 struct（单测 `TestOperLogSearchParamsQuery` 验证）；改 `OperLogSearch.BeginTime/EndTime` 标 `form:"-"`，api 层 `c.Query("params[beginTime]")` 显式取（gin `c.Query` 是原始 key 匹配，能取到 bracket key）。
3. **casbin 未启用 → 无需碰菜单 seed**：`initialize/router.go:79` 的 `CasbinHandler()` 仍注释，PrivateGroup 不做接口级校验，新路由登录即可访问。菜单 seed（`source/system/sys_menu.go`）已预埋 `route.log_{loginlog,operlog}` 路由项与按钮 Perms，前端 `hasAuth` 已对齐。
4. **DataScope 自动跳过**：日志表 `sys_` 前缀，`DataScope` 回调自动跳过，不受数据权限影响。
5. **unlock 走 ClearLoginFail**：`UnlockLoginLog` 转调同包 `security_lock.ClearLoginFail(ctx, username)`，清失败计数与锁（复用既有能力，不另造）。
6. **建表已就绪**：`initialize/gorm.go:52-53` + `source/system/sys_user.go:32` 已注册 `SysLoginLog`/`SysOperLog` AutoMigrate（[[system-log-models]] 待办「启用 AutoMigrate」实际已完成，表已建）。

## 待办

- [ ] 日志导出 `/log/{operlog,loginlog}/export`：用户决定本次不做（前端 `operlog/index.vue:handleExport` 仍指向旧 `/monitor/operlog/export`），导出能力待统一另行处理。
- [ ] （可选）IP 归属地 / deptName：接 ip2region 填 `OperLocation`、claims 扩展 deptId 查 `deptName`。

## 相关文件

- 新建：`server/model/system/request/sys_{login,oper}_log.go`、`server/api/v1/system/sys_{login,oper}_log.go`、`server/router/system/sys_{login,oper}_log.go`、`server/api/v1/system/sys_log_binding_test.go`
- 改动：`server/service/system/sys_{login,oper}_log.go`、`server/api/v1/system/enter.go`、`server/router/system/enter.go`、`server/initialize/router.go`
- 关联：[[system-log-models]]（承接其 model）[[system-oper-log-middleware]]（承接其写路径，读路径待办于此完成）[[menu-seed-routes-alignment]] [[httponly-cookie-auth]]
