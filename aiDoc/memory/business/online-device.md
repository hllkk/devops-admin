# 个人中心在线设备后端实现

> 2026-07-23。前端 `views/_builtin/user-center/modules/online-table.vue` + `service/api/monitor/online.ts` + `typings/api/monitor.api.d.ts` 早已完整(分页表格:设备类型/IP/登录地点/浏览器/操作系统/登录时间 + 强制下线;GET /monitor/online、DELETE /monitor/online/myself/:tokenId),后端完全缺失。gva 3.0-beta 无此功能(只有同款 jwt 黑名单 + 多端互踢 + 登录日志无设备/地点),无可借鉴实现,需自研。本次补齐后端,前端零改动(仅修 online-table row-key `noticeId`→`tokenId` 残留 bug)。

## 现状核对

- 认证 httpOnly cookie 单 x-token(use-multipoint=false 时 Redis 不存 token);jwt claims 无 jti → tokenId 无来源
- JwtBlacklist + `JsonInBlacklist`/`GetRedisJWT`/`LoadAll` 已有;LoadAll 把黑名单载入 OPS_CACHE,黑名单检查 = `OPS_CACHE.Get(token)`
- 登录 `sys_user.go` Login/TokenNext 已 `ParseUserAgent`(browser/os/deviceType)+ ClientIP + 写 sys_login_log;`ParseIPLocation(ip)`→"国家|区域|城市|ISP|国家码"
- `SetRedisJWT` 每用户只存 1 个 token(多端互踢用),不满足"多会话列表" → 需独立多会话存储

## 核心机制(自研)

- **token 加 jti**:`utils/jwt.go` CreateClaims 设 `RegisteredClaims.ID=uuid.NewString()`(jwt.go 本次补 import github.com/google/uuid)。tokenId=claims.RegisteredClaims.ID。注:`CustomClaims` 值嵌入 `BaseClaims.ID`(int64 userId) 与 `jwt.RegisteredClaims.ID`(string jti) 同名,访问须全限定 `claims.BaseClaims.ID` / `claims.RegisteredClaims.ID`,裸 `claims.ID` 编译报 ambiguous selector
- **Redis 多会话存储**(不入库):key=`online:session:<userId>`(Hash),field=tokenId,value=`OnlineSession` JSON(含 Token 明文供踢下线入黑名单)。整 key TTL=jwt 过期时间。列表查询兜底清理:`ParseToken` 失败(过期)/黑名单命中(`OPS_CACHE.Get`)→ HDEL 清理并跳过
- **GET /monitor/online**:个人中心视角只返当前用户自己(userName 搜索参数忽略),ipaddr 模糊过滤,loginTime 降序,`PageInfo.LimitOffset` 分页,`PageResult{Rows=[]OnlineDevice}`
- **DELETE /monitor/online/myself/:tokenId**:从 claims 取 userId,HGET 校验属于自己(伪造 tokenId 也只在自己 hash 查 → 天然防踢他人)→ token 入黑名单 → HDEL → `data=true`
- **loginTime 返回毫秒时间戳(int64)**对齐前端 `OnlineUser.loginTime: number`(字段级对齐,非 time.Time ISO)
- **deptName 登录时按 deptId 查 sys_departments 快照存入会话**

## 分层落点(纯 system 包内,路由 /monitor/online 顶级)

- `model/system/sys_online.go`:`OnlineDevice`(对外)+ `OnlineSession`(Redis 内部含 Token)
- `model/system/request/sys_online.go`:`OnlineSearch{PageInfo;Ipaddr}`
- `service/system/sys_online.go`:`OnlineService{RecordSession/ListSessions/GetOnlineList/RemoveSession/KickSession}` + 注册 enter.go
- `api/v1/system/sys_online.go`:`OnlineApi{GetOnlineList/KickOnlineDevice}` + Swagger + 注册 enter.go
- `router/system/sys_online.go`:`OnlineRouter{InitOnlineRouter}` + 注册 enter.go;`initialize/router.go` PrivateGroup 块加 InitOnlineRouter
- `utils/jwt.go`:CreateClaims 加 jti
- `api/v1/system/sys_user.go`:TokenNext 三处成功点调 `recordSession`(私有 helper);Logout 删当前会话

## 契约要点

- 前端 online-table.vue remote 分页,传 pageNum/pageSize(query),后端 `ShouldBindQuery(OnlineSearch)`
- tokenId 路径参数 `DELETE /monitor/online/myself/:tokenId`,`c.Param("tokenId")`
- 强制下线返回 `data=true`(对齐前端 boolean)
- 前端 row-key 原为 `noticeId`(OnlineUser 无此字段,残留 bug),已改 `tokenId`

## 已知点:多端互踢(use-multipoint=true)分支未专门删旧会话

被踢 token 入黑名单后,列表查询的黑名单检查(`OPS_CACHE.Get`)会过滤并清理,靠兜底即可。当前 use-multipoint=false,该分支不执行。未为假想场景加专门清理逻辑(不过度设计)。

## 验证

`go build ./...` + `go vet` 通过;前端 `pnpm typecheck` 通过。gofmt -l 列表含大量项目原有文件(项目本不严格 gofmt,常态),仅对本次新建的 `model/sys_online.go`、`router/sys_online.go` 跑 gofmt -w;`sys_user.go` 仅有原有 import 顺序问题(非本次引入),不动以免无关改动。

## 关联

- 认证基座见 [[httponly-cookie-auth]](单 x-token cookie 简化实现);jwt 黑名单见 `service/system/jwt_black_list.go`
- 个人中心其他卡片见 [[user-center-profile]](基本资料/头像/改密)、[[social-binding]](第三方应用)
- 登录设备/地点解析见 [[log-ip-location]](ParseIPLocation)、`utils/useragent.go`(ParseUserAgent)
