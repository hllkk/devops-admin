# 服务器模块·agent 重启/卸载 + 资产查看抽屉

> 日期：2026-10-08 ｜ 状态：已实现（build/vet/test + typecheck/lint 全过，待运行时验证）

## 需求

用户要求：agent 支持卸载和重启；资产行操作精简为 查看/编辑/删除；「查看」开抽屉展示服务器基本信息 + CPU/内存/网络/磁盘状态 + agent 状态，抽屉内提供 重启 Agent / 卸载 Agent。

## 实现

### agent 运维（service/server/agent_ops.go）

- `StartRestart`/`StartUninstall` 复用安装流全套：SSH 公钥通道 + Redis 异步任务状态（`server:agent-install:<taskId>`，与安装**同一轮询接口** `GET /server/asset/install-status/:taskId`，status/step/message 结构不变）
- 重启：`systemctl restart` + `is-active` 校验；超时 30s（纯命令操作）
- 卸载：`disable --now` → 删 unit/二进制/`/var/lib/aiops-agent` → 服务端清 `agent_token_hash/agent_version/agent_hostname/last_heartbeat_at`、状态回 none → 删 Redis 快照。**平台公钥不回收**（SSH 信任保留，可随时重装）
- 前置校验：资产已 SSH 验证 + 已安装 agent（未装报错）

### 资源快照（轻量版，slice3 前置）

- **agent 端**（`cmd/agent/collect.go`）：/proc 只读采集——CPU%（/proc/stat 差值）/负载/内存（meminfo）/根分区磁盘（statfs）/网络速率（/proc/net/dev 差值÷心跳间隔，排除 lo）/uptime；首帧无差值为 0
- **心跳携带**：`heartbeat` 请求体加可选 `metrics` 字段（旧 agent 不带，兼容）
- **服务端**：心跳写 Redis `server:asset-snapshot:<assetId>`（JSON，TTL=2×心跳间隔+60s，心跳停则过期）+ `:at` 时间键；查看接口 `GET /server/asset/:id/snapshot` 返回（stale 标记）
- **不进 PG**：快照性质数据；历史趋势/告警评估留给 slice3 完整管道（Redis 热窗口+PG 降采样），与本实现无耦合

### 前端（asset-detail-drawer.vue）

- 操作列精简：查看/编辑/删除（安装入口移入抽屉）
- 抽屉四区：基本信息（NDescriptions：类型/IP/SSH 用户名/验证状态/OS/环境/位置/启用/监控状态/描述）→ Agent 状态（状态 tag/版本/主机名/最近心跳）→ 资源状态（CPU/内存/磁盘卡片带 NProgress 与阈值变色、网络速率、uptime；stale 显示「暂无实时数据」空态；打开期间 30s 自动刷新）→ Agent 运维（按钮：未装=安装[弹窗]、已装=重启+卸载[确认对话框+内联进度]；操作完成 emit changed 刷新列表）

## 验证

- go build/vet/test 全过；typecheck/oxlint/eslint 0 错误；agent 双架构二进制已重建（含采集器）
- 待用户运行时验证：安装→抽屉见实时资源状态→重启→卸载→状态回未安装

## 相关

- [[server-asset-verify-on-save]]（SSH 验证通道是运维操作的前置）
- [[server-module-plan]]（slice2 记录）
