# 发布·在线升级后自动清理历史版本镜像

> 需求日期：2026-11-11。关联：[[ai-gateway-overview]]

## 背景

用户指出 SoyDisk（`/home/SoyDisk`，网盘独立项目）updater 实现了"在线升级自动清理生产环境老旧镜像"，确认存在后要求借鉴。核实：SoyDisk `deploy/updater/install.go` 有 `pruneOldImages`（历史版本镜像清理）+ `pruneOldCaches`（升级缓存清理）；devops-admin updater 同源，当年只移植了 `pruneOldCaches`，镜像清理缺失——多次在线升级后 `devops-admin/{server,web,updater}` 各版本 tag 在宿主堆积蚕食磁盘。

## 已实现（go vet / go test / go build 通过）

移植 SoyDisk 同名机制，三处改动（`deploy/updater/`）：

- `install.go`：`runInstall` 成功终态（健康检查通过 + success 落盘）后，`pruneOldCaches` 之外追加调用 `pruneOldImages(*version)`；新增 `pruneOldImages`（遍历自研三仓库 `devops-admin/server|web|updater`，`docker images --format {{.Tag}}` 列举后逐个 `docker rmi` 历史版本）+ 纯函数 `oldImageTags`（挑出非空、非当前版本、非 `<none>` 的 tag；悬空 tag 不主动碰）。
- `main_test.go`：补 `TestOldImageTags`（对齐 SoyDisk 用例）。
- `README.md`：架构图 success 分支加清理说明 + 排查表回滚行补"镜像已被清理时从发布服务器重载对应版本增量包，`docker load` 后回滚"。

## 设计决策（沿用 SoyDisk 已生产验证的语义）

- **只清自研三仓库**：pgsql/redis/rustfs/litellm 等第三方镜像不碰（litellm 本就手动升级，天然隔离）。
- **时机与容错**：只在健康检查通过后的成功终态执行；列举/删除失败只记 job 日志不回滚、不影响升级结果（旧镜像残留只是占盘）。
- **语义代价（README 已注明）**：镜像删除后「改 .env APP_VERSION 回滚」不再可用，镜像级回滚改为从发布服务器重载对应版本增量包；升级缓存同样只留当前版本。
- 与 SoyDisk 的差异仅仓库列表（devops-admin/* vs soydisk/*）与注释语境。

## 关联

- 蓝本：`/home/SoyDisk/deploy/updater/install.go`（`pruneOldImages`/`oldImageTags`/`TestOldImageTags`）
