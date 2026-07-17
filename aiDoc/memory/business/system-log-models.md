# 登录日志 / 操作日志 model

- 日期：2026-07-17
- 状态：Model 层已落地（`go build ./model/system/` / `go vet` 通过）。仅 model 层，未含 request DTO / Service / API-Router。
- 关联前端：`web/src/typings/api/log.api.d.ts`（`Api.Log.{OperLog,LoginLog,BusinessType}`）、`web/src/typings/api/api.d.ts`（`Common.CommonRecord/EnableStatus`）、`web/src/typings/api/system.api.d.ts`（`System.DeviceType`）、`web/src/typings/common.d.ts`（`CommonType.IdType = string|number`）
- 接口契约（前端已定义，后端待补）：操作日志 `GET/DELETE /monitor/operlog/{list,/{ids},clean}`；登录日志 `GET/DELETE /log/loginlog/{list,/{ids},unlock/{user},clean}`

## 背景

`[[system-model-rebuild]]`（07-17）重建了 10 个 system model 但未含日志；`[[menu-seed-routes-alignment]]` 的菜单 seed 已预埋 loginlog/operlog 的接口权限码（按 RuoYi 规范），但 model 与接口一直缺。本次按前端 `Api.Log.*` 契约补齐 model。

## 新建 model

- `SysOperLog`（表 `sys_oper_log`）：`server/model/system/sys_oper_log.go`
- `SysLoginLog`（表 `sys_login_log`）：`server/model/system/sys_login_log.go`

## 字段对齐（前端契约 → 后端字段）

两表均嵌入 `global.OPS_AUDIT_MODEL` → 对齐 `Common.CommonRecord` 的 createBy/createTime/updateBy/updateTime（与 SysRole/SysNotice 一致）。

**SysOperLog**（`Api.Log.OperLog`）：operId(主键 int64,json `operId,string`)、title、businessType(`'0'~'9'` size:1)、method、requestMethod、operatorType(size:1)、operName、deptName、operUrl、operIp、operLocation、operParam(text)、jsonResult(text)、status(`'0'/'1'` size:1)、errorMsg(text)、operTime(time.Time)、costTime(int)

**SysLoginLog**（`Api.Log.LoginLog`）：infoId(主键 int64,json `infoId,string`)、userName、clientKey、deviceType(`pc|android|ios|xcx` size:8)、ipaddr、loginLocation、browser、os、status(`'0'/'1'` size:1)、msg、loginTime(time.Time)

## 关键决策

1. **业务时间 vs 记录时间并存**：operTime/loginTime 是业务发生时间，与 createTime（记录写入时间）语义不同，两者并存（与 RuoYi 一致）；前端类型为 string，后端用 time.Time（json 序列化为 ISO string 对齐）。
2. **嵌入 OPS_AUDIT_MODEL**：日志虽不常改，但前端 `CommonRecord` 固定展开 createBy/updateBy/createTime/updateTime，嵌入审计基座保证字段齐全 + 沿用现有模式（不另造轻量基座）。带 `DeletedAt` 软删除——清空/批量删除走物理删除时由后续 Service 层 `Unscoped()` 处理。
3. **枚举对齐前端字面量**：status=`'0'/'1'`(EnableStatus)、businessType=`'0'~'9'`(BusinessType)、deviceType=`pc/android/ios/xcx`(DeviceType)，后端均用 `string`（不做 iota 枚举）。
4. **ID 字符串传输**：operId/infoId 走雪花 int64 + `json:",string"`，对齐 `CommonType.IdType`。
5. **AutoMigrate 保持注释占位**：`initialize/gorm.go` 补 `// system.SysOperLog{},` 与既有 `// system.SysLoginLog{},` 成对，**不启用**（与同批未启用迁移的 SysNotice/SysPost/SysDictData 等一致，等接口链路重建时统一启用）。

## 待办（后续最小闭环）

- [ ] request DTO（OperLogSearchParams/LoginLogSearchParams，对齐前端 `Pick<...> & CommonSearchParams`）
- [ ] Service（列表分页走 `request.PageInfo` + `info.LimitOffset()`；批量删除/clean/unlock）
- [ ] API + Router（`/monitor/operlog`、`/log/loginlog`，Swagger `@Success` 落 `response.PageResult{Rows=[]SysOperLog}`）
- [ ] 启用 AutoMigrate 注册（取消注释）
- [ ] 操作日志切面（middleware/aspect 记录 operName/deptName/operIp/operParam/jsonResult/costTime/status）

## 相关文件

- 新建：`server/model/system/sys_oper_log.go`、`server/model/system/sys_login_log.go`
- 配套：`server/initialize/gorm.go`（迁移占位注释）
- 关联：[[system-model-rebuild]] [[menu-seed-routes-alignment]] [[snowflake-id-generator]] [[system-user-role-models]]
