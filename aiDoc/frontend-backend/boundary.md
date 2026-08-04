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
- **请求入参结构体同样要带 `,string`**：`model/.../request/` 下接收实体主键 ID 的入参结构体（如 `TriggerTimedTask.ID`、`ToggleTimedTask.ID`），json tag 必须同样写 `json:"...,string"`，与模型主键的传输格式对齐。漏写时前端按字符串传 ID 会被 `encoding/json` 拒绝：`json: cannot unmarshal string into Go struct field ... of type uint/int64`。主键的字符串传输是**全链路约定**（模型 + 请求入参 + 响应），不只作用于模型结构体本身。
- **前端约束**：所有 ID 字段一律按 `string` 收发，**禁止当 number 做数值运算或比较**；需要排序/比较时转 BigInt。
- **显式 ID 不被覆盖**：创建时若已指定主键，回调不会覆盖（雪花落地后生效）。

> 例外：基座认证链路（`BaseClaims.ID` / `Login.GetUserId` / `GetById.ID` 等）目前仍是独立的 `uint`/`int`，待登录链路重建时统一处理（届时 claims 改用 string 存储 ID 解决精度问题）。

## 时间字段契约

- 后端所有时间字段保持原生 `time.Time`（公共基座 `OPS_AUDIT_MODEL` 的 `CreatedAt`/`UpdatedAt`、各业务 `XxxTime`），JSON 序列化为 RFC3339Nano（如 `2026-07-19T21:25:53.071037-04:00`），**保留时区与精度**。
- **禁止改全局时间序列化**：不要给 `time.Time` 加自定义 `MarshalJSON` 输出 `2026-07-19 21:25:53` 这类友好字符串——会丢时区/精度、影响所有表所有消费方、还干扰前端 `NTime` 对 ISO 的解析，属于"为局部展示问题改全局契约"。
- 时间展示格式化是**前端职责**：前端用 `NTime` 把 ISO 转成人可读格式，见 `frontend-rules.md`「时间展示格式化」。后端契约保持机器友好的 ISO 不变。

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
- 验证码（双引擎，配置 `captcha.type`：`image`=传统图形验证码 base64Captcha / `click|slide|rotate`=go-captcha 行为验证码）：总开关 `SysSecurityConfig.CaptchaEnabled`（false 则 `/auth/captcha` 永远返回 captchaEnabled=false）；启用后按触发策略决定是否要求，阈值复用 `CaptchaOpen`（`0`=每次都要 / `N`=失败N次后触发）、窗口复用 `CaptchaTimeout`、命中容差 `CaptchaTolerance`（click/slide 像素、rotate 角度）；`KeyLong` 同时作 image 验证码长度与 click 文字点选字符数。`GET /auth/captcha?username=` 返回 `{captchaEnabled,type,captchaId,masterImage,tileImage,thumbImage,thumbX,thumbY,thumbWidth,thumbHeight,angle,thumbSize}`；`captchaEnabled=false` 表示当前无需验证码。`POST /auth/login` 请求体携带 `captchaId` + `captcha`（用户答案 JSON：click `[{x,y}]` / slide `{x,y}` / rotate `{angle}` / image 文本），后端**先于密码校验**并一次性消费（校验即删、TTL 过期）。答案与失败计数统一存 `global.OPS_CACHE`（Redis 优先、降级 memory，key 前缀 `gocaptcha:`）。验证码子系统已重建（2026-07-18），login 校验接入随 [[httponly-cookie-auth]]。
- 前端：所有请求 `withCredentials: true`（**待落地**：当前 axios 实例未配置，靠 `VITE_HTTP_PROXY` 同源代理转发 cookie）；`getAuthorization()` 返回 `null`（已实现）；登录态信号 `isAuthenticated`+`tokenExpiresAt`（localStorage，可读非敏感）。
- CORS：`middleware.Cors()` 回显 Origin + `Allow-Credentials: true`。

## 动态路由（/route/*）

前端 Soybean 支持 `VITE_AUTH_ROUTE_MODE` 静态/动态两态（`web/.env` 默认 `static`）。动态模式下前端从后端 `/route/*` 取路由，后端把 `SysMenu`（RuoYi 契约）**实时转换**为前端 `Api.Route.MenuRoute`（即 Elegant Router 的 `ElegantConstRoute` + `id`）下发，存储层（`sys_menu`）不改。

- `GET /route/getConstantRoutes` → `MenuRoute[]`：**公开（挂 PublicGroup，登录前可调——constant 路由含登录页本身，前端 guard 在未登录阶段即调用）**。返回**空数组**，dynamic 模式下 constant 路由（login/404/init/user-center/iframe-page 等 `_builtin`）由前端静态生成 fallback，后端不下发。`getUserRoutes`/`isRouteExist` 挂 PrivateGroup（需登录态）。
- `GET /route/getUserRoutes` → `UserRoute { routes: MenuRoute[], home }`：按当前用户角色过滤 `sys_menu`（`menuType` 取 `M`/`C`，`F` 按钮不进路由、仅走 `userInfo.permissions`）。超管（任一角色 `SuperAdmin`）返回全部；普通用户按 角色→`sys_role_menu`→`sys_menu` 过滤，并向上回溯补齐祖先目录（防授权子菜单时父目录缺失致树断裂）。`home` 取 `admin`（对齐 `VITE_ROUTE_HOME`），无权时取第一个顶层路由 key。
- `GET /route/isRouteExist?routeName=` → `bool`：路由名是否在当前用户有权路由名集合中（前端守卫用）。

**转换规则（`RouteService`，`server/service/system/sys_route.go`）**：

| SysMenu | MenuRoute |
|---|---|
| `Path`（`system/user`） | `name`=`routeKey(Path)`（去首尾斜杠、`/`→`_` → `system_user`）；`path`=`"/"+Path`（`/system/user`） |
| `MenuName`（`route.system_user`） | `meta.i18nKey`（显示走 i18n）；`meta.title`=`routeKey`（兜底） |
| **`Component`** | **布局外壳**：`resolveLayout` 提取——含 `layout.blank`→`blank`、其余（`Layout`/`xxx/index`/`FrameView`/空）→`base`。菜单管理“布局”单选（默认/空白）把布局编码进 component |
| `MenuType=M` 有子 / `MenuType=M` 顶层无子 | 多级目录 `component=layout.<layout>` / 顶层无子按**单级** `layout.<layout>$view.<key>`（按 children 有无判定，不盲信 menuType） |
| `MenuType=C` 顶层无子 / 子级 | 单级 `layout.<layout>$view.<key>` / 子级 `view.<key>`（子级无外壳，布局不生效，布局跟随父目录） |

> **布局与 module 解耦**：`Component` 决定布局外壳（base/blank），`module` 仅做菜单隔离与角色授权树分组，不再触发布局。component 里的 view 路径段（录入的 views 目录下划线化）**被丢弃**，view 段始终由 `routeKey(Path)` 规范化——避免与构建期 RouteKey 不一致致 transform 静默丢路由。
| `Icon`=`mdi:xxx` 等 | `meta.icon` |
| `Icon`=`local-icon-<n>` | `meta.localIcon`=`menu-<n>` |
| `OrderNum` | `meta.order` |
| `Visible=1` | `meta.hideInMenu=true` |
| `IsCache=0` | `meta.keepAlive=true` |
| `IsFrame=0` | `meta.href`=规范化 Path（外链） |

> `name`/`view.<key>` 必须命中前端 Elegant Router 构建期生成的 `RouteKey`/`views` map（`web/src/router/elegant/imports.ts`），否则 `transform.ts` 解析组件时静默丢路由。新增页面时确保 `sys_menu.path` 对应的 `routeKey` 与 `views/` 目录生成的 key 一致。

## 完成前检查

跨前后端改动结束前，至少确认以下几点：

1. 后端响应结构仍然满足前端预期
2. 前端仍在使用正确的字段名和数据类型
3. 若契约发生了长期变化，对应说明已经补到 `aiDoc/`
