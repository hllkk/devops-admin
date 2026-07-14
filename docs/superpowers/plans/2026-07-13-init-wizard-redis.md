# 三步初始化向导（DB → Redis → 管理员密码）实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: 用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 按任务逐个实现。步骤用复选框（`- [ ]`）跟踪。

**Goal:** 把现有「单步合并初始化」改造为「须知 → ①数据库 → ②Redis → ③管理员密码」向导，DB/Redis 各加「测试连接」按钮，末步一次性提交，提交后 Redis 配置落盘并即时连接。

**Architecture:** 前端 4 屏（须知 + 3 步 stepper）只收集数据；DB/Redis 的测试连接调独立 ephemeral ping 接口（不落库不建库）；末步调增强后的 `POST /init/initdb` 原子完成建库+建表+种子+回写 config。Redis 配置通过「编排器前置写 `OPS_CONFIG.Redis`+`System.UseRedis`，复用 per-DB handler 的全量回写」落盘，4 个 handler 零改；即时连接经现有 `dbReadyCallback`（避免 `service/system`↔`initialize` 循环 import）。

**Tech Stack:** 后端 Go + Gin + GORM + go-redis/v9 + viper；前端 Vue3 + TS + NaiveUI（NSteps/NForm）+ `@sa/axios` + vue-i18n。

## Global Constraints

- Go module = `github.com/hllkk/devops-admin/server`，go.mod 在 `server/`；后端测试命令在 `server/` 目录下跑。
- 统一响应 `{code:"0000"|"0001", data, msg}`，`code` 是**字符串**（GVA 范式）；ping 成功 `data` 为空串，前端只看 `error`。
- Redis 只暴露**单实例三字段**（Addr/Password/DB）；**不引入**集群开关、连接池字段（YAGNI）。
- ping 端点在 `global.OPS_DB != nil`（系统已初始化）时**一律拒绝**，防 SSRF 探测滥用。
- 后端分层 `Router → API → Service → Model`，Service 不依赖 `gin.Context`；对外 API 写完整准确 Swagger。
- `service/system` 不得 import `initialize`（反向会循环）；即时连接 Redis 走 `dbReadyCallback`。
- 前端禁硬编码中文（走 i18n key `page.init.*`）；禁裸 axios（用 `@sa/axios` 的 flat request）；不要读 `node_modules/`。
- 提交信息用中文，前缀 `feat:` / `test:` / `docs:` / `refactor:`；每次任务结束提交一次。
- 前端 pre-commit 强制门：`pnpm typecheck && pnpm lint && pnpm fmt`。

---

## 文件结构

**后端**
- 改 `server/model/system/request/sys_init.go`：`InitDB` 增 Redis 三字段 + `ToRedisConfig()`；新增 `DBConnTest`、`PingRedis` 请求体。
- 改 `server/initialize/redis.go`：抽 `DialRedis(cfg) (client, error)`（不写 global、不 panic）；`Redis()` 复用。
- 新 `server/service/system/sys_init_conn.go`：`InitDBService.PingDB` / `PingRedis`（纯连接测试）。
- 新 `server/service/system/sys_init_conn_test.go`：连接测试单测。
- 改 `server/service/system/sys_init.go`：编排器 `InitDB` 在 WriteConfig 前调 `applyRedisConfig`。
- 改 `server/initialize/init.go`：`dbReadyCallback` 扩展为即时连接 `OPS_REDIS`（guarded）。
- 改 `server/api/v1/system/sys_init.go`：新增 `PingDB` / `PingRedis` handler（含 Swagger、含 `OPS_DB!=nil` 守卫）。
- 改 `server/router/system/sys_init.go`：挂 `POST init/db/ping`、`POST init/redis/ping`。

**前端**
- 改 `web/src/typings/api/init.d.ts`：`InitDBForm` 增 Redis 三字段；新增 `PingDBForm`、`PingRedisForm`。
- 改 `web/src/service/api/init.ts`：新增 `fetchPingDB` / `fetchPingRedis`。
- 改 `web/src/locales/langs/{zh-cn,en-us}.ts`：补 `page.init.step.*` / `page.init.form.redis*` / `page.init.testConnection*` / `page.init.next|prev|finish`。
- 改 `web/src/views/_builtin/init/index.vue`：须知 + 3 步 stepper 重构。

**文档/记忆**
- 新 `aiDoc/memory/business/init-wizard-redis.md`；改 `aiDoc/memory/demand-index.md`、`aiDoc/frontend-backend/boundary.md`。

---

## Task 1: 后端请求 DTO（Redis 字段 + 测试连接请求体 + ToRedisConfig）

**Files:**
- Modify: `server/model/system/request/sys_init.go`
- Test: `server/model/system/request/sys_init_req_test.go`（新建）

**Interfaces:**
- Consumes: `github.com/hllkk/devops-admin/server/config`（`config.Redis`）
- Produces:
  - `request.InitDB` 增字段 `RedisAddr string` / `RedisPassword string` / `RedisDB int`（json：`redisAddr`/`redisPassword`/`redisDB`）
  - `func (i *InitDB) ToRedisConfig() config.Redis`
  - `type request.DBConnTest struct{ DBType, Host, Port, UserName, Password, DBName, DBPath, Template string }`（无 required 绑定）
  - `type request.PingRedis struct{ Addr, Password string; DB int }`

- [ ] **Step 1: 写失败测试**

新建 `server/model/system/request/sys_init_req_test.go`：
```go
package request

import "testing"

func TestToRedisConfig(t *testing.T) {
	i := InitDB{RedisAddr: "127.0.0.1:6379", RedisPassword: "pw", RedisDB: 2}
	c := i.ToRedisConfig()
	if c.Addr != "127.0.0.1:6379" || c.Password != "pw" || c.DB != 2 {
		t.Fatalf("redis config mismatch: %+v", c)
	}
	if c.UseCluster {
		t.Fatal("单实例模式 UseCluster 应为 false")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd /home/devops-admin/server && go test ./model/system/request/ -run TestToRedisConfig -v`
Expected: FAIL（`i.ToRedisConfig undefined`）

- [ ] **Step 3: 实现**

在 `server/model/system/request/sys_init.go` 的 `InitDB` 结构体末尾追加三字段，并在文件末尾追加方法与两个请求体：
```go
type InitDB struct {
	AdminPassword string `json:"adminPassword" binding:"required"`
	DBType        string `json:"dbType"`
	Host          string `json:"host"`
	Port          string `json:"port"`
	UserName      string `json:"userName"`
	Password      string `json:"password"`
	DBName        string `json:"dbName" binding:"required"`
	DBPath        string `json:"dbPath"`
	Template      string `json:"template"`
	// Redis 配置（向导第 2 步采集，随 initdb 一次性提交）
	RedisAddr     string `json:"redisAddr"`
	RedisPassword string `json:"redisPassword"`
	RedisDB       int    `json:"redisDB"`
}

// ToRedisConfig 转换为 config.Redis（单实例，向导不暴露集群/连接池）
func (i *InitDB) ToRedisConfig() config.Redis {
	return config.Redis{
		Name:     "default",
		Addr:     i.RedisAddr,
		Password: i.RedisPassword,
		DB:       i.RedisDB,
	}
}

// DBConnTest 数据库连接测试请求体（向导「测试连接」按钮，不建库、不落盘）
// 与 InitDB 的 DB 字段同构，但无 required 绑定、无管理员/Redis 字段。
type DBConnTest struct {
	DBType   string `json:"dbType"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	UserName string `json:"userName"`
	Password string `json:"password"`
	DBName   string `json:"dbName"`
	DBPath   string `json:"dbPath"`
	Template string `json:"template"`
}

// PingRedis Redis 连接测试请求体
type PingRedis struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd /home/devops-admin/server && go test ./model/system/request/ -run TestToRedisConfig -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
cd /home/devops-admin && git add server/model/system/request/sys_init.go server/model/system/request/sys_init_req_test.go
git commit -m "feat(model): InitDB 增 Redis 字段与 ToRedisConfig，新增 DBConnTest/PingRedis 请求体"
```

---

## Task 2: initialize/redis.go 抽 DialRedis（不写 global、不 panic）

**Files:**
- Modify: `server/initialize/redis.go`
- Test: `server/initialize/redis_test.go`（新建）

**Interfaces:**
- Consumes: `config.Redis`、`github.com/redis/go-redis/v9`
- Produces: `func initialize.DialRedis(redisCfg config.Redis) (redis.UniversalClient, error)`（建客户端 + Ping，失败返回 error，不 panic、不写 global）；`Redis()` 改为复用它。

- [ ] **Step 1: 写失败测试**

新建 `server/initialize/redis_test.go`：
```go
package initialize

import (
	"testing"

	"github.com/hllkk/devops-admin/server/config"
)

func TestDialRedis_WrongAddr(t *testing.T) {
	_, err := DialRedis(config.Redis{Addr: "127.0.0.1:33999"}) // 33999 几乎必然无人监听
	if err == nil {
		t.Fatal("错误地址应返回连接错误，得到 nil")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd /home/devops-admin/server && go test ./initialize/ -run TestDialRedis_WrongAddr -v`
Expected: FAIL（`undefined: DialRedis`）

- [ ] **Step 3: 实现**

把 `server/initialize/redis.go` 的 `initRedisClient` 重命名为导出的 `DialRedis`，删除其对 `global.OPS_LOG` 的依赖（改为返回 error 由调用方记日志/panic），并让 `Redis()` 复用它：
```go
package initialize

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/hllkk/devops-admin/server/config"
	"github.com/hllkk/devops-admin/server/global"
)

// DialRedis 按 cfg 建客户端并 Ping；不写 global，失败返回 error（不 panic）。
// Redis()（启动）与 dbReadyCallback（首初始化后即时连接）共用此函数。
func DialRedis(redisCfg config.Redis) (redis.UniversalClient, error) {
	var client redis.UniversalClient
	if redisCfg.UseCluster {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    redisCfg.ClusterAddrs,
			Password: redisCfg.Password,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     redisCfg.Addr,
			Password: redisCfg.Password,
			DB:       redisCfg.DB,
		})
	}
	pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}
	global.OPS_LOG.Info("redis connect ping response:", zap.String("name", redisCfg.Name), zap.String("pong", pong))
	return client, nil
}

func Redis() {
	redisClient, err := DialRedis(global.OPS_CONFIG.Redis)
	if err != nil {
		panic(err)
	}
	global.OPS_REDIS = redisClient
}

func RedisList() {
	redisMap := make(map[string]redis.UniversalClient)
	for _, redisCfg := range global.OPS_CONFIG.RedisList {
		client, err := DialRedis(redisCfg)
		if err != nil {
			panic(err)
		}
		redisMap[redisCfg.Name] = client
	}
	global.OPS_REDISList = redisMap
}
```
> 日志沿用 `*zap.Logger` 结构化写法（`global.OPS_LOG` 为 `*zap.Logger`，见 `api/v1/system/sys_init.go` 的 `zap.Error/zap.String` 用法），import 补 `go.uber.org/zap`。

- [ ] **Step 4: 编译 + 跑测试确认通过**

Run: `cd /home/devops-admin/server && go build ./initialize/ && go test ./initialize/ -run TestDialRedis_WrongAddr -v`
Expected: 编译通过；测试 PASS（`dial tcp 127.0.0.1:33999: connect: connection refused` 类错误被返回）

- [ ] **Step 5: 提交**

```bash
cd /home/devops-admin && git add server/initialize/redis.go server/initialize/redis_test.go
git commit -m "refactor(initialize): 抽 DialRedis 复用建客户端+Ping 逻辑，Redis/RedisList 改用它"
```

---

## Task 3: 连接测试 service（PingDB / PingRedis）+ 单测

**Files:**
- Create: `server/service/system/sys_init_conn.go`
- Test: `server/service/system/sys_init_conn_test.go`（新建）

**Interfaces:**
- Consumes: `request.DBConnTest`、`request.PingRedis`、`request.InitDB`（复用其 `*EmptyDsn()`）、`github.com/redis/go-redis/v9`、`database/sql`
- Produces:
  - `func (initDBService *InitDBService) PingDB(conf request.DBConnTest) error`
  - `func (initDBService *InitDBService) PingRedis(conf request.PingRedis) error`

- [ ] **Step 1: 写失败测试**

新建 `server/service/system/sys_init_conn_test.go`（`package system`，内部包以便后续访问未导出 `pingSQL`/`pingSqliteDir`）：
```go
package system

import (
	"path/filepath"
	"testing"

	sysReq "github.com/hllkk/devops-admin/server/model/system/request"
)

func TestPingRedis_WrongAddr(t *testing.T) {
	svc := &InitDBService{}
	if err := svc.PingRedis(sysReq.PingRedis{Addr: "127.0.0.1:33999"}); err == nil {
		t.Fatal("错误 Redis 地址应返回连接错误")
	}
}

func TestPingRedis_Local(t *testing.T) {
	svc := &InitDBService{}
	if err := svc.PingRedis(sysReq.PingRedis{Addr: "127.0.0.1:6379"}); err != nil {
		t.Skipf("本地 Redis 不可用，跳过: %v", err)
	}
}

func TestPingDB_Sqlite_ValidDir(t *testing.T) {
	svc := &InitDBService{}
	dir := t.TempDir()
	if err := svc.PingDB(sysReq.DBConnTest{DBType: "sqlite", DBPath: dir, DBName: "x"}); err != nil {
		t.Fatalf("合法目录应通过: %v", err)
	}
}

func TestPingDB_Sqlite_NotExistDir(t *testing.T) {
	svc := &InitDBService{}
	if err := svc.PingDB(sysReq.DBConnTest{DBType: "sqlite", DBPath: filepath.Join(t.TempDir(), "no-such-subdir"), DBName: "x"}); err == nil {
		t.Fatal("不存在的目录应失败")
	}
}

func TestPingDB_Mysql_WrongAddr(t *testing.T) {
	svc := &InitDBService{}
	err := svc.PingDB(sysReq.DBConnTest{DBType: "mysql", Host: "127.0.0.1", Port: "33999", UserName: "x", Password: "x", DBName: "x"})
	if err == nil {
		t.Fatal("错误 mysql 地址应失败")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd /home/devops-admin/server && go test ./service/system/ -run "TestPingRedis|TestPingDB" -v`
Expected: FAIL（`svc.PingRedis undefined` / `svc.PingDB undefined`）

- [ ] **Step 3: 实现**

新建 `server/service/system/sys_init_conn.go`：
```go
package system

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"

	sysReq "github.com/hllkk/devops-admin/server/model/system/request"
)

// PingDB 测试数据库连接：只 ping 不建库、不落盘、无副作用。
func (initDBService *InitDBService) PingDB(conf sysReq.DBConnTest) error {
	if conf.DBType == "sqlite" {
		return pingSqliteDir(conf.DBPath)
	}
	// 复用 request.InitDB 上的 dsn 构造方法（DBConnTest 与 InitDB 的 DB 字段同构）
	ic := sysReq.InitDB{
		DBType: conf.DBType, Host: conf.Host, Port: conf.Port,
		UserName: conf.UserName, Password: conf.Password,
		DBName: conf.DBName, DBPath: conf.DBPath,
	}
	switch conf.DBType {
	case "mysql":
		return pingSQL("mysql", ic.MysqlEmptyDsn())
	case "pgsql":
		return pingSQL("pgx", ic.PgsqlEmptyDsn())
	case "mssql":
		return pingSQL("sqlserver", ic.MssqlEmptyDsn())
	default:
		return fmt.Errorf("不支持的数据库类型: %s", conf.DBType)
	}
}

// pingSQL 打开连接 → Ping → 关闭，不执行任何 SQL、不建库。
func pingSQL(driver, dsn string) error {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Ping()
}

// pingSqliteDir 无副作用校验父目录可写（不创建 .db 文件）。
func pingSqliteDir(dbPath string) error {
	if dbPath == "" {
		return errors.New("sqlite 数据库文件路径不能为空")
	}
	f, err := os.CreateTemp(dbPath, ".init-ping-*")
	if err != nil {
		return fmt.Errorf("sqlite 路径不可写: %w", err)
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Remove(f.Name())
}

// PingRedis 测试 Redis 连接：建客户端 → Ping → 关闭，不写 global、不落盘。
func (initDBService *InitDBService) PingRedis(conf sysReq.PingRedis) error {
	if conf.Addr == "" {
		return errors.New("redis 地址不能为空")
	}
	client := redis.NewClient(&redis.Options{Addr: conf.Addr, Password: conf.Password, DB: conf.DB})
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis 连接失败: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd /home/devops-admin/server && go test ./service/system/ -run "TestPingRedis|TestPingDB" -v`
Expected: PASS（`TestPingRedis_Local` 在无本地 Redis 时 SKIP，其余 PASS）

- [ ] **Step 5: 提交**

```bash
cd /home/devops-admin && git add server/service/system/sys_init_conn.go server/service/system/sys_init_conn_test.go
git commit -m "feat(service): 新增 PingDB/PingRedis 连接测试（只 ping 不建库、无副作用）"
```

---

## Task 4: ping API handler + 路由注册 + Swagger

**Files:**
- Modify: `server/api/v1/system/sys_init.go`
- Modify: `server/router/system/sys_init.go`

**Interfaces:**
- Consumes: `initDBService.PingDB` / `PingRedis`（Task 3）、`request.DBConnTest` / `request.PingRedis`（Task 1）、`response` 包
- Produces: `DBApi.PingDB(c *gin.Context)`、`DBApi.PingRedis(c *gin.Context)`；路由 `POST init/db/ping`、`POST init/redis/ping`

- [ ] **Step 1: 实现 handler**

在 `server/api/v1/system/sys_init.go` 的 `CheckDB` 之后追加两个 handler。两者统一前置守卫「已初始化则拒绝」：
```go
// PingDB 测试数据库连接（仅在系统未初始化时可用；不建库、不落盘）
// @Tags     SysInit
// @Summary  测试数据库连接
// @Produce  application/json
// @Param    data  body      request.DBConnTest               true  "数据库连接参数"
// @Success  200   {object}  response.Response{data=string}  "连接成功"
// @Router   /init/db/ping [post]
func (i *DBApi) PingDB(c *gin.Context) {
	if global.OPS_DB != nil {
		response.FailWithMessage("系统已初始化，无需测试连接", c)
		return
	}
	var conf request.DBConnTest
	if err := c.ShouldBindJSON(&conf); err != nil {
		response.FailWithMessage("参数校验不通过", c)
		return
	}
	if err := initDBService.PingDB(conf); err != nil {
		global.OPS_LOG.Error("数据库连接测试失败!", zap.Error(err))
		response.FailWithMessage("数据库连接失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("数据库连接成功", c)
}

// PingRedis 测试 Redis 连接（仅在系统未初始化时可用；不落盘）
// @Tags     SysInit
// @Summary  测试 Redis 连接
// @Produce  application/json
// @Param    data  body      request.PingRedis                true  "Redis 连接参数"
// @Success  200   {object}  response.Response{data=string}  "连接成功"
// @Router   /init/redis/ping [post]
func (i *DBApi) PingRedis(c *gin.Context) {
	if global.OPS_DB != nil {
		response.FailWithMessage("系统已初始化，无需测试连接", c)
		return
	}
	var conf request.PingRedis
	if err := c.ShouldBindJSON(&conf); err != nil {
		response.FailWithMessage("参数校验不通过", c)
		return
	}
	if err := initDBService.PingRedis(conf); err != nil {
		global.OPS_LOG.Error("Redis 连接测试失败!", zap.Error(err))
		response.FailWithMessage("Redis 连接失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("Redis 连接成功", c)
}
```

- [ ] **Step 2: 注册路由**

把 `server/router/system/sys_init.go` 的分组块改为：
```go
func (s *InitRouter) InitInitRouter(Router *gin.RouterGroup) {
	initRouter := Router.Group("init")
	{
		initRouter.POST("initdb", dbApi.InitDB)       // 初始化数据库
		initRouter.POST("checkdb", dbApi.CheckDB)     // 检测是否需要初始化数据库
		initRouter.POST("db/ping", dbApi.PingDB)      // 测试数据库连接
		initRouter.POST("redis/ping", dbApi.PingRedis) // 测试 Redis 连接
	}
}
```

- [ ] **Step 3: 编译确认**

Run: `cd /home/devops-admin/server && go build ./...`
Expected: 编译通过（若 `request` 未 import，确认 `import` 含 `github.com/hllkk/devops-admin/server/model/system/request`——该文件已 import，无需新增）

- [ ] **Step 4: 手动冒烟（可选，需后端在跑）**

启动后端后：
```bash
curl -sS -X POST http://127.0.0.1:8888/init/redis/ping -H 'Content-Type: application/json' -d '{"addr":"127.0.0.1:33999"}'
```
Expected: `{"code":"0001",...,"msg":"Redis 连接失败：..."}`（证明拒绝路径通）

- [ ] **Step 5: 提交**

```bash
cd /home/devops-admin && git add server/api/v1/system/sys_init.go server/router/system/sys_init.go
git commit -m "feat(api,router): 新增 /init/db/ping、/init/redis/ping 测试连接端点（已初始化拒绝）"
```

---

## Task 5: 编排器 Redis 落盘 + dbReadyCallback 即时连接

**Files:**
- Modify: `server/service/system/sys_init.go`
- Modify: `server/initialize/init.go`
- Test: `server/service/system/sys_init_redis_test.go`（新建）

**Interfaces:**
- Consumes: `request.InitDB.ToRedisConfig()`（Task 1）、`initialize.DialRedis`（Task 2，仅 `init.go` 侧调用，不跨层）
- Produces: `func applyRedisConfig(conf request.InitDB)`（service 层，把 Redis 配置落到 `global.OPS_CONFIG`，供编排器 WriteConfig 前调用）；`dbReadyCallback` 扩展为即时连接 `OPS_REDIS`。

- [ ] **Step 1: 写失败测试**

新建 `server/service/system/sys_init_redis_test.go`（`package system`）：
```go
package system

import (
	"testing"

	"github.com/hllkk/devops-admin/server/global"
	sysReq "github.com/hllkk/devops-admin/server/model/system/request"
)

func TestApplyRedisConfig(t *testing.T) {
	// 重置前置状态
	global.OPS_CONFIG.System.UseRedis = false
	global.OPS_CONFIG.Redis.Addr = ""

	applyRedisConfig(sysReq.InitDB{RedisAddr: "127.0.0.1:6379", RedisPassword: "pw", RedisDB: 2})

	if !global.OPS_CONFIG.System.UseRedis {
		t.Fatal("UseRedis 应被置为 true")
	}
	if global.OPS_CONFIG.Redis.Addr != "127.0.0.1:6379" || global.OPS_CONFIG.Redis.DB != 2 {
		t.Fatalf("Redis 配置未落盘: %+v", global.OPS_CONFIG.Redis)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd /home/devops-admin/server && go test ./service/system/ -run TestApplyRedisConfig -v`
Expected: FAIL（`undefined: applyRedisConfig`）

- [ ] **Step 3: 实现 service 侧（applyRedisConfig + 编排器接入）**

在 `server/service/system/sys_init.go` 新增函数，并在 `InitDB()` 编排器里 `initHandler.WriteConfig(ctx)` **之前**调用：
```go
// applyRedisConfig 把 Redis 配置落到运行时 OPS_CONFIG。
// 在编排器 WriteConfig 前调用：各 per-DB handler 的 WriteConfig 会全量 StructToMap 回写，
// 故 Redis 段与 system.use-redis 随本次落盘，4 个 handler 无需改动。
func applyRedisConfig(conf request.InitDB) {
	global.OPS_CONFIG.Redis = conf.ToRedisConfig()
	global.OPS_CONFIG.System.UseRedis = true
}
```
在 `InitDB()` 内，定位到 `if err = initHandler.WriteConfig(ctx); err != nil {`，**在其上一行**插入：
```go
	applyRedisConfig(conf)
```
（`conf` 即 `InitDBService.InitDB(conf request.InitDB)` 的入参，作用域内可见。）

- [ ] **Step 4: 实现 initialize 侧（dbReadyCallback 即时连 Redis）**

改 `server/initialize/init.go` 的 `SetupHandlers`，把 `SetDBReadyCallback` 回调扩展为在雪花回调后再 guarded 连 Redis。补 zap import：
```go
package initialize

import (
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/hllkk/devops-admin/server/utils"
	"go.uber.org/zap"
)

func SetupHandlers() {
	utils.GlobalSystemEvents.RegisterReloadHandler(func() error {
		return Reload()
	})
	system.SetDBReadyCallback(func() {
		RegisterCallbacks(global.OPS_DB)
		// 首初始化（/init/initdb）后即时连接 Redis：向导已 ping 测过，
		// 此处失败仅告警不 panic，重启后由 RunServer 兜底重连。
		if global.OPS_CONFIG.System.UseRedis && global.OPS_REDIS == nil {
			if client, err := DialRedis(global.OPS_CONFIG.Redis); err != nil {
				global.OPS_LOG.Warn("init 后即时连接 Redis 失败，重启后将自动重试", zap.Error(err))
			} else {
				global.OPS_REDIS = client
			}
		}
	})
}
```

- [ ] **Step 5: 编译 + 跑测试确认通过**

Run: `cd /home/devops-admin/server && go build ./... && go test ./service/system/ -run TestApplyRedisConfig -v`
Expected: 编译通过；测试 PASS

- [ ] **Step 6: 回归已有 e2e（确认未碰坏权限基座闭环）**

Run: `cd /home/devops-admin/server && go test ./service/system/ -run TestE2EPermissionBaseClosedLoop -v`
Expected: PASS（该 e2e 不走 `InitDB()` 编排器，不受 applyRedisConfig 影响）

- [ ] **Step 7: 提交**

```bash
cd /home/devops-admin && git add server/service/system/sys_init.go server/service/system/sys_init_redis_test.go server/initialize/init.go
git commit -m "feat(service,initialize): 编排器 WriteConfig 前落 Redis 配置；dbReadyCallback 即时连接 OPS_REDIS"
```

---

## Task 6: 前端类型 + service + i18n

**Files:**
- Modify: `web/src/typings/api/init.d.ts`
- Modify: `web/src/service/api/init.ts`
- Modify: `web/src/locales/langs/zh-cn.ts`
- Modify: `web/src/locales/langs/en-us.ts`

**Interfaces:**
- Consumes: 后端契约（`POST /init/db/ping` 请求 `DBConnTest`、`POST /init/redis/ping` 请求 `PingRedis`、`/init/initdb` 请求 `InitDB`+Redis 三字段）
- Produces: `Api.Init.InitDBForm` 增 `redisAddr/redisPassword/redisDB`；`Api.Init.PingDBForm`、`Api.Init.PingRedisForm`；`fetchPingDB(data)`、`fetchPingRedis(data)`；i18n key。

- [ ] **Step 1: 类型（仅新增 ping 请求体，不改 InitDBForm）**

> `InitDBForm` 的 Redis 字段挪到 Task 7 Step 0 与 vue model 同步改——否则本任务改完类型、vue 未改，`init/index.vue` 的 `model` 会缺字段，Task 6 的 pre-commit typecheck 会失败。本任务只新增 ping 请求体类型，不碰 `InitDBForm`。

在 `web/src/typings/api/init.d.ts` 的 `namespace Init` 内（`InitDBForm` 之后）追加两个 interface：
```ts
    /** /init/db/ping 请求体（数据库连接测试，不建库不落盘） */
    interface PingDBForm {
      dbType: DBType;
      host?: string;
      port?: string;
      userName?: string;
      password?: string;
      dbName: string;
      dbPath?: string;
      template?: string;
    }

    /** /init/redis/ping 请求体（Redis 连接测试） */
    interface PingRedisForm {
      addr: string;
      password?: string;
      db: number;
    }
```

- [ ] **Step 2: service 封装**

在 `web/src/service/api/init.ts` 末尾追加：
```ts
/**
 * 测试数据库连接（仅 ping，不建库不落盘）
 *
 * 对应后端 POST /init/db/ping。
 */
export function fetchPingDB(data: Api.Init.PingDBForm) {
  return request<string>({ url: '/init/db/ping', method: 'post', data });
}

/**
 * 测试 Redis 连接（不落盘）
 *
 * 对应后端 POST /init/redis/ping。
 */
export function fetchPingRedis(data: Api.Init.PingRedisForm) {
  return request<string>({ url: '/init/redis/ping', method: 'post', data });
}
```

- [ ] **Step 3: i18n（中文）**

在 `web/src/locales/langs/zh-cn.ts` 的 `page.init` 节点下补齐以下 key（若 `page.init` 已有 `form`/`title` 等，合并而非覆盖；仅新增下列键）。**禁硬编码中文到组件**：
```ts
    step: {
      db: '数据库',
      redis: 'Redis',
      admin: '管理员密码'
    },
    form: {
      // ... 既有 adminPassword/dbType/dbName/host/port/userName/password/dbPath/template 保留
      redisAddr: 'Redis 地址',
      redisAddrPlaceholder: '请输入 Redis 地址，如 127.0.0.1:6379',
      redisPassword: 'Redis 密码',
      redisPasswordPlaceholder: '无密码可留空',
      redisDB: 'Redis 库号',
      redisDBPlaceholder: '请输入库号，如 0'
    },
    testConnection: '测试连接',
    testing: '测试中…',
    testConnectionSuccess: '连接成功',
    testConnectionFailed: '连接失败',
    next: '下一步',
    prev: '上一步',
    finish: '完成',
    rule: {
      // 既有 adminPasswordLength 保留
      redisAddrRequired: '请输入 Redis 地址'
    }
```

- [ ] **Step 4: i18n（英文）**

在 `web/src/locales/langs/en-us.ts` 对应 `page.init` 节点补英文（key 结构与中文一致）：
```ts
    step: { db: 'Database', redis: 'Redis', admin: 'Admin Password' },
    form: {
      // 既有键保留
      redisAddr: 'Redis Address',
      redisAddrPlaceholder: 'e.g. 127.0.0.1:6379',
      redisPassword: 'Redis Password',
      redisPasswordPlaceholder: 'Leave empty if none',
      redisDB: 'Redis DB',
      redisDBPlaceholder: 'e.g. 0'
    },
    testConnection: 'Test Connection',
    testing: 'Testing…',
    testConnectionSuccess: 'Connected',
    testConnectionFailed: 'Connection failed',
    next: 'Next',
    prev: 'Previous',
    finish: 'Finish',
    rule: { /* adminPasswordLength 保留 */ redisAddrRequired: 'Redis address is required' }
```

- [ ] **Step 5: typecheck 确认**

Run: `cd /home/devops-admin/web && pnpm typecheck`
Expected: PASS（本任务只新增类型、不碰 `InitDBForm`，`init/index.vue` 不受影响）

- [ ] **Step 6: 提交**

```bash
cd /home/devops-admin && git add web/src/typings/api/init.d.ts web/src/service/api/init.ts web/src/locales/langs/zh-cn.ts web/src/locales/langs/en-us.ts
git commit -m "feat(web): 新增 PingDB/PingRedis 类型与封装、补 Redis 步 i18n"
```

---

## Task 7: 前端向导重构（须知 + 3 步 stepper）

**Files:**
- Modify: `web/src/views/_builtin/init/index.vue`

**Interfaces:**
- Consumes: `fetchPingDB` / `fetchPingRedis` / `fetchInitDB`（Task 6 + 现状）、`Api.Init.InitDBForm` / `PingDBForm` / `PingRedisForm`（Task 6）、`useFormRules` / `useNaiveForm`（现状已用）、NaiveUI `NSteps`/`NStep`
- Produces: 4 屏向导（须知 → DB → Redis → 管理员）；测试连接 4 态按钮；末步一次性提交。

- [ ] **Step 0: 类型——给 InitDBForm 补 Redis 三字段**

在 `web/src/typings/api/init.d.ts` 的 `InitDBForm` 末尾追加（与下方 model 默认值同步，避免 model 类型报错）：
```ts
      /** Redis 地址 host:port（必填） */
      redisAddr: string;
      /** Redis 密码（可空） */
      redisPassword: string;
      /** Redis 库号（默认 0） */
      redisDB: number;
```

- [ ] **Step 1: 重写 `<script setup>`（状态 + model + 测试 + 提交）**

把 `web/src/views/_builtin/init/index.vue` 的 `<script setup lang="ts">` 整段替换为：
```ts
import { computed, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { fetchInitDB, fetchPingDB, fetchPingRedis } from '@/service/api';
import { resetSystemInitCheck } from '@/router/guard/route';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { useAppStore } from '@/store/modules/app';
import { useThemeStore } from '@/store/modules/theme';
import loginBackground from '@/assets/svg-icon/login-background.svg';
import { $t } from '@/locales';

defineOptions({ name: 'Init' });

const router = useRouter();
const appStore = useAppStore();
const themeStore = useThemeStore();
const { createRequiredRule } = useFormRules();

/** stage: notice=须知屏；wizard=三步向导 */
const stage = ref<'notice' | 'wizard'>('notice');
/** 当前向导步骤：1=DB 2=Redis 3=Admin */
const currentStep = ref(1);
const submitting = ref(false);

// 三个独立表单 ref，分步校验
const { formRef: dbFormRef, validate: validateDB } = useNaiveForm();
const { formRef: redisFormRef, validate: validateRedis } = useNaiveForm();
const { formRef: adminFormRef, validate: validateAdmin } = useNaiveForm();

const dbTypeOptions: { label: string; value: Api.Init.DBType }[] = [
  { label: 'MySQL', value: 'mysql' },
  { label: 'PostgreSQL', value: 'pgsql' },
  { label: 'SQLite', value: 'sqlite' },
  { label: 'MSSQL', value: 'mssql' }
];

const connDefaults: Record<Exclude<Api.Init.DBType, 'sqlite'>, Partial<Api.Init.InitDBForm>> = {
  mysql: { host: '127.0.0.1', port: '3306', userName: 'root' },
  pgsql: { host: '127.0.0.1', port: '5432', userName: 'postgres', template: 'template0' },
  mssql: { host: '127.0.0.1', port: '1433', userName: 'sa' }
};

const model = reactive<Api.Init.InitDBForm>({
  adminPassword: '',
  dbType: 'mysql',
  host: '127.0.0.1',
  port: '3306',
  userName: 'root',
  password: '',
  dbName: 'devops_admin',
  redisAddr: '127.0.0.1:6379',
  redisPassword: '',
  redisDB: 0
});

const isSqlite = computed(() => model.dbType === 'sqlite');
const isPgsql = computed(() => model.dbType === 'pgsql');

// 测试连接：4 态
type TestState = 'idle' | 'testing' | 'success' | 'error';
const dbTest = ref<TestState>('idle');
const redisTest = ref<TestState>('idle');

// 「下一步/完成」是否可点：测试通过才放行
const dbStepReady = computed(() => dbTest.value === 'success');
const redisStepReady = computed(() => redisTest.value === 'success');

function handleDbTypeChange(val: string | number) {
  const type = val as Api.Init.DBType;
  model.dbType = type;
  dbTest.value = 'idle'; // 改动字段 → 失效已测
  if (type === 'sqlite') {
    model.host = undefined;
    model.port = undefined;
    model.userName = undefined;
    model.password = undefined;
    model.template = undefined;
    if (model.dbPath === undefined) model.dbPath = '';
    return;
  }
  model.dbPath = undefined;
  Object.assign(model, connDefaults[type]);
  if (type !== 'pgsql') model.template = undefined;
}

const dbRules = computed(() => {
  const needConn = !isSqlite.value;
  return {
    dbType: [createRequiredRule($t('page.init.form.dbType'))],
    dbName: [createRequiredRule($t('page.init.form.dbNamePlaceholder'))],
    host: needConn ? [createRequiredRule($t('page.init.form.hostPlaceholder'))] : [],
    port: needConn ? [createRequiredRule($t('page.init.form.portPlaceholder'))] : [],
    userName: needConn ? [createRequiredRule($t('page.init.form.userNamePlaceholder'))] : [],
    password: needConn ? [createRequiredRule($t('page.init.form.passwordPlaceholder'))] : [],
    dbPath: isSqlite.value ? [createRequiredRule($t('page.init.form.dbPathPlaceholder'))] : [],
    template: []
  };
});

const redisRules = {
  redisAddr: [createRequiredRule($t('page.init.rule.redisAddrRequired'))]
};

const adminRules = {
  adminPassword: [createRequiredRule($t('page.init.form.adminPasswordPlaceholder'))]
};

/** 任一相关字段变更 → 失效对应测试标记 */
function invalidateOnDbInput() {
  dbTest.value = 'idle';
}
function invalidateOnRedisInput() {
  redisTest.value = 'idle';
}

async function testDB() {
  dbTest.value = 'testing';
  const payload: Api.Init.PingDBForm = {
    dbType: model.dbType,
    host: model.host,
    port: model.port,
    userName: model.userName,
    password: model.password,
    dbName: model.dbName,
    dbPath: model.dbPath,
    template: model.template
  };
  const { error } = await fetchPingDB(payload);
  if (error) {
    dbTest.value = 'error';
    return; // 错误文案由请求层统一提示
  }
  dbTest.value = 'success';
  window.$message?.success($t('page.init.testConnectionSuccess'));
}

async function testRedis() {
  redisTest.value = 'testing';
  const { error } = await fetchPingRedis({
    addr: model.redisAddr,
    password: model.redisPassword,
    db: model.redisDB
  });
  if (error) {
    redisTest.value = 'error';
    return;
  }
  redisTest.value = 'success';
  window.$message?.success($t('page.init.testConnectionSuccess'));
}

async function nextFromDb() {
  try {
    await validateDB();
  } catch {
    return;
  }
  currentStep.value = 2;
}

async function nextFromRedis() {
  try {
    await validateRedis();
  } catch {
    return;
  }
  currentStep.value = 3;
}

async function handleSubmit() {
  try {
    await validateAdmin();
  } catch {
    return;
  }
  if (model.adminPassword.trim().length < 6) {
    window.$message?.error($t('page.init.rule.adminPasswordLength'));
    return;
  }
  submitting.value = true;
  const { error } = await fetchInitDB(model);
  submitting.value = false;
  if (error) return;
  resetSystemInitCheck();
  window.$message?.success($t('page.init.successTitle'));
  router.replace({ name: 'login' });
}

function handleBack() {
  router.replace({ name: 'login' });
}
```

- [ ] **Step 2: 重写 `<template>`（须知屏 + NSteps + 三个 NForm）**

把 `<template>` 内 `<main>...</main>` 整段替换为下面的结构（左侧插画/头部/全屏 loading 保持原样不动，只改 `<main>`）：
```html
      <main class="m-auto w-full max-w-560px px-24px">
        <!-- 须知屏 -->
        <div v-if="stage === 'notice'" class="rounded-8px p-16px text-center">
          <h2 class="mb-12px text-22px font-600">{{ $t('page.init.noticeTitle') }}</h2>
          <p class="mb-24px text-15px leading-relaxed color-gray-600 dark:color-gray-300">
            {{ $t('page.init.noticeDesc') }}
          </p>
          <NSpace vertical :size="12">
            <NButton type="primary" size="large" block @click="stage = 'wizard'">
              {{ $t('page.init.confirm') }}
            </NButton>
            <NButton quaternary size="large" block @click="handleBack">
              {{ $t('page.init.back') }}
            </NButton>
          </NSpace>
        </div>

        <!-- 三步向导 -->
        <div v-else>
          <h2 class="mb-16px text-22px font-600">{{ $t('page.init.title') }}</h2>
          <NSteps :current="currentStep" size="small" class="mb-24px">
            <NStep :title="$t('page.init.step.db')" />
            <NStep :title="$t('page.init.step.redis')" />
            <NStep :title="$t('page.init.step.admin')" />
          </NSteps>

          <!-- 步骤 1：数据库 -->
          <NForm
            v-show="currentStep === 1"
            ref="dbFormRef"
            :model="model"
            :rules="dbRules"
            label-placement="top"
            size="large"
          >
            <NFormItem path="dbType" :label="$t('page.init.form.dbType')">
              <NSelect :value="model.dbType" :options="dbTypeOptions" @update:value="handleDbTypeChange" />
            </NFormItem>
            <NFormItem path="dbName" :label="$t('page.init.form.dbName')">
              <NInput v-model:value="model.dbName" :placeholder="$t('page.init.form.dbNamePlaceholder')" @update:value="invalidateOnDbInput" />
            </NFormItem>
            <template v-if="!isSqlite">
              <div class="flex gap-16px">
                <NFormItem path="host" :label="$t('page.init.form.host')" class="flex-1">
                  <NInput v-model:value="model.host" :placeholder="$t('page.init.form.hostPlaceholder')" @update:value="invalidateOnDbInput" />
                </NFormItem>
                <NFormItem path="port" :label="$t('page.init.form.port')" class="w-120px">
                  <NInput v-model:value="model.port" :placeholder="$t('page.init.form.portPlaceholder')" @update:value="invalidateOnDbInput" />
                </NFormItem>
              </div>
              <div class="flex gap-16px">
                <NFormItem path="userName" :label="$t('page.init.form.userName')" class="flex-1">
                  <NInput v-model:value="model.userName" :placeholder="$t('page.init.form.userNamePlaceholder')" @update:value="invalidateOnDbInput" />
                </NFormItem>
                <NFormItem path="password" :label="$t('page.init.form.password')" class="flex-1">
                  <NInput v-model:value="model.password" type="password" show-password-on="click" :placeholder="$t('page.init.form.passwordPlaceholder')" @update:value="invalidateOnDbInput" />
                </NFormItem>
              </div>
            </template>
            <NFormItem v-if="isSqlite" path="dbPath" :label="$t('page.init.form.dbPath')">
              <NInput v-model:value="model.dbPath" :placeholder="$t('page.init.form.dbPathPlaceholder')" @update:value="invalidateOnDbInput" />
            </NFormItem>
            <NFormItem v-if="isPgsql" path="template" :label="$t('page.init.form.template')">
              <NInput v-model:value="model.template" :placeholder="$t('page.init.form.templatePlaceholder')" @update:value="invalidateOnDbInput" />
            </NFormItem>

            <NSpace vertical :size="12" class="mt-8px">
              <NButton size="large" block :loading="dbTest === 'testing'" :type="dbTest === 'success' ? 'success' : 'default'" @click="testDB">
                {{ $t('page.init.testConnection') }}
              </NButton>
              <NButton type="primary" size="large" block :disabled="!dbStepReady" @click="nextFromDb">
                {{ $t('page.init.next') }}
              </NButton>
              <NButton quaternary size="large" block @click="handleBack">{{ $t('page.init.back') }}</NButton>
            </NSpace>
          </NForm>

          <!-- 步骤 2：Redis -->
          <NForm
            v-show="currentStep === 2"
            ref="redisFormRef"
            :model="model"
            :rules="redisRules"
            label-placement="top"
            size="large"
          >
            <NFormItem path="redisAddr" :label="$t('page.init.form.redisAddr')">
              <NInput v-model:value="model.redisAddr" :placeholder="$t('page.init.form.redisAddrPlaceholder')" @update:value="invalidateOnRedisInput" />
            </NFormItem>
            <NFormItem :label="$t('page.init.form.redisPassword')">
              <NInput v-model:value="model.redisPassword" type="password" show-password-on="click" :placeholder="$t('page.init.form.redisPasswordPlaceholder')" @update:value="invalidateOnRedisInput" />
            </NFormItem>
            <NFormItem :label="$t('page.init.form.redisDB')">
              <NInputNumber v-model:value="model.redisDB" :placeholder="$t('page.init.form.redisDBPlaceholder')" class="w-full" @update:value="invalidateOnRedisInput" />
            </NFormItem>
            <NSpace vertical :size="12" class="mt-8px">
              <NButton size="large" block :loading="redisTest === 'testing'" :type="redisTest === 'success' ? 'success' : 'default'" @click="testRedis">
                {{ $t('page.init.testConnection') }}
              </NButton>
              <NSpace justify="space-between" :wrap="false">
                <NButton quaternary size="large" @click="currentStep = 1">{{ $t('page.init.prev') }}</NButton>
                <NButton type="primary" size="large" :disabled="!redisStepReady" @click="nextFromRedis">{{ $t('page.init.next') }}</NButton>
              </NSpace>
            </NSpace>
          </NForm>

          <!-- 步骤 3：管理员密码 -->
          <NForm
            v-show="currentStep === 3"
            ref="adminFormRef"
            :model="model"
            :rules="adminRules"
            label-placement="top"
            size="large"
          >
            <NFormItem path="adminPassword" :label="$t('page.init.form.adminPassword')">
              <NInput v-model:value="model.adminPassword" type="password" show-password-on="click" :placeholder="$t('page.init.form.adminPasswordPlaceholder')" />
            </NFormItem>
            <NSpace vertical :size="12" class="mt-8px">
              <NButton type="primary" size="large" block :loading="submitting" @click="handleSubmit">{{ $t('page.init.finish') }}</NButton>
              <NButton quaternary size="large" block :disabled="submitting" @click="currentStep = 2">{{ $t('page.init.prev') }}</NButton>
            </NSpace>
          </NForm>
        </div>
      </main>
```
> 头部 `<header>`、左侧插画列、全屏 loading `<Teleport>` 保持原文件不动。删掉旧的 `showForm`/` Transition`（已由 `stage` 取代）。

- [ ] **Step 3: 删除已废弃的旧逻辑**

确认旧 `<script setup>` 里的 `showForm`、旧 `handleSubmit` 内联在表单里的引用、旧 `<Transition>` 包裹均已移除；`useNaiveForm` 现取三个实例（`dbFormRef/redisFormRef/adminFormRef`），原单个 `formRef/validate` 不再使用。

- [ ] **Step 4: 类型检查 + lint + 格式化**

Run: `cd /home/devops-admin/web && pnpm typecheck && pnpm lint && pnpm fmt`
Expected: 全部通过（pre-commit 也强制这三项）。若 typecheck 报 `InitDBForm` 缺字段——回查 Step 0 是否完成；若报 NStep/NSteps 未注册——NaiveUI 自动导入应已覆盖，必要时确认 `web/src/typings/components.d.ts` 含全局 NaiveUI 组件（本项目按 auto-import，通常无需手改）。

- [ ] **Step 5: 提交**

```bash
cd /home/devops-admin && git add web/src/views/_builtin/init/index.vue
git commit -m "feat(web): 初始化向导重构为须知+三步(DB→Redis→管理员)，DB/Redis 带测试连接"
```

---

## Task 8: 文档与业务记忆（按 AGENT.MD）

**Files:**
- Create: `aiDoc/memory/business/init-wizard-redis.md`
- Modify: `aiDoc/memory/demand-index.md`
- Modify: `aiDoc/frontend-backend/boundary.md`

- [ ] **Step 1: 业务需求记忆**

新建 `aiDoc/memory/business/init-wizard-redis.md`：
```markdown
# 初始化向导 Redis 步

- **来源**：2026-07-13 用户需求
- **关联**：设计 `docs/superpowers/specs/2026-07-13-init-wizard-redis-design.md`；实现 `docs/superpowers/plans/2026-07-13-init-wizard-redis.md`

## 需求
初始化向导由「单步合并（DB+管理员）」改为「须知 → ①数据库 → ②Redis → ③管理员密码」；DB/Redis 各提供「测试连接」按钮（ephemeral ping，不建库不落盘）；Redis 必填、单实例（Addr/Password/DB）；末步一次性提交，提交后 `config.Redis` + `system.use-redis:true` 落盘并即时连接 `OPS_REDIS`。

## 关键约束
- 提交模型：末步一次性（保持原子性，中途放弃无半成品）。
- ping 端点 `OPS_DB!=nil` 拒绝（防滥用）；`service/system` 不得 import `initialize`，即时连 Redis 走 `dbReadyCallback`。
```

- [ ] **Step 2: 更新需求索引**

在 `aiDoc/memory/demand-index.md` 追加一行（按文件现有格式）：
```markdown
- [初始化向导 Redis 步](business/init-wizard-redis.md) — 向导改三步、加测试连接、Redis 落盘即连（2026-07-13）
```

- [ ] **Step 3: 同步前后端契约文档**

在 `aiDoc/frontend-backend/boundary.md` 的 init/初始化相关段落（若无则在「契约规则」后补一节）追加：
```markdown
## 初始化向导（/init/*）

- `POST /init/checkdb`：返回 `{needInit}`，守卫用，语义不变。
- `POST /init/db/ping`：请求 `DBConnTest{dbType,host,port,userName,password,dbName,dbPath,template}`，ephemeral 连接测试（不建库不落盘）；`OPS_DB!=nil` 时拒绝。
- `POST /init/redis/ping`：请求 `PingRedis{addr,password,db}`，ephemeral 连接测试；`OPS_DB!=nil` 时拒绝。
- `POST /init/initdb`：请求 `InitDB`（含 `adminPassword` + DB 字段 + `redisAddr/redisPassword/redisDB`），原子完成建库+建表+种子+回写 config（含 Redis、`use-redis:true`）。
- 统一响应 `{code:"0000"|"0001", data, msg}`；ping 成功 `data` 为空串，前端 flat request 只看 `error`。
```

- [ ] **Step 4: 提交**

```bash
cd /home/devops-admin && git add aiDoc/memory/business/init-wizard-redis.md aiDoc/memory/demand-index.md aiDoc/frontend-backend/boundary.md
git commit -m "docs(aiDoc): 记录初始化向导 Redis 需求并同步 /init/* 契约说明"
```

---

## 收尾验收（全部任务完成后）

- [ ] 后端全量测试：`cd /home/devops-admin/server && go test ./...`
- [ ] 前端门禁：`cd /home/devops-admin/web && pnpm typecheck && pnpm lint && pnpm fmt`
- [ ] 端到端手测：删除/备份 `config.yaml` 的 DB 配置使其 `needInit=true` → 访问 `/init` → 须知 → 填 DB → 测试通过 → 填 Redis → 测试通过 → 填管理员密码 → 完成 → 跳登录 → 用 `super` + 设置的密码登录成功 → 检查 `config.yaml` 含 `redis:` 段且 `system.use-redis: true`。
