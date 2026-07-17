# 前后端边界说明

## 归属边界

- 后端负责路由、参数校验、业务逻辑和响应结构
- 前端负责页面流程、交互体验、本地状态和展示层
- 共同行为通过明确的 API 契约协作，不通过隐式约定耦合

## 契约规则

- 保持统一响应结构：`{ code, data, msg }`，`code` 为**字符串**（成功 `"0000"`；失败码 = `response.ERROR`，当前值为 `"7"`，凡 `!= "0000"` 前端一律视为失败）。前端请求适配器 `isBackendSuccess` 判 `String(code) === VITE_SERVICE_SUCCESS_CODE`（默认 `"0000"`，见 `web/.env`），**不是判 `code === 0`**；`onBackendFail` 按 `.env` 的 `VITE_SERVICE_LOGOUT_CODES`(8888/8889) / `MODAL_LOGOUT_CODES`(7777/7778) / `EXPIRED_TOKEN_CODES`(9999/9998/3333) 分流登出与刷新 token
- 保持统一分页结构：响应 `{ pageNum, pageSize, total, rows }`（`response.PageResult` 字段 `Rows/Total/PageNum/PageSize`）；请求分页入参为同名字段（后端 `request.PageInfo.PageNum`/`PageSize`，json tag 均为 `pageNum`/`pageSize`），即**请求与响应字段名对称**（`pageNum`/`pageSize`）
- 字段名不要随意漂移
- 前后端字段类型必须保持一致
- 后端必须提供完整而准确的 Swagger 接口说明
- 前端接口调用应以实际 Swagger 与后端实现为准

## 主键 ID 契约

**目标**：业务表整型主键统一用雪花算法生成的 `int64`，GORM `BeforeCreate` 回调 `ops:snowflake_id` 在主键为 0 时按 `PrioritizedPrimaryField` 自动填充、不覆盖显式值。主键不放在全局基座里，由各业务模型自定义列名/JSON 名（对外实体用业务命名 `userId`/`roleId`，内部表用 `id`）。

> **当前状态（2026-07-16）**：`utils/snowflake` 与 `ops:snowflake_id` 回调**尚未落地**，`OPS_DB` 赋值点未注册该回调，现阶段主键走 **DB 自增**；新建模型须显式声明主键列。雪花链路待重建（历史设计见 `aiDoc/memory/business/snowflake-id-generator.md`）。

- **传输格式**：JSON 中 ID 一律以**字符串**传输（Go 端 `json:"...,string"`，如 `json:"userId,string"` / `json:"id,string"`），规避 JS `Number` 仅能安全表示到 2^53 的精度丢失（雪花 ID 通常 18~19 位十进制）。
- **前端约束**：所有 ID 字段一律按 `string` 收发，**禁止当 number 做数值运算或比较**；需要排序/比较时转 BigInt。
- **显式 ID 不被覆盖**：创建时若已指定主键，回调不会覆盖（雪花落地后生效）。

> 例外：基座认证链路（`BaseClaims.ID` / `Login.GetUserId` / `GetById.ID` 等）目前仍是独立的 `uint`/`int`，待登录链路重建时统一处理（届时 claims 改用 string 存储 ID 解决精度问题）。

## 变更规则

- 涉及破坏性接口调整时，要先写清楚变更范围
- Swagger 或其他接口说明必须与真实实现一致
- 前端接口封装应继续放在 `web/src/service/api/` 或 `web/src/plugin/<name>/api/`
- 可复用逻辑优先复用 `web/src/utils/` 现有能力

## 初始化向导（/init/*，重构后待重建）

> 重构（`1d632d9`）后 `/init/*` 接口与对应 service/router **未保留**，以下为重构前的契约设计，重建时以此为准（历史记录见 `aiDoc/memory/business/system-init-flow.md`、`init-wizard-redis.md`）。

- `POST /init/checkdb`：返回 `{needInit}`，路由守卫用，语义 = `OPS_DB==nil`。
- `POST /init/db/ping`：请求 `DBConnTest{dbType,host,port,userName,password,dbName,dbPath,template}`，ephemeral 连接测试（**只 ping 不建库、不落盘**）；`OPS_DB!=nil`（已初始化）时拒绝。
- `POST /init/redis/ping`：请求 `PingRedis{addr,password,db}`，ephemeral 连接测试（不落盘）；`OPS_DB!=nil` 时拒绝。
- `POST /init/initdb`：请求 `InitDB`（含 `adminPassword` + DB 字段 + `redisAddr/redisPassword/redisDB`），原子完成建库+建表+种子+回写 config（含 Redis 段、`system.use-redis:true`）+ 即时连接 `OPS_REDIS`。
- 统一响应 `{code:"0000"|"7", data, msg}`（成功 `SUCCESS="0000"`、失败 `ERROR="7"`）；ping 成功 `data` 为空串，前端 flat request 只看 `error`。

## 认证（httpOnly cookie，重构后待重建）

> 重构（`1d632d9`）后 `/auth/*` 接口与对应 service/router **未保留**，以下为重构前的契约设计（access/refresh 双 cookie、go-captcha 行为验证码、鉴权失败 HTTP200+业务 code），重建时以此为准（历史记录见 `aiDoc/memory/business/httponly-cookie-auth.md`、`go-captcha-login.md`）。

- 认证载体：access + refresh 双 **httpOnly cookie**，token 不进 JS。
  - cookie 名：`token`（access）、`refresh-token`（refresh）
  - 属性：`HttpOnly=true`、`SameSite=Strict`、`Secure=RequestIsSecure(X-Forwarded-Proto→TLS)`、`Path=/`、`Domain=""`
- 取 token：后端 `utils.GetToken` 优先 `Authorization: Bearer` 头，其次 `token` cookie；**禁止从 query 取 token**。
- 鉴权失败契约：统一 **HTTP 200 + 业务 code**（不再 401）：
  - `9999`（`VITE_SERVICE_EXPIRED_TOKEN_CODES`）：access 失效，前端调 `/auth/refreshToken` 刷新并重试
  - `8888`（`VITE_SERVICE_LOGOUT_CODES`）：refresh 也失效（仅 `/auth/refreshToken` 返回），前端登出
  - `/auth/refreshToken` **禁止返回 9999**，防前端死循环刷新
- 登录响应体只回 `{ expiresAt }`（毫秒），**不回 token**。
- 端点：`GET /auth/captcha`、`POST /auth/login`、`POST /auth/refreshToken`、`POST /auth/logout`（public）；`GET /auth/getUserInfo`（private）。
- 行为验证码（go-captcha，配置 `captcha.go-captcha`）：登录按触发策略（`threshold` 阈值 / `always` / `off`）决定是否要求。`GET /auth/captcha?username=` 返回 `{captchaEnabled,type,captchaId,masterImage,tileImage,thumbImage,thumbX,thumbY,thumbWidth,thumbHeight,angle,thumbSize}`；`captchaEnabled=false` 表示当前无需验证码。`POST /auth/login` 请求体携带 `captchaId` + `captcha`（用户答案 JSON：click `[{x,y}]` / slide `{x,y}` / rotate `{angle}`），后端**先于密码校验**并一次性消费（校验即删、TTL 过期）。答案存 Redis（`gocaptcha:` 前缀），不可用降级进程内 `local_cache`。
- 前端：所有请求 `withCredentials: true`（**待落地**：当前 axios 实例未配置，靠 `VITE_HTTP_PROXY` 同源代理转发 cookie）；`getAuthorization()` 返回 `null`（已实现）；登录态信号 `isAuthenticated`+`tokenExpiresAt`（localStorage，可读非敏感）。
- CORS：`middleware.Cors()` 回显 Origin + `Allow-Credentials: true`。

## 完成前检查

跨前后端改动结束前，至少确认以下几点：

1. 后端响应结构仍然满足前端预期
2. 前端仍在使用正确的字段名和数据类型
3. 若契约发生了长期变化，对应说明已经补到 `aiDoc/`
