# 三步初始化向导（DB → Redis → 管理员密码）设计

- **状态**：待实现（2026-07-13）
- **范围**：把现有「单步合并初始化」改造为「须知 + 3 步」向导，新增 Redis 为第 2 步，DB/Redis 各提供「测试连接」按钮
- **关联**：前端 `web/src/views/_builtin/init/*`、后端 `server/{api,router,service,model}/system/sys_init*` 与 `server/initialize/redis.go`；承接已完成的「权限基座闭环」（初始化向导是其入口）

---

## 1. 背景与现状（探索结论）

### 1.1 现有初始化向导——已联通，但是「单步合并」

| 组件 | 现状 |
|---|---|
| 前端页面 `web/src/views/_builtin/init/index.vue` | ✅ 2 屏：①初始化须知 ②单表单（管理员密码 + DB 配置合并）。提交 `fetchInitDB(model)` → 成功后 `resetSystemInitCheck()` + 跳登录 |
| 前端类型 `web/src/typings/api/init.d.ts` | ✅ `InitDBForm{adminPassword,dbType,host,port,userName,password,dbName,dbPath?,template?}`，**无 Redis 字段** |
| 前端服务 `web/src/service/api/init.ts` | ✅ `fetchCheckDB()` / `fetchInitDB(data)` |
| 前端守卫 `web/src/router/guard/route.ts` | ✅ `ensureInitChecked()` 调 `/init/checkdb`，`needInit` 时强制跳 `/init` |
| 后端路由 `server/router/system/sys_init.go` | ✅ PublicGroup 挂 `POST init/initdb`、`POST init/checkdb` |
| 后端 API `server/api/v1/system/sys_init.go` | ✅ `DBApi.InitDB`（`OPS_DB!=nil` 时拒绝）/ `DBApi.CheckDB`（`needInit = OPS_DB==nil`） |
| 后端编排 `server/service/system/sys_init.go` | ✅ `InitDBService.InitDB`：EnsureDB → InitTables → InitData → WriteConfig；管理员密码经 `ctx.WithValue("adminPassword",…)` 注入 seed |
| 请求体 `server/model/system/request/sys_init.go` | ✅ `InitDB{AdminPassword(required),DBType,Host,Port,UserName,Password,DBName(required),DBPath,Template}`，**无 Redis 字段** |
| WriteConfig（`sys_init_{mysql,pgsql,sqlite,mssql}.go`） | ⚠️ 每个 handler 各自 `StructToMap(global.OPS_CONFIG)` 全量 `viper.Set` + `WriteConfig()`；只写 `Mysql/…` + `System.DbType` + `JWT.SigningKey`，**不碰 Redis** |

### 1.2 Redis 现状——基础设施在，但与向导脱节

| 组件 | 现状 |
|---|---|
| `server/config/redis.go` | ✅ `Redis{Name,Addr,Password,DB,UseCluster,ClusterAddrs}`，**无连接池字段** |
| `server/initialize/redis.go` | ✅ `Redis()`/`RedisList()`：`redis.NewClient`/`NewClusterClient` → `Ping` → 写 `OPS_REDIS`；ping 失败 `panic` |
| `server/global/global.go` | ✅ `OPS_REDIS` / `OPS_REDISList` / `GetRedis(name)` |
| `server/core/server.go` | ⚠️ Redis 初始化被 `if System.UseRedis` 门控，**不在 `initializeSystem()` 内** |
| `server/config.yaml` / `config.docker.yaml` | ⚠️ `system.use-redis: false`；`redis:` 段已预填默认值（`127.0.0.1:6379`） |
| `go.mod` | ✅ `github.com/redis/go-redis/v9 v9.21.0` |
| 向导 ↔ Redis | ❌ `request.InitDB` 无 Redis 字段；`WriteConfig` 不写 Redis；向导无法配置/启用 Redis |

### 1.3 核心矛盾

向导把「DB 配置」与「管理员密码」塞进同一个表单一次性提交，且完全没有 Redis 配置入口。需要拆成 3 步、补上 Redis 步、并给 DB/Redis 加「测试连接」——同时不破坏现有原子初始化范式与 `checkdb` 守卫语义。

---

## 2. 目标与非目标

### 2.1 目标
1. 向导改为「须知 → ①数据库 → ②Redis → ③管理员密码」4 屏流程，末步一次性提交。
2. DB、Redis 两步各提供「测试连接」按钮，调独立 ping 接口（ephemeral，不落库、不建库）。
3. Redis 作为必填步骤接入：提交后 `config.Redis` 与 `system.use-redis:true` 落盘，当前进程即时连上 `OPS_REDIS`，重启后由 `RunServer` 自动连接。

### 2.2 非目标（后续）
- ❌ Redis 集群模式 / 连接池调优（向导只暴露单实例 Addr/Password/DB；集群与池参数仍走手动改 `config.yaml`）
- ❌ Redis 在业务层的实际消费（captcha store、token 黑名单等由后续模块接入）
- ❌ 向导「测试连接」之外的任何 DB 健康检查 / 监控
- ❌ 已初始化状态下重新配置 DB/Redis 的入口（本期 ping 端点在 `OPS_DB!=nil` 时直接拒绝）

---

## 3. 已确认的核心决策

| 决策 | 选择 | 说明 |
|---|---|---|
| 提交与持久化模型 | **末步一次性提交** | 三步仅在前端收集数据；测试连接 ephemeral；最后一步调增强后的 `/init/initdb` 原子完成建库+建表+种子+回写 config。中途放弃无半成品，改动最聚焦 |
| Redis 是否必填 | **必填** | 步骤 2 必须填并通过测试才能进入步骤 3；提交后 `use-redis:true` 始终启用 |
| Redis 表单字段 | **单实例三字段**：Addr/Password/DB（默认 `127.0.0.1:6379` / 空 / `0`） | 集群留手动配置；`config.Redis` 结构体不改 |
| 须知页 | **保留**为独立首屏 | 进入后点「确认」再进入 3 步 stepper（即「须知 → 步骤1 → 步骤2 → 步骤3」4 屏） |
| DB 测试语义 | **只 ping 不建库** | 连 DB 服务器（不含目标库名）ping；sqlite 校验父目录可写；建库仍在末步 `EnsureDB` |
| 测试与「下一步」关系 | **测试通过才能「下一步」** | 「下一步/完成」在对应测试通过前 disabled；改字段后失效「已测」标记需重测 |
| Redis 配置落盘方式 | **编排器前置写 `OPS_CONFIG`** | 在 `WriteConfig` 前给 `OPS_CONFIG.Redis` / `System.UseRedis` 赋值，复用各 handler 已有的全量回写，**无需改 4 个 per-DB handler** |
| ping 端点安全 | **仅 `OPS_DB==nil` 可用** | 已初始化后拒绝，防 SSRF 式探测滥用；与 `InitDB` 自身守卫一致 |
| ping 路由命名 | `POST init/db/ping`、`POST init/redis/ping` | 与 `initdb/checkdb` 同组，语义清晰 |
| 连接池字段 | **不加** | `config.Redis` 不扩字段；向导不暴露、当前无消费方（YAGNI） |

---

## 4. 后端改动

### 4.1 `server/model/system/request/sys_init.go`
- `InitDB` 增加 3 字段：
  ```go
  RedisAddr     string `json:"redisAddr"`     // Redis 地址 host:port
  RedisPassword string `json:"redisPassword"` // Redis 密码（可空）
  RedisDB       int    `json:"redisDB"`       // Redis 库号（默认 0）
  ```
- 新增方法：
  ```go
  func (i *InitDB) ToRedisConfig() config.Redis {
      return config.Redis{
          Name:   "default",
          Addr:   i.RedisAddr,
          Password: i.RedisPassword,
          DB:     i.RedisDB,
      }
  }
  ```
- 新增两个 ping 请求体（**不复用 `InitDB`**——其 `AdminPassword/DBName` 带 `binding:"required"`，会卡住步骤 1 的测试）：
  ```go
  // DBConnTest 数据库连接测试（不建库）
  type DBConnTest struct {
      DBType   string `json:"dbType"`
      Host     string `json:"host"`
      Port     string `json:"port"`
      UserName string `json:"userName"`
      Password string `json:"password"`
      DBName   string `json:"dbName"`
      DBPath   string `json:"dbPath"`   // sqlite
      Template string `json:"template"` // pgsql
  }

  // PingRedis Redis 连接测试
  type PingRedis struct {
      Addr     string `json:"addr"`
      Password string `json:"password"`
      DB       int    `json:"db"`
  }
  ```
  > `DBConnTest` 与 `InitDB` 的 DB 字段重叠，是有意为之：前者无 required 绑定、无 admin/redis 字段，服务于「只测连接」语义。`*EmptyDsn()` 等辅助方法仍挂在 `InitDB` 上；`PingDB` service 内部构造一个临时 `InitDB`（或抽公共 dsn 函数）复用 dsn 逻辑，避免重复实现。

### 4.2 新文件 `server/service/system/sys_init_conn.go`（连接测试，纯业务、不碰 global）
- `PingDB(conf request.DBConnTest) error`
  - 复用现有 `*EmptyDsn()` + `sql.Open` + `Ping` + `Close` 范式（参考 `sys_init.go:createDatabase` 的 ping 段）；
  - **只 ping，不执行 `CREATE DATABASE`**；
  - sqlite 分支：**无副作用**地校验——`dbPath` 父目录存在且可写（`os.Stat` 父目录 + 在父目录下创建临时文件再删除，探测写权限）；**不** `sql.Open` 目标 `.db` 文件（避免提前建库文件），建文件仍留给末步 `EnsureDB`；
  - 失败返回带可读原因的 error（如 `dial tcp ...:3306: connect: connection refused`），由 API 层透出。
- `PingRedis(conf request.PingRedis) error`
  - `redis.NewClient(&redis.Options{Addr,Password,DB})` → `Ping(ctx)` → `Close`；
  - 把 `initialize/redis.go:initRedisClient` 里的 ping 逻辑抽成一个**不写 global** 的纯函数（如 `DialRedis(cfg) (redis.UniversalClient, error)`），`PingRedis` 与 `initialize.Redis()` 共用，避免两份 ping 代码。

### 4.3 `server/api/v1/system/sys_init.go` —— `DBApi` 新增两个 handler
```go
// PingDB 测试数据库连接（仅在系统未初始化时可用）
// @Tags     SysInit
// @Summary  测试数据库连接
// @Param    data  body  request.DBConnTest  true  "数据库连接参数"
// @Success  200   {object}  response.Response{data=string}
// @Router   /init/db/ping [post]
func (i *DBApi) PingDB(c *gin.Context) { ... }

// PingRedis 测试 Redis 连接（仅在系统未初始化时可用）
// @Router /init/redis/ping [post]
func (i *DBApi) PingRedis(c *gin.Context) { ... }
```
两者统一前置守卫：
```go
if global.OPS_DB != nil {
    response.FailWithMessage("系统已初始化，无需测试连接", c)
    return
}
```
绑定 → 调 service → `response.OkWithMessage("连接成功", c)` / `FailWithMessage(err.Error(), c)`。Swagger 注释完整准确（功能/参数/响应/路由/无需鉴权）。

### 4.4 `server/router/system/sys_init.go`
```go
initRouter.POST("db/ping", dbApi.PingDB)       // 测试数据库连接
initRouter.POST("redis/ping", dbApi.PingRedis) // 测试 Redis 连接
```

### 4.5 `server/service/system/sys_init.go` —— 编排器 `InitDB()` 改动
在 `initHandler.WriteConfig(ctx)` **之前**插入：
```go
global.OPS_CONFIG.Redis = conf.ToRedisConfig()
global.OPS_CONFIG.System.UseRedis = true
```
> 原理：每个 per-DB handler 的 `WriteConfig` 都执行 `StructToMap(global.OPS_CONFIG)` 全量 `viper.Set` + `WriteConfig()`，故 Redis 段与 `use-redis` 会随本次回写自动落盘，**4 个 handler 一行不改**。

`WriteConfig` 之后，给当前进程连接 Redis（免 reload 即可用，对齐编排器里 `global.OPS_DB = db` 的即时注入范式）：
```go
// 复用抽出的 DialRedis；ping 已在向导测过，此处失败仅记日志不 panic
if client, err := initialize.DialRedis(global.OPS_CONFIG.Redis); err == nil {
    global.OPS_REDIS = client
} else {
    global.OPS_LOG.Warn("init 后即时连接 Redis 失败，重启后将自动重试", zap.Error(err))
}
```
> 不直接调 `initialize.Redis()`：它在 ping 失败时 `panic`，向导流程里不应因 Redis 抖动而崩；改为 guarded 注入，重启后由 `RunServer` 兜底。

### 4.6 `server/initialize/redis.go` —— 抽公共 dial
新增导出函数：
```go
// DialRedis 按 cfg 建客户端并 Ping，不写 global。ping 失败返回 error（不 panic）。
func DialRedis(redisCfg config.Redis) (redis.UniversalClient, error)
```
`Redis()` 改为：`client, err := DialRedis(...); if err != nil { panic(err) }; global.OPS_REDIS = client`（行为不变，只是把 ping 逻辑抽走复用）。

### 4.7 不改
- `config/redis.go`、`config/system.go`（`UseRedis` 已在）、`config/config.go`
- `core/server.go`（重启后 `RunServer` 自然走 Redis 初始化）
- 4 个 per-DB `WriteConfig` handler

---

## 5. 前端改动

### 5.1 `web/src/typings/api/init.d.ts`
- `InitDBForm` 增 3 字段：`redisAddr: string`、`redisPassword: string`、`redisDB: number`。
- 新增：
  ```ts
  interface PingDBForm { dbType: DBType; host?: string; port?: string; userName?: string; password?: string; dbName: string; dbPath?: string; template?: string; }
  interface PingRedisForm { addr: string; password?: string; db: number; }
  ```
- 注释更新为「字段与后端 `request.InitDB` 对齐（含 Redis）」。

### 5.2 `web/src/service/api/init.ts`
```ts
export function fetchPingDB(data: Api.Init.PingDBForm) {
  return request<string>({ url: '/init/db/ping', method: 'post', data });
}
export function fetchPingRedis(data: Api.Init.PingRedisForm) {
  return request<string>({ url: '/init/redis/ping', method: 'post', data });
}
```

### 5.3 `web/src/views/_builtin/init/index.vue` 重构
- 顶层状态机：`stage: 'notice' | 'wizard'`；`wizard` 内 `currentStep: 1|2|3`。
- **须知屏**：保留现有「初始化须知」整屏与样式，「确认」按钮 → `stage='wizard'`，「返回」→ 跳登录（沿用 `handleBack`）。
- **Stepper（NaiveUI `NSteps`，3 步）**：顶部常驻步骤指示。
- 一个响应式 `model: InitDBForm`（含 DB + Redis + admin 全字段，默认值：DB 沿用现状；Redis `redisAddr='127.0.0.1:6379'`、`redisPassword=''`、`redisDB=0`）。
- **3 个独立 `NForm` ref**（`dbFormRef / redisFormRef / adminFormRef`）做分步校验，避免单表单整表校验。
- **Step 1 数据库**：`dbType` Select（切换重置连接默认值，沿用 `handleDbTypeChange`）+ `dbName` + 条件性 host/port/userName/password（sqlite→`dbPath`，pgsql→`template`）+「测试连接」按钮 +「下一步」。
- **Step 2 Redis**：`addr` + `password` + `db`（NInputNumber）+「测试连接」+「上一步 / 下一步」。
- **Step 3 管理员密码**：`adminPassword`（密码框，≥6 位校验沿用）+「上一步 / 完成」。
- **测试按钮**：4 态（idle/loading/success/error）；用 `dbTested` / `redisTested` ref 记录是否通过；**任一相关字段变更 → 清对应标记**，强制重测；`message` 提示结果。
- **「下一步/完成」disabled 条件**：步骤 1 → `!dbTested`；步骤 2 → `!redisTested`；步骤 3 「完成」→ 始终可点（提交前再校验密码长度）。
- **「下一步」交互**：先 `await 当前步 formRef.validate()` 通过，再 `currentStep++`。
- **「完成」提交**：`await adminFormRef.validate()` + 密码长度复核 → `fetchInitDB(model)` → 成功 `resetSystemInitCheck()` + `router.replace({name:'login'})`（沿用现状）；全屏 loading 沿用。
- ping 调用从 `model` 切片构造 payload（DB 切片 / Redis 切片）。

### 5.4 `web/src/locales/langs/{zh-cn,en-us}.ts`
新增 i18n key（**禁硬编码中文**）：
- `page.init.step.db / page.init.step.redis / page.init.step.admin`（步骤名）
- `page.init.form.redisAddr / redisPassword / redisDB`（含 `.placeholder`）
- `page.init.testConnection`（按钮）、`page.init.testConnectionSuccess / testConnectionFailed`
- `page.init.next / page.init.prev / page.init.finish`

### 5.5 不改
- `router/guard/route.ts`（`checkdb` 驱动不变）
- Elegant Router 生成物（路由 `init` 已存在，页面路径不变）

---

## 6. 接口契约（前后端对齐）

| 方法 | 路径 | 请求体 | 响应 `data` | 鉴权 | 可用条件 |
|---|---|---|---|---|---|
| POST | `/init/checkdb` | — | `{needInit:boolean}` | 无 | 始终 |
| POST | `/init/db/ping` | `DBConnTest` | `""` | 无 | `OPS_DB==nil` |
| POST | `/init/redis/ping` | `PingRedis` | `""` | 无 | `OPS_DB==nil` |
| POST | `/init/initdb` | `InitDB`（含 admin + DB + Redis） | `""` | 无 | `OPS_DB==nil` |

- 统一响应 `{code:"0000"|"0001", data, msg}`；ping 成功 `data` 为空串，前端只看 `error`。
- 字段命名沿用驼峰：`redisAddr/redisPassword/redisDB`（与 `adminPassword/dbName` 一致）。
- ID 类型契约不涉及（本流程无业务 ID 传输）。

---

## 7. 边界与安全
- **ping 端点防滥用**：`OPS_DB != nil` 时一律拒绝（系统已初始化），关闭「借用服务器探测任意 DB/Redis」的 SSRF 口子。
- **不破坏原子性**：DB 建库/建表/种子仍在末步 `/init/initdb` 一次完成；ping 阶段零副作用。
- **Redis 即时连接容错**：向导已 ping 测过，但编排器里二次连接失败只告警不 panic，重启由 `RunServer` 兜底。
- **中途放弃**：未点「完成」前不落任何配置；`checkdb` 仍返回 `needInit:true`，守卫继续拦在 `/init`。
- **改字段需重测**：DB/Redis 任一字段变更失效对应「已测」标记，避免「测过 A 配置、提交 B 配置」。

---

## 8. 测试

### 8.1 后端
- 新增 `server/service/system/sys_init_conn_test.go`：
  - `PingRedis`：本地 redis → 正确密码成功；错误 addr / 错误密码失败。
  - `PingDB`：mysql dsn 分支（可集成本地 mysql 或对 `sql.Open` 做最小验证）；sqlite 分支校验父目录可写 / 不可写两种。
- 集成测试（沿用 `91b584b` e2e 风格）：走完整 `/init/initdb`（带 Redis 字段）→ 断言 `config.yaml` 含 `redis:` 段且 `system.use-redis: true`，且 `global.OPS_REDIS != nil`。
- ping 端点守卫：`OPS_DB != nil` 时 `/init/db/ping`、`/init/redis/ping` 返回失败。

### 8.2 前端
- `pnpm gen-route`（页面路径未变，预计无新路由；如有组件变动再跑）。
- pre-commit 强制门：`pnpm typecheck && pnpm lint && pnpm fmt`。
- 手动 e2e：须知 → 填 DB → 测试通过 → 填 Redis → 测试通过 → 填管理员密码 → 完成 → 跳登录 → 用 super/管理员密码登录成功。

---

## 9. 项目规则动作（按 `AGENT.MD`）
1. 新增业务需求记忆 `aiDoc/memory/business/init-wizard-redis.md`（一需求一文件），并更新 `aiDoc/memory/demand-index.md`。
2. 跨端契约变更同步 `aiDoc/frontend-backend/boundary.md`：init 模块新增 `/init/db/ping`、`/init/redis/ping` 两个端点说明，`InitDBForm` 增 Redis 字段。
3. 提交信息走中文规范（`feat:` 前缀），按 `docs/superpowers/plans/` 产出实现计划后再分步提交。

---

## 10. 关键文件清单

**后端**
- 改：`server/model/system/request/sys_init.go`、`server/api/v1/system/sys_init.go`、`server/router/system/sys_init.go`、`server/service/system/sys_init.go`、`server/initialize/redis.go`
- 新：`server/service/system/sys_init_conn.go`、`server/service/system/sys_init_conn_test.go`

**前端**
- 改：`web/src/typings/api/init.d.ts`、`web/src/service/api/init.ts`、`web/src/views/_builtin/init/index.vue`、`web/src/locales/langs/{zh-cn,en-us}.ts`

**文档/记忆**
- 新：`aiDoc/memory/business/init-wizard-redis.md`
- 改：`aiDoc/memory/demand-index.md`、`aiDoc/frontend-backend/boundary.md`
