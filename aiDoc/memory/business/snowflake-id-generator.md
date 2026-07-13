# 雪花算法 ID 生成器（主键策略改造）

- 日期：2026-07-11
- 状态：已实现（后端 + 配置 + 契约文档），编译/vet/单测/集成测试全绿
- 关联分支：feat/multi-module-isolation
- **后续演进（2026-07-12）**：`global.OPS_MODEL` 已移除 `ID` 字段，主键改由各业务模型自定义（对外 `userId`/`roleId`、内部 `id`）。雪花回调改为按 `PrioritizedPrimaryField` 填充任意整型主键，**不再依赖基座是否含 `ID`**。下文「`OPS_MODEL.ID`」相关描述为引入时的历史状态。见 [[system-user-role-models]]。

## 背景

为后续创建用户表等业务表做准备，引入雪花算法（Snowflake）作为统一主键策略。原 `global.OPS_MODEL.ID` 是 `uint` 自增主键，全仓无 ID 生成器；改造时业务表几乎未建（仅 `sys_error` 一张在用 `OPS_MODEL`，`SysUser`/菜单/角色全是注释桩），属最佳时机。

## 契约决策（用户拍板）

1. **ID 传输**：字符串传输。Go 端 `json:"ID,string"`，前端按 string 收发——规避 JS `Number` 2^53 精度丢失。
2. **实现方式**：自实现（`utils/snowflake`，约 120 行），不引第三方依赖。
3. **推进节奏**：先只做雪花算法，用户表随后。
4. **范围**：最小聚焦——只动主键生成本身，认证链路（`Login`/`BaseClaims`/`GetById`）暂不碰。

## 实现要点

- **雪花算法** `utils/snowflake/snowflake.go`：经典 64-bit 分配（1 符号 + 41 毫秒时间戳 + 10 worker + 12 序列号）。`MustInit(workerID, epoch)` 启动期初始化（panic-on-error，`sync.Once` 幂等）；`NextID() (int64, error)` / `MustNextID()` 生成；`sync.Mutex` 保护；时钟回拨阈值内（5ms）自旋等待、超阈值报错。配套 `snowflake_test.go`（100 goroutine×10000 并发唯一性、单调性、worker 编码、panic 校验）。
- **主键类型** `global/common.go`：`OPS_MODEL.ID` `uint` → `int64`，tag 改 `gorm:"primaryKey;autoIncrement:false" json:"ID,string"`（显式关自增，字符串传输）。
- **GORM 回调** `initialize/callbacks.go`：`RegisterCallbacks(db)` 注册 `BeforeCreate` 回调 `ops:snowflake_id`，用 `Schema.PrioritizedPrimaryField` 在整型主键为 0 时反射赋值（`Field.ReflectValueOf`/`Set`），支持 struct 与 slice；用 `Get` 判幂等避免重载重复注册。集成测试 `callbacks_test.go`（sqlite 内存库）验证单条/批量/显式 ID 不覆盖 + json 字符串序列化。
- **三路径注册**：
  - 常规启动 `main.go:initializeSystem()` 在 `OPS_DB = Gorm()` 后调 `RegisterCallbacks`
  - 热重载 `initialize/reload.go` 在 `OPS_DB = Gorm()` 后调
  - 首初始化 `initialize/init.go` 的 `SetupHandlers()` 注入 `system.SetDBReadyCallback(...)`（钩子在 `service/system/sys_init.go:149`）
- **节点初始化** `initialize/other.go` 的 `OtherInit()`：解析 `Snowflake.Epoch`（RFC3339）+ `Node`，调 `snowflake.MustInit`（幂等，热重载安全）。
- **配置**：新增 `config/snowflake.go`（`Snowflake{Node, Epoch}`），`config/config.go` 的 `Server` 加字段，`config.yaml`/`config.docker.yaml` 加 `snowflake:` 段。
- **契约** `aiDoc/frontend-backend/boundary.md` 新增「主键 ID 契约」节。

## 暂不动（已知技术债，做用户表/登录时统一处理）

- `model/system/sys_user.go` 的 `Login` 接口 `GetUserId()/GetAuthorityId() uint`
- `model/system/request/jwt.go` 的 `BaseClaims.ID/AuthorityId uint`
- `utils/claims.go` 的 `GetUserID()/GetUserAuthorityId() uint`、`LoginToken`
- `model/common/request/common.go` 的 `GetById.ID int`/`IdsReq.Ids []int`/`GetById.Uint()`
- `model/common/basetypes.go` 的 `TreeNode.GetID()/GetParentID() int`
- JWT token payload 内 ID 精度问题（届时 claims 改 string 存储）

理由：均为空壳、0 业务调用、与 `OPS_MODEL.ID` 无编译耦合（`SysUser` 仍注释中）。

## 验证

- `go build ./...`、`go vet ./...` 通过
- `go test ./utils/snowflake/` 6 个用例全过（含 100 万并发唯一性）
- `go test ./initialize/ -run TestCallbackAssignsSnowflakeID` 集成测试通过

## 注意

- `sys_error` 表 id 列原为自增整型，改 int64 雪花后需重建表（dev 无生产数据，drop 后 AutoMigrate 即可，列须 BIGINT）。
- `service/system/sys_error.go` 的 ID 形参已是 `string`（`"id = ?"` 参数化），对 int64 列查询兼容性待真实 DB 验证；若驱动报错，届时在该 service 查询处转 int64。
- 多实例部署时 `snowflake.node` 必须唯一（当前单实例从 config 读；演进为 Redis 自增/DB 号段）。
- db-list 多 DB 暂未注册回调（无业务，演进项）。

## 演进参考（对比 yitter/IdGenerator）

调研了 `yitter/idgenerator` 的 Go 实现（`SnowWorkerM1`）。其核心创新针对**高频/多实例**场景，当前 devops 后台低频主键用经典方案已足够，**暂不借鉴**，记录备查：

- **漂移机制（OverCost）**：某毫秒序列耗尽时不 sleep，`_LastTimeTick++` 借用下一毫秒继续生成（`TopOverCostCount` 默认 2000 次），状态机 `_IsOverCost` 切换。应对突发高频，吞吐显著高于「等待下一毫秒」。未来单毫秒生成量接近 4096 时可考虑引入。
- **位分配全可配**：`WorkerIdBitLength`(默认6)/`SeqBitLength`(默认6)/`BaseTime` 全从 options 读，约束 `seq+worker ≤ 22`，时间戳位默认 51。我们写死经典 10+12（时间戳 41 位，约 69 年）；若未来节点规模或写入速率需调整，可改为 config 可配。
- **回拨不阻塞 + MinSeqNumber 保留位**：每毫秒序列从 5 起，0–4 保留（0=手工新值，1–4=回拨次序位），回拨时用 `_TurnBackTimeTick` 在回拨时间戳继续生成、`_TurnBackIndex` 1–4 循环，不等待不报错，支持 4 次重叠回拨。我们用「5ms 内 sleep 追上、超阈值 error」，对回拨罕见的 devops 场景够用。
- **worker id 自动注册**：yitter 的 `Go/regworkerid/` 提供 worker id 注册（依赖外部存储）。未来多实例时可作为我们 worker id 分配（当前从 config 读 node）的演进参考。
