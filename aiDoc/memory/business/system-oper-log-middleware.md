# 操作日志中间件（异步落表 + 路由接入）

- 日期：2026-07-18
- 状态：中间件 + 异步写入 + 路由接入已落地（`go build ./...` / `go vet` 通过）。读路径（list/delete/clean/export API）未含。
- 关联前端：`web/src/views/_admin/log/operlog/`（index.vue / oper-log-search.vue / oper-log-view-drawer.vue）、`web/src/typings/api/log.api.d.ts`（`Api.Log.OperLog`）、`web/src/service/api/log/oper-log.ts`
- 接口契约（前端已定义，后端待补）：`GET/DELETE /monitor/operlog/{list,/{ids},clean}`、`/monitor/operlog/export`

## 背景

`[[system-log-models]]`（07-17）落地了 `SysOperLog`/`SysLoginLog` model 但仅 model 层，其「待办」第 5 项「操作日志切面（middleware 记录 operName/.../status）」一直未做。本次按前端 `Api.Log.OperLog` 契约 + 「安全/高性能/合理」要求，补齐 `SysOperLog` 的采集与落表，并接入路由实际生效。异步写入复刻同包 `DataAccessLogService`（chan + sync.Once 批量写）范式。

## 诊断

`server/middleware/operation.go` 此前**完全无法编译**：仍按重构前旧 model 写字段（`Body/Resp/ErrorMessage/LatencyMs/Agent/UserID/RequestID/TraceID/DeviceID`），而这些字段在 RuoYi 风格 `SysOperLog` 中已不存在；且 `OperationRecord()` 在路由里从未挂载。

## 实现（4 文件）

1. `server/service/system/sys_oper_log.go`（新增）：`SysOperLogService`，`Enqueue`（非阻塞）+ `StartWriter`/`startWriter`（后台批量：1024 缓冲、攒满 100 或每 2s 落表、满则丢弃告警）。复刻 `DataAccessLogService` 范式。
2. `server/service/system/enter.go`：`ServiceGroup` 注册 `SysOperLogService`。
3. `server/middleware/operation.go`（重写）：对齐 `SysOperLog` 字段、复用 AccessLog 脱敏缓冲、异步入队。
4. `server/initialize/router.go`：`Routers()` 启动写入协程；`PrivateGroup` 在 `JWTAuth` 之后挂 `OperationRecord()`。

## 字段填充策略

- 直接取：`OperIp`(ClientIP)、`OperUrl`(URL.Path)、`RequestMethod`(HTTP method)、`Method`(c.HandlerName)、`OperTime/CreateTime/UpdateTime`(start)、`CostTime`(ms)。
- `OperName/OperatorType`：仅读 JWTAuth 已放入 ctx 的 claims（不主动解析 token，公开路由无错误噪声）；有 claims → operName=username、operatorType='1'(后台)，否则 '0'。
- `OperParam`：GET/HEAD 取 query 串→单值 map→JSON 脱敏截断；其余复用 AccessLog 已脱敏截断的请求体（multipart 为「[文件]」）。
- `JsonResult`：复用 AccessLog 响应缓冲（上限 1MB）二次脱敏截断；二进制/下载响应只记「[二进制响应]」。
- `Status/ErrorMsg`：5xx 或 gin 私有错误 → '1' 异常 + errorMsg；否则 '0'。
- `Title`：路由模板去前缀前两段（如 `system/user`）；可被 `c.Set("ops_oper_title",...)` 覆盖。
- `BusinessType`：export/import 命中→'5'/'6'；否则 POST→'1' / PUT,PATCH→'2' / DELETE→'3' / 其余 '0'；可被 `c.Set("ops_oper_business_type",...)` 覆盖。
- 暂留空：`OperLocation`（无 IP 归属地库）、`DeptName`（claims 不含部门）——不臆造，后续扩展再填。

## 关键决策

1. **异步落表（高性能）**：请求路径零 DB、零 goroutine、零阻塞；满队丢弃保业务（审计尽力而为），与 `DataAccessLogService` 同策略。
2. **复用 AccessLog 缓冲（高性能 + 不重复读 body）**：AccessLog 全局阶段已读 body 并包装 writer 捕获响应；operation 只读 `ctxReqBodyKey`/`ctxRespBufferKey`，不二次读 body、不二次包装 writer。
3. **脱敏 + 截断（安全）**：请求/响应统一过 `logger.SanitizeBody`（敏感字段打码）+ `Truncate(AccessLogMaxBytes)`；二进制响应不落正文。
4. **operName 仅信 claims（安全）**：不主动解析 token，避免公开路由每请求一条错误日志。
5. **无注解框架，留覆盖钩子（合理 + 可扩展）**：title/businessType 默认按路由/方法推导，handler 可用 `c.Set` 两个键精确覆盖，零框架成本。
6. **保留后端独有扩展字段**：应「后端有前端无的字段需合理保留以便扩展」要求，补回 `RequestID`/`TraceID`（带 index），与 access log 的 request_id/trace_id 打通链路排障；`DeviceID` 较冷门暂不加。
7. **CreateTime 显式赋值**：`OPS_BASE.CreateTime` 无 autoCreateTime 标签，异步落表前显式赋 start，避免零值（比 `DataAccessLogService` 更严谨）。
8. **挂载位置**：`PrivateGroup` 中置于 `JWTAuth` 之后、`CasbinHandler` 之前——operName 已可用，且能记录授权拒绝(403)与业务结果；当前 PrivateGroup 路由全注释（登录链路待重建 [[httponly-cookie-auth]]），现阶段零运行时影响，路由恢复即生效。

## 待办（后续最小闭环）

- [ ] 读路径 Service/API/Router：`GET /monitor/operlog/list`（分页走 `request.PageInfo` + `LimitOffset()`）、`DELETE /monitor/operlog/{ids}`、`DELETE /monitor/operlog/clean`、`/monitor/operlog/export`；Swagger `@Success` 落 `response.PageResult{Rows=[]SysOperLog}`。
- [ ] 批量删除/clean 走物理删除（`Unscoped()`，因模型带软删除 `DeletedAt`）。
- [ ] （可选）IP 归属地：接 ip2region/qqwry 填 `OperLocation`；claims 扩展 deptId 或中间件查 `deptName`。
- [ ] （可选）title/businessType 精确化：在关键写接口 `c.Set` 覆盖，或引入轻量注解。

## 相关文件

- 新增：`server/service/system/sys_oper_log.go`
- 重写：`server/middleware/operation.go`
- 改动：`server/service/system/enter.go`、`server/initialize/router.go`、`server/model/system/sys_oper_log.go`（+RequestID/TraceID）
- 建表已注册：`initialize/gorm.go:53` + `source/system/sys_user.go:28`（AutoMigrate 会自动加 request_id/trace_id 列）
- 关联：[[system-log-models]]（承接其待办 #5；其 model 层于此补齐落表链路）[[system-model-rebuild]] [[menu-seed-routes-alignment]] [[httponly-cookie-auth]]
