# docker-dev 配置目录

本目录存放 `deploy/docker-dev/docker-compose.yml` 各服务的挂载配置，均以只读方式挂入容器。

| 文件/目录 | 挂载服务 | 说明 |
|---|---|---|
| `postgresql.conf` | pgsql | PG 自定义配置 |
| `redis.conf` | redis | Redis 自定义配置 |
| `litellm.yaml` | litellm | LiteLLM 底座开关（模型/凭证/密钥由管理面同步，不在此配置） |
| `dbhub.toml` | dbhub | DBHub 数据库 MCP Server 多库配置（详见下文） |

---

## dbhub.toml 配置指南

DBHub 是自托管的数据库 MCP Server（HTTP 端点），平台按远程型纳管（`http://dbhub:8080/mcp` + Bearer token）。**只支持关系型**（PG/MySQL/MariaDB/SQLServer/SQLite），Redis 不在支持列表。

### 工具模型

工具与库（source）绑定，声明在 `[[tools]]` 段。**多 source 时工具名自动带后缀**：source `shop` 的 `execute_sql` 注册为 `execute_sql_shop`；source 的 `description` 会拼进工具描述，AI 靠它判断查询发给哪个库。

内置工具四个：

| name | 作用 | 备注 |
|---|---|---|
| `execute_sql` | 执行 SQL；`readonly=true` 只放行 select/with/explain/show/describe/desc | `max_rows` 控制返回行数上限，超限截断带 `truncated: true` |
| `search_objects` | 探索库结构：按类型（table/view/column/index/procedure/function…）+ LIKE 模式搜对象元数据 | 给 AI "先看清库里有什么"，避免瞎猜表名 |
| `explain_sql` | 看执行计划 | opt-in，需显式声明 |
| `health_check` | 连接池/缓存命中率探测 | opt-in，需显式声明 |

只读双保险：工具级 `readonly=true`（SQL 分类器拦截，写操作返回 `READONLY_VIOLATION`）+ 数据库侧只读账号仅 SELECT。当前占位 source 连 dev pgsql 业务库（走 postgres 账号，仅工具级拦截）；接入真实库时在目标库侧建只读账号补上第二道。

### 新增工具的三种方式（纯配置，不写代码）

改完 `dbhub.toml` **热重载立即生效**（HTTP 模式含工具增减，无需重启容器）。

**① 新增一个库**：`[[sources]]` 加一段 + 给它绑工具：

```toml
[[sources]]
id = "crm"
description = "CRM 库：客户与跟进记录"
dsn = "mysql://ai_reader:${DBHUB_MYSQL_PASSWORD}@10.0.0.1:3306/dev_crm?sslmode=disable"

[[tools]]
name = "execute_sql"
source = "crm"
readonly = true
max_rows = 500

[[tools]]
name = "search_objects"
source = "crm"
```

DSN 里 `${DBHUB_MYSQL_PASSWORD}` 这类凭据变量需在 compose 的 `dbhub.environment` 加同名注入（当前仅注入 `DBHUB_PG_PASSWORD`）。

**② 启用 opt-in 内置工具**：

```toml
[[tools]]
name = "explain_sql"
source = "shop"
```

**③ 自定义工具**——把常用查询封装成具名参数化工具，AI 不用每次现编 SQL：

```toml
[[tools]]
name = "get_channel_gmv"
description = "按渠道汇总指定日期区间的 GMV"
source = "report"
statement = "SELECT channel, SUM(gmv) FROM daily_sales WHERE stat_date BETWEEN ? AND ? GROUP BY channel"

[[tools.parameters]]
name = "date_from"
type = "string"
description = "起始日期 YYYY-MM-DD"

[[tools.parameters]]
name = "date_to"
type = "string"
description = "结束日期 YYYY-MM-DD"
```

参数支持类型/必填/枚举/默认值校验；占位符按引擎：MySQL/MariaDB/SQLite 用 `?`，PG 用 `$1`，SQLServer 用 `@p1`。

### 平台联动（易漏）

DBHub 热重载只更新它自己，网关侧工具投影不会自动跟：需在管理后台「MCP 管理」对该 server 点**刷新工具**，新工具才拉进平台（已有工具的计费配置按 tool_name 保留）。

### 迁移到真实库 / 常见坑

- **换目标库**：只改 `dbhub.toml` 的 DSN（热重载生效），目标库侧建只读账号（仅 SELECT + 限库）做第二道保险。演示用 `mysql-dev` 服务已移除（2026-09-06），DBHub 目标库统一走真实场景库
- **DSN 凭据**：走 `${VAR}` 插值由 compose `environment` 注入，本文件不落明文密码
- **中文乱码**：MySQL source 带中文数据的连接配 `charset = "utf8mb4"`，否则取回乱码
