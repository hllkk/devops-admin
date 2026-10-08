# 本机 Docker 内置资产自动登记

> 日期：2026-10-07 ｜ 状态：已实现（dev 库已插入内置行，幂等验证通过）

## 需求

「先纳管本机 Docker」决策下（[[server-module-plan]] 决策点 2），本机 Docker 是平台自有确定性资源（部署形态决定、sock 路径固定），不应让用户手动录入——启动期自动登记一条内置 docker_host 资产，开箱即见。

## 实现

- `source/server/local_docker.go`：`EnsureBuiltinLocalDocker(db)` 幂等函数 + initializer（/initdb 路径，initOrder = InitOrderSystem + 300，gateway 用 +200 避撞）
- 双路调用：重启路径 `initialize/gorm.go` RegisterTables 尾部（对齐 SeedProviderPrefix 模式：失败仅记日志不阻断启动）+ /initdb initializer 链
- MigrateTable 里列 server 两表（Asset/Credential，与 gorm.go RegisterTables 等价——/initdb 路径建表）

## 关键决策与踩坑

- **幂等键 = channel_config 内 internalKey 值子串**：不能用完整键值对 LIKE——PG jsonb 读出是规范化文本（`"key": "value"` 冒号带空格且键序重排），Go json.Marshal 无空格，两种来源都对不上；用 `LIKE '%builtin-docker-local%'`（值本身特异，跨库通用）
- **含软删查重**（Unscoped）：用户删除内置资产后重启不复活（软删行占位）；用户改名不影响幂等（internalKey 在行数据里与 asset_name 解耦）
- **manage_ip 留空**：本机资产走 sock 直连 IP 无意义，宿主信息由 slice6 采集回填；不关联凭据（sock 本身即凭证）
- dev 库已直接插入内置行（与 Go 生成数据一致），幂等查询实测命中（重启不重复）

## 后续（slice6 Docker 纳管）

- prod 需 compose 给 server 容器挂 `docker.sock`（:ro，注意等效 root 风险）；dev 的 go run 在宿主机上天然可访问
- 采集按 internalKey 定位内置行，monitor/agent 状态回写自然生效

## 相关

- [[server-module-plan]]（服务器模块规划与 slice1 记录）
