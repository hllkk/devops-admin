# httpOnly Cookie 认证改造设计 (HttpOnly Cookie Auth)

- **状态**：已批准（2026-07-14）
- **范围**：把本地 `server/`+`web/` 认证从「残缺的 `x-token` 头 / `Authorization` 头 / localStorage 单 token」改造为「access + refresh 双 httpOnly cookie」，对标目标主机 `172.21.10.40` 的实现，并删除其他认证方式残留
- **关联**：承接「权限基座闭环」（`2026-07-13-permission-base-closedloop-design.md`）的登录链路；本次只改认证载体与续期机制，不动 RBAC/casbin/雪花 ID 迁移

---

## 1. 背景与现状

### 1.1 架构
- 后端：Go + Gin + GORM，GVA 范式，module `github.com/hllkk/devops-admin/server`，全局前缀 `OPS_`
- 前端：SoybeanAdmin 2.x（Vue3 + Vite + TS + NaiveUI + Pinia）

### 1.2 目标主机（172.21.10.40）httpOnly cookie 方案（已提取，作为对标基准）
| 组件 | 实现 |
|---|---|
| 取 token（`utils/claims.go::GetToken`） | `Authorization: Bearer` 头优先 → `token` httpOnly cookie → **拒绝 query 参数**（防 URL 日志/Referer 泄漏） |
| `RequestIsSecure(c)` | `X-Forwarded-Proto` → `c.Request.TLS`，动态决定 `Secure` 标志（反代友好） |
| 写 cookie（`setLoginCookies`） | `token`/`refresh-token`/`username` 三 cookie，`HttpOnly=true`、`SameSite=Strict`、`Secure=动态`、`Path=/`、`Domain=""`，MaxAge 跟随 JWT 配置 |
| 双 token | access + refresh，audience 区分；`ParseAccessToken` 强制 access；业务接口拒绝 refresh |
| 刷新 | 读 `refresh-token` cookie → 轮换 access+refresh → 旧 token 入黑名单 |
| CORS | `Allow-Credentials: true` + 动态回显 Origin |
| 前端 | `withCredentials: true`；`getAuthorization()` 返回 `null`；登录响应只回 `{userInfo, expiresAt}` 不回 token；登录态信号 = `isAuthenticated`+`tokenExpiresAt`（localStorage，可读非敏感）；80% 寿命主动刷新 + 失败响应式刷新 |

### 1.3 本地项目现状（探索结论，多处断裂）

| 组件 | 现状 | 问题 |
|---|---|---|
| 前端请求适配（`web/src/service/request/index.ts`） | 发 `Authorization: Bearer <token>`，无 `withCredentials` | 与后端读取方式不一致；不带 cookie |
| 前端 token 存储（`store/modules/auth`） | `token`/`refreshToken` 存 localStorage | XSS 可读 token；httpOnly 化后需移除 |
| 后端取 token（`utils/claims.go::GetToken`） | 读 `x-token` 头 → `x-token` cookie | 与前端 `Authorization` 头对不上，**鉴权请求实际失败** |
| 后端 `SetToken/ClearToken`（`utils/claims.go`） | 半成品，写 `x-token` cookie | httpOnly 化后废弃 |
| 后端鉴权失败（`model/common/response/response.go::NoAuth`） | HTTP 401 + code `0001` | axios 走 `onError`，**无法命中前端 `EXPIRED_TOKEN_CODES` 刷新逻辑** |
| 后端滑动续期（`middleware/jwt.go`） | `new-token`/`new-expires-at` 响应头 | cookie 模型下改由 refresh 端点轮换，废弃 |
| 后端刷新/登出端点 | **不存在**（路由仅 login/code/getUserInfo） | 前端已在调 `/auth/refreshToken`、`/auth/logout`，404 |
| 后端 CORS 中间件 | `middleware.Cors()` 在 `router.go` 中是注释，且 **middleware 目录无 cors.go** | 跨域带 cookie 跑不起来，需新建 |
| 后端 token 模型（`utils/jwt.go`） | 单 token + `BufferTime` 滑动 | 需扩为 access+refresh 双 token |
| 前端登录入参（`pwd-login.vue`+`api/auth.ts`） | 发 `{username,password,clientId,grantType}` + `isEncrypt/isToken/repeatSubmit` header | 后端 `Login{Username,Password,Captcha,CaptchaId}` 不收这些，残留需清理 |

### 1.4 核心矛盾
本地登录链路前后端契约在「token 载体」上根本对不上（前端 Authorization 头 vs 后端 x-token），且刷新/登出端点缺失。本次统一到 httpOnly cookie 模型，顺带补齐端点、清理残留。

---

## 2. 目标与非目标

### 2.1 目标
- 前后端统一以 **access + refresh 双 httpOnly cookie** 承载认证，token 不进 JS
- cookie 安全属性齐备：`HttpOnly` + `SameSite=Strict` + 动态 `Secure`
- 补齐 `/auth/refreshToken`、`/auth/logout` 端点，刷新走 cookie 轮换 + 黑名单
- 鉴权失败契约对齐 Soybean 适配器（HTTP 200 + 业务 code 分流刷新/登出）
- 启用 CORS(credentials)，本地可跨域带 cookie 运行
- 删除 `x-token` 头/cookie、`new-token` 头续期、localStorage token、Authorization Bearer 头等所有旧认证残留

### 2.2 非目标（后续子项目）
- ❌ 防爆破（IP 锁 / username 递增延迟 / Redis 限流）
- ❌ 多点登录 / 在线设备追踪
- ❌ claims 绑定 ClientIP / UserAgentHash（令牌失窃检测）
- ❌ 验证码开关闭环（现有 `/auth/code` 返回 `captchaEnabled=false` 维持）
- ❌ `BaseClaims.ID` 由 `uint` 迁移 `int64`（雪花 ID 精度，boundary.md 标注待办，本次保持 `uint`）
- ❌ 单点登录（WeCom/Gitee/GitHub OAuth）闭环

---

## 3. 已确认的核心决策

| 决策 | 选择 | 说明 |
|---|---|---|
| 认证模型 | **完整双 cookie（对标目标）** | access + refresh 两 httpOnly cookie，含 refresh 轮换 + 黑名单 + 80% 主动刷新 + 失败响应式刷新 |
| SameSite | **Strict** | 目标同款；后台 SPA 无跨站首登场景 |
| 附加安全能力 | **仅基础项** | cookie 三件套 + RequestIsSecure + CORS(credentials) + 双 token；不引入防爆破/多点/ClientIP 绑定/验证码 |
| ID 迁移 | **不纳入** | 保持 `BaseClaims.ID` 为 `uint`，与雪花 ID 迁移解耦 |
| cookie 命名 | `token`（access）/ `refresh-token`（refresh） | 与目标一致；不写 `username` cookie（基础项不需要） |
| 鉴权失败契约 | **HTTP 200 + 业务 code** | access 失败→`9999`（EXPIRED_TOKEN_CODES，前端刷新）；refresh 失败→`8888`（LOGOUT_CODES，前端登出）；refresh 端点禁止返回 `9999`（防死循环） |
| 登录响应体 | `{ expiresAt }`（毫秒） | 不回 token；前端靠 cookie + `isAuthenticated` 信号 |
| `BufferTime` 配置 | **保留字段、废弃逻辑** | 移除滑动续期逻辑，config 字段标注废弃避免破坏现有配置加载 |

---

## 4. 后端设计（`server/`）

### 4.1 claims 与双 token
- `model/system/request/jwt.go`：`CustomClaims` **不新增字段**，复用内嵌 `jwt.RegisteredClaims.Audience` 区分 access/refresh。
- `utils/jwt.go`：
  - 新增 `CreateAccessToken(claims)`（Audience=`["access"]`）、`CreateRefreshToken(claims)`（Audience=`["refresh"]`，ExpiresAt 用 `RefreshExTime`）
  - 新增 `ParseAccessToken(t)`：解析 + 校验 Audience 含 `access`，否则报错（业务接口拒绝 refresh token）
  - 新增 `ParseRefreshToken(t)`：解析 + 校验 Audience 含 `refresh`
  - 新增 `JoinBlacklist(t)` / `IsBlacklisted(t)`：复用 `global.BlackCache`
  - 保留 `CreateClaims`（填 BaseClaims + RegisteredClaims，Audience 由 Create* 方法覆盖）
- `config/jwt.go`：新增 `RefreshExTime string`（mapstructure `refresh-ex-time`），默认 `168h`。`config.yaml` 同步。
- 黑名单持久化：基础项用内存 `BlackCache` 即可（进程重启失效，可接受；后续若需持久化可落库，属非目标）。

### 4.2 取 token 与 cookie 工具（重写 `utils/claims.go` + 新增 `utils/cookie.go`）
- `GetToken(c) (string, error)`：
  1. `Authorization` 头，校验 `Bearer ` 前缀（大小写不敏感），返回裸 token
  2. 否则 `c.Cookie("token")`
  3. 都没有 → 返回 error（**禁止 query 参数**）
- `RequestIsSecure(c) bool`：`X-Forwarded-Proto == https` → `c.Request.TLS != nil`
- `GetClaims(c)`：`GetToken` → `ParseAccessToken`（强制 access）
- `LoginToken(user)`：返回 access、refresh 两串（audience 区分）
- `utils/cookie.go` 新增：
  - `SetLoginCookies(c, access, refresh)`：`c.SetSameSite(http.SameSiteStrictMode)` 后 `c.SetCookie("token", access, accessMaxAge, "/", "", secure, true)` 与 `c.SetCookie("refresh-token", refresh, refreshMaxAge, "/", "", secure, true)`；MaxAge 分别取 `JWT.ExpiresTime` / `JWT.RefreshExTime`
  - `ClearLoginCookies(c)`：同 SameSite/Secure，MaxAge `-1` 清两 cookie
- **删除**：旧 `SetToken` / `ClearToken`（x-token cookie 半成品）。`GetUserID`/`GetUserName`/`GetUserInfo` 等基于 `c.Get("claims")` 的助手保留，内部改走新 `GetClaims`。

### 4.3 鉴权失败契约（`model/common/response/response.go`）
- 新增 `NoAuthWithCode(code, message string, c)`：HTTP 200 + 业务 code（沿用 `Result`）。
- 旧 `NoAuth`（HTTP 401）保留但不再在认证链路使用（避免破坏潜在调用，标注「认证链路请用 NoAuthWithCode」）。

### 4.4 中间件
- `middleware/jwt.go` 重写 `JWTAuth()`：
  1. `GetToken(c)`，空 → `NoAuthWithCode("9999", "未登录或令牌失效", c)` + Abort
  2. `IsBlacklisted` → `NoAuthWithCode("9999", ...)` + Abort
  3. `ParseAccessToken`，过期/无效 → `NoAuthWithCode("9999", ...)` + Abort
  4. `c.Set("claims", claims)` → `c.Next()`
  - **删除** `new-token`/`new-expires-at` 头滑动续期整段、`SetRedisJWT`（多点）调用。
- 新建 `middleware/cors.go`：
  - `Cors()`：动态回显 `Access-Control-Allow-Origin`（取请求 Origin）、`Allow-Credentials: true`、`Allow-Headers`（含 Authorization/Content-Type）、`Allow-Methods`、`Expose-Headers`（New-Token/New-Expires-At/Download-Filename）、放行 OPTIONS（204）
- `initialize/router.go`：`Router.Use(middleware.Cors())`（public+private 之前，全局）。

### 4.5 API 与路由
- `api/v1/system/sys_user.go`：
  - `Login`：`ShouldBindJSON` + `Verify` → `userService.Login(username,password)`（返回 access,refresh,user,err）→ `SetLoginCookies(c, access, refresh)` → `OkWithDetailed(gin.H{"expiresAt": <毫秒}, "登录成功", c)`。**响应体不含 token**。
  - 新增 `RefreshToken`：读 `refresh-token` cookie；空 → `NoAuthWithCode("8888", ...)`。`ParseRefreshToken`；黑名单 → `8888`。签发新 access+refresh（基于 refresh claims 的 BaseClaims）→ 旧 refresh（与旧 access 若可解析）入黑名单 → `SetLoginCookies` → `OkWithDetailed({expiresAt})`。refresh 端点**任何失败都返回 `8888`**，绝不返回 `9999`。
  - 新增 `Logout`：`ClearLoginCookies(c)`；若 `GetToken` 能取到 access 则入黑名单 → `OkWithMessage("退出成功", c)`。
- `service/system/sys_user.go::Login`：返回值由 `(token, refreshToken, user, err)` 改为 `(access, refresh, user, err)`，内部用 `CreateAccessToken`/`CreateRefreshToken` 签发两串（不再 `refreshToken = token`）。
- 路由 `router/system/sys_base.go`：
  - public：`POST /auth/login`、`POST /auth/code`、`POST /auth/refreshToken`、`POST /auth/logout`
  - private：`GET /auth/getUserInfo`

### 4.6 删除清单（后端）
- `utils/claims.go`：`SetToken` / `ClearToken`
- `middleware/jwt.go`：`new-token` / `new-expires-at` 头续期逻辑、`SetRedisJWT` 调用
- `utils/jwt.go`：`BufferTime` 相关滑动续期用法（`CreateClaims` 仍可保留 BufferTime 字段填充，但中间件不再消费）
- `service Login` 的 `refreshToken = token` 占位逻辑

---

## 5. 前端设计（`web/`）

### 5.1 请求适配（`src/service/request/index.ts`）
- `createFlatRequest` 配置加 `withCredentials: true`
- `onRequest`：`const Authorization = getAuthorization();` 为 `null` 时**不设头**（避免 axios 发字面量 `"null"`）
- `onBackendFail` 内刷新重试段同步：刷新后 `getAuthorization()` 仍为 null，直接 `instance.request(response.config)`
- `demoRequest`：示例用，保持现状（不在主认证链路，不强行清理以免扩大改动）

### 5.2 请求共享（`src/service/request/shared.ts`）
- `getAuthorization(): null` → 恒返回 `null`
- `handleRefreshToken()`：调 `fetchRefreshToken()`（无参，cookie 带 refresh）→ 成功：`localStg.set('tokenExpiresAt', data.expiresAt)` + `scheduleProactiveRefresh(data.expiresAt)`；失败：`resetStore('session_expired')`
- 新增 `scheduleProactiveRefresh(expiresAtMs)`：80% 寿命后定时调 `fetchRefreshToken`；幂等（同 expiresAt 不重排）；新增 `clearProactiveRefreshTimer()`

### 5.3 auth store（`src/store/modules/auth/`）
- `shared.ts`：
  - `getToken()`：`return localStg.get('isAuthenticated') ? 'authenticated' : ''`
  - `clearAuthStorage()`：`localStg.remove('isAuthenticated')` + `localStg.remove('tokenExpiresAt')`
- `index.ts`：
  - `login(form)`：`fetchLogin`（响应 `{expiresAt}`）→ 成功：`localStg.set('isAuthenticated', true)`、`token.value='authenticated'`、`storeTokenExpiry(expiresAt)`（存 `tokenExpiresAt` + 排主动刷新）→ `getUserInfo()` → 重定向。失败 `resetStore()`
  - **移除** `localStg.set('token'/'refreshToken')`、`loginByToken` 中对 `loginToken.token` 的消费
  - `resetStore()`：`clearProactiveRefreshTimer()` + `localStg.remove('tokenExpiresAt')` + `fetchLogout()`(容错) + `clearAuthStorage()` + 跳登录
  - `initUserInfo()`：`getToken()`（isAuthenticated）→ 有则 `getUserInfo()`，失败 `resetStore()`
  - 暴露接口保持 `login/logout/resetStore/initUserInfo/isLogin/...`，`login` 形参收敛为 `PwdLoginForm`

### 5.4 auth api（`src/service/api/auth.ts`）
- `fetchLogin(data)`：响应类型 `Api.Auth.LoginToken`；去掉 `isEncrypt/isToken/repeatSubmit` header（后端无对应中间件）；入参 `{username, password}`（去掉 `clientId/grantType`）
- `fetchRefreshToken()`：`POST /auth/refreshToken`，`data: {}`（cookie 带 refresh）
- `fetchLogout()`：`POST /auth/logout`（保留）
- `fetchGetUserInfo()`：保留
- `fetchRegister` / `fetchSocialLoginCallback` / `fetchCustomBackendError`：保留文件内但不接入主链路（非目标，不强行删以免触动 UI）

### 5.5 类型（`src/typings/` 或 d.ts）
- `Api.Auth.LoginToken` 改为 `{ expiresAt: number }`
- `Api.Auth.PwdLoginForm` 收敛为 `{ username: string; password: string }`（`code/uuid/captchaToken/clientId/grantType` 移除或标可选；本期验证码关闭）
- 检查并更新所有引用处（`store/modules/auth`、`pwd-login.vue`）

### 5.6 删除清单（前端）
- localStorage `token` / `refreshToken` 读写
- `getAuthorization()` 的 `Bearer ${token}` 逻辑
- 登录响应体 `token`/`refreshToken` 字段消费
- `clientId`/`grantType` 入参与 `isEncrypt`/`isToken`/`repeatSubmit` header

---

## 6. 数据流

```
登录:
  FE POST /auth/login {username,password}
  → BE 校验 → userService.Login → SetLoginCookies(access,refresh)
  → 200 {code:"0000", data:{expiresAt}}
  → FE: isAuthenticated=true, tokenExpiresAt=expiresAt, scheduleProactiveRefresh, getUserInfo

鉴权请求:
  FE withCredentials 自动带 token cookie
  → BE GetToken(cookie) → ParseAccessToken(强制 access) → c.Set("claims")
  → 正常处理

access 过期(响应式刷新):
  BE 返回 code "9999"(HTTP 200)
  → FE handleExpiredRequest → POST /auth/refreshToken (cookie 带 refresh)
  → BE ParseRefreshToken → 签发新 access+refresh → 旧 token 黑名单 → SetLoginCookies
  → 200 {expiresAt} → FE 更新 tokenExpiresAt → 重试原请求

主动刷新:
  80% 寿命定时器 → fetchRefreshToken → 更新 tokenExpiresAt + 重排

登出:
  FE POST /auth/logout → BE ClearLoginCookies + access 入黑名单
  → FE clearProactiveRefreshTimer + clearAuthStorage + 跳登录
```

---

## 7. 安全自检

| 项 | 措施 |
|---|---|
| XSS 盗 token | token 仅存 httpOnly cookie，JS 不可读 |
| CSRF | SameSite=Strict（跨站请求不带 cookie）+ 业务接口强制 access token |
| 中间人 | 动态 Secure（HTTPS 链路才下发 secure cookie） |
| URL 泄漏 | GetToken 拒绝 query 参数 |
| refresh 死循环 | refresh 端点失败一律返回 `8888`（LOGOUT），不返回 `9999`（EXPIRED） |
| 旧 token 复用 | refresh 轮换时旧 access+refresh 入黑名单 |
| 跨域 | CORS credentials + 动态 origin 回显（生产建议收口白名单，属部署侧） |

---

## 8. 测试

### 8.1 后端
- 登录：成功响应含 `Set-Cookie: token=...; HttpOnly; SameSite=Strict`、`refresh-token=...`，响应体无 token
- 鉴权：带 cookie 访问 private 接口 200；不带 401-code-9999；带 refresh token 访问业务接口被拒
- refresh：带有效 refresh-token cookie → 200 + 新 cookie；旧 refresh 重放 → 8888；refresh 过期 → 8888
- logout：清两 cookie + 旧 access 黑名单
- CORS：预检 OPTIONS 204 + `Allow-Credentials: true`

### 8.2 前端
- 登录后刷新页面仍登录态（isAuthenticated + getUserInfo 通过）
- access 过期：业务请求触发 9999 → 自动刷新重试成功
- 主动刷新：定时器到期触发 refresh
- 登出：状态清空、cookie 清除、跳登录页
- 跨域：dev proxy / 不同 origin 下 cookie 正确携带（withCredentials）

---

## 9. 完成前检查（对齐 boundary.md）

1. 后端响应结构仍满足 `{code,data,msg}`，code 为字符串
2. 鉴权失败 code 与前端 `.env` 的 `EXPIRED_TOKEN_CODES`(9999,9998,3333) / `LOGOUT_CODES`(8888,8889) 对齐
3. 登录响应字段 `expiresAt` 前后端类型一致（number，毫秒）
4. cookie 契约（名称/属性）写入 `aiDoc/frontend-backend/boundary.md`
5. Swagger 注释与真实行为一致（login 不再返回 token、新增 refreshToken/logout 注释）
