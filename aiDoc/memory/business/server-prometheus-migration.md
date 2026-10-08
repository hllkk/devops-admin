# 服务器模块·Prometheus 架构转型（自研 agent 退役）

> 日期：2026-10-08 ｜ 状态：已实现（go build/vet/test + typecheck/lint + 路由冒烟全过），待用户 dev 运行时验证

## 需求与决策

用户质疑自研采集必要性（「spug 都不用 agent」到「能不能集成 Prometheus」），对比 OTel vs Prometheus 后决策：**数据面转 Prometheus pull 架构**（node_exporter + Prometheus 容器），OTel 否决（定位应用 APM 统一管道，OS 监控单一场景杀鸡用牛刀，且不带存储/查询/告警）。这是 AI 网关选 LiteLLM 底座同款「复用成熟开源」决策。

用户场景确认：安装采集端时自动放行目标机防火墙 9100（firewalld/ufw 幂等，云安全组控制台另行）；docker 集成 Prometheus 可行。

## 架构（最终形态）

```
平台资产表 ──HTTP SD(Bearer sd-token)──→ Prometheus 容器 ──pull──→ 150台 node_exporter:9100
     ↑                                          │
     └─SyncMonitorStatus 定时(*/2min)刮 up 回写   └─后端代查 API(instant/range)→ 快照/趋势
资产监控状态 online/offline + 采集端 running/lost
```

## 转型落点

- **退役**：aiops-agent 程序（cmd/agent）、agent 注册/心跳（agent_registry.go）、Redis 快照/热窗口、server_metric_minute 分钟表+聚合/清理任务、CheckLostAgents、公开组 register/heartbeat 端点、server.agent 的 heartbeat/lost/server-url 配置
- **保留/改造**：SSH 公钥通道与录入验证（管理面不变，P4 公钥授权不受影响）；安装流（agent_install.go 重写：探测架构→sftp 上传 node_exporter→systemd unit(:9100)→**放行防火墙**→curl 校验+提版本）；重启/卸载（agent_ops.go 重写：卸载撤防火墙端口+清字段；平台公钥不回收可重装）；Asset 的 agent_* 列语义改为「采集端(node_exporter)」
- **新增**：`service/server/prometheus.go`（PrometheusService：query/queryRange 封装、GetSnapshot/GetTrend **保持自研时代接口字段形态（前端零改动）**、SDTargets、SyncMonitorStatus）；`GET /prometheus/sd` 公开组端点（Bearer sd-token，返回 HTTP SD 格式 targets）；定时任务 SyncMonitorStatus（*/2min 种子）
- **部署**：prod/dev compose 均加 prometheus 服务（v3.7.1，数据卷，retention 30d，`--web.enable-lifecycle`）；**prometheus.yml 不支持 env 插值——SD token 用 entrypoint `printf > /run/sd.token` + `credentials_file`（踩坑记档）**；dev 经 `host.docker.internal:host-gateway` 访问宿主 go run 的 8888；node_exporter 下载脚本 `scripts/download-node-exporter.sh`（官方源→gh-proxy 回退，dev 产物已就位）+ prod Dockerfile 构建期下载进镜像 resource/node-exporter/
- **配置**：server.agent 精简（install-timeout/binary-dir/ssh-key-dir/exporter-port 9100）+ server.prometheus 新段（base-url/sd-token，prod env 覆盖 PROMETHEUS_BASE_URL/PROMETHEUS_SD_TOKEN）
- **前端**：接口路径/字段形态不变零逻辑改动；文案「Agent」→「采集端」（i18n zh/en 同步）

## 告警路线（slice4 调整）

原「自研阈值评估器」改为 **Prometheus alerting rules + Alertmanager webhook 回流平台通知体系**（企微/站内信复用），比自研更成熟。

## 相关

- [[server-agent-ops-detail-drawer]]（查看抽屉——快照/趋势数据源换 Prometheus，接口形态未变）
- [[server-asset-verify-on-save]]（SSH 验证通道是安装/运维前置）
- [[server-module-plan]]（模块总规划——slice3 的自建管道设计被本次转型取代）
