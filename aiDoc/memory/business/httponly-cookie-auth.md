# httpOnly Cookie 认证改造（HttpOnly Cookie Auth）

> 类型：认证链路需求 · 状态：设计中（spec `docs/superpowers/specs/2026-07-14-httponly-cookie-auth-design.md` 已批准，待实现）

## 需求

把本地 `server/`+`web/` 认证从「残缺的 `x-token` 头 / `Authorization` 头 / localStorage 单 token」改造为「access + refresh 双 httpOnly cookie」，对标目标主机 `172.21.10.40`（`backend/`+`frontend/` 子模块，module `devopsAdmin`），并删除其他认证方式残留。要求前后端完整且安全地支持 httpOnly cookie。

### 已确认决策
- 认证模型：**完整双 cookie（access + refresh）**，含 refresh 轮换 + 黑名单 + 80% 寿命主动刷新 + 失败响应式刷新
- SameSite：**Strict**；cookie 安全三件套 HttpOnly + 动态 Secure（`RequestIsSecure`：X-Forwarded-Proto → c.Request.TLS）
- cookie 命名：`token`（access）/ `refresh-token`（refresh）；不写 `username` cookie
- 范围：**仅基础项**（cookie 三件套 + RequestIsSecure + CORS(credentials) + 双 token）；不引入防爆破 / 多点在线 / ClientIP·UAHash 绑定 / 验证码开关闭环
- `BaseClaims.ID` 保持 `uint`，**不做** 雪花 int64 迁移（与 [[snowflake-id-generator]] 解耦）

### 关键契约
- 鉴权失败改 **HTTP 200 + 业务 code**（不再 401）：access 失败→`9999`（`VITE_SERVICE_EXPIRED_TOKEN_CODES`，前端刷新）；refresh 失败→`8888`（`VITE_SERVICE_LOGOUT_CODES`，前端登出）；refresh 端点**禁止返回 9999**（防死循环）
- 登录响应体只回 `{ expiresAt }`（毫秒），**不回 token**；前端登录态信号 = `isAuthenticated`+`tokenExpiresAt`（localStorage，可读非敏感）
- 前端 `withCredentials: true`，`getAuthorization()` 返回 `null`

## 待实现（见 spec）

- 后端：双 token（audience 区分 access/refresh）+ `ParseAccessToken`/`ParseRefreshToken` + 黑名单；重写 `utils/claims.go`（GetToken 拒 query、RequestIsSecure、LoginToken 双签）；新增 `utils/cookie.go`（SetLoginCookies/ClearLoginCookies）；重写 `middleware/jwt.go`（删 new-token 头续期）；新建 `middleware/cors.go`；改 `Login` 响应 + 新增 `/auth/refreshToken`、`/auth/logout`
- 前端：`withCredentials`、`getAuthorization→null`、`handleRefreshToken` 走 cookie、主动刷新定时器；auth store 改 `isAuthenticated` 信号；登录入参收敛 `{username,password}`、响应 `{expiresAt}`
- 删除：localStorage token、Authorization Bearer 头、x-token cookie 工具、new-token 头续期、BufferTime 滑动逻辑、`refreshToken=token` 占位

## 关联

- 承接 [[system-user-role-models]] / 权限基座闭环（`2026-07-13-permission-base-closedloop-design.md`）的登录链路
- cookie 契约需同步进 `aiDoc/frontend-backend/boundary.md`
