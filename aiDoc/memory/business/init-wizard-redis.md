# 初始化向导 Redis 步

- **状态**：已实现（2026-07-13，分支 `feat/init-wizard-redis`）
- **来源**：2026-07-13 用户需求
- **关联**：设计 `docs/superpowers/specs/2026-07-13-init-wizard-redis-design.md`；实现 `docs/superpowers/plans/2026-07-13-init-wizard-redis.md`

## 需求
初始化向导由「单步合并（DB+管理员密码）」改为「**须知 → ①数据库 → ②Redis → ③管理员密码**」四屏；DB、Redis 两步各提供「测试连接」按钮（ephemeral ping，不建库、不落盘）；Redis **必填**、单实例（Addr/Password/DB）；末步一次性提交，提交后 `config.Redis` + `system.use-redis:true` 落盘并即时连接 `OPS_REDIS`。

## 关键约束（实现已落地）
- **提交模型**：末步一次性（保持原子性，中途放弃无半成品）。三步仅在前端收集数据，末步调 `/init/initdb` 一次完成建库+建表+种子+回写 config。
- **ping 端点防滥用**：`/init/db/ping`、`/init/redis/ping` 在 `global.OPS_DB != nil`（已初始化）时一律拒绝。
- **分层解耦**：`service/system` 不能反向 import `initialize`（循环），故即时连 `OPS_REDIS` 走 `dbReadyCallback`（`initialize/init.go` 扩展），非编排器直接调。
- **Redis 落盘技巧**：编排器在 `WriteConfig` 前给 `OPS_CONFIG.Redis` + `System.UseRedis` 赋值，复用 per-DB handler 的全量 `StructToMap` 回写，4 个 handler 零改。
- **Redis 范围**：只暴露单实例三字段（Addr/Password/DB）；不引入集群开关、连接池字段（YAGNI）。

## 接口
- `POST /init/checkdb`：`{needInit}`，守卫用，语义不变。
- `POST /init/db/ping`：请求 `DBConnTest`，ephemeral 连接测试。
- `POST /init/redis/ping`：请求 `PingRedis`，ephemeral 连接测试。
- `POST /init/initdb`：请求 `InitDB`（含 admin + DB + `redisAddr/redisPassword/redisDB`）。

## 待办 / 已知限制
- `MssqlEmptyDsn` 带 `database=DBName`：mssql「测试连接」需目标库已存在（mssql `EnsureDB` 本就不建库），与 mysql/pgsql 体验不一致。留后续子项目处理。
- `DialRedis` ping 失败仅返回 error（丢失了原 `initRedisClient` 的结构化 `OPS_LOG.Error`），`Redis()`/`RedisList()` 失败只剩 panic 栈。可选在 panic 前补 `OPS_LOG.Error`。
