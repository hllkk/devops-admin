# AI 网关·DBHub 数据库 MCP 纳管（MySQL 多库）

- 日期：2026-09-05（2026-09-06 更新：dev 演示 mysql-dev 移除，目标库统一走真实场景库）
- 状态：dev 已部署并验证（docker 侧完成；平台侧注册/发布待用户页面操作）
- 关联：[[ai-gateway-mcp-server]]（MCP 管理全链路）、[[ai-gateway-mcp-sap-s4-onboard]]（stdio 型对照方案，本方案是其"远程型最优解"的实例）

## 需求

用户要纳管 MySQL 类 MCP（源自 MCP 市场调研，如 PulseMCP）。选型 Bytebase DBHub 自托管 HTTP 端点：不碰 litellm 镜像、不挂 docker.sock、与 litellm 重启解耦，避开 stdio 型三短板。另要求分析多库能力。

## DBHub 关键事实（v1.2.3 实测）

- **多库**：TOML `[[sources]]` 数组单实例多库（可跨实例、可混引擎），工具按 `source` 绑定，多 source 时工具名自动带后缀（如 `execute_sql_shop`），source 的 `description` 会拼进工具描述帮 AI 区分
- **只读双保险**：工具级 `readonly=true`（SQL 分类器拦 write/ddl，实测 DELETE 返回 READONLY_VIOLATION，只放行 select/with/explain/show/describe/desc）+ 数据库侧只读账号
- **内置工具**：execute_sql / search_objects；explain_sql / health_check 是 opt-in；其余名字+statement 即自定义参数化工具
- **安全**：`--auth-token`（Bearer，可逗号分隔多 token）+ `--allowed-hosts`（DNS 重绑定防护，dev 用 `*` 关闭）
- **热重载**：HTTP 模式改 dbhub.toml 立即生效（含工具增减），无需重启
- **不支持 Redis**：仅 PG/MySQL/MariaDB/SQLServer/SQLite；Redis 需独立 MCP server 另立项
- MCP 端点 `/mcp`（streamable HTTP），`/` 是 Web Workbench 调试台

## dev 落地配方（已部署；2026-09-06 起 mysql-dev 移除）

1. `deploy/docker-dev/docker-compose.yml` 仅 `dbhub` 一个服务（宿主 18080→8080，`--transport http --config /etc/dbhub/dbhub.toml --auth-token ${DBHUB_AUTH_TOKEN:-sk-devops-dbhub-dev} --allowed-hosts '*'`），无本地演示 MySQL——目标库统一连真实场景数据库
2. `config/dbhub.toml`：**零 [[sources]] 时 DBHub 拒绝启动（实测 Fatal）**，故留一个占位 source 连 dev pgsql 业务库 devops_admin（DSN 凭据 `${DBHUB_PG_PASSWORD}` 插值由 compose 注入）+ execute_sql/search_objects 各 readonly=true max_rows=500；接入真实库时改 DSN 热重载即生效，文件内附真实库示例段
3. 平台注册参数：serverName=`dbhub`、transport=streamable_http、url=`http://dbhub:8080/mcp`（LiteLLM 容器视角服务名）、authType=bearer、token=sk-devops-dbhub-dev；换/改 source 后工具名会变，须在「MCP 管理」点刷新工具同步投影
4. （已移除）原演示 mysql-dev（8.4，宿主 13306，dev_shop/dev_report + ai_reader 只读账号 + mysql-init/ 建库）——保留一条坑知识备用：mysql 镜像 init SQL 开头必须 `SET NAMES utf8mb4`，否则按容器 locale（latin1）解析 init 字节、中文双重编码入库

## 生产化差异（搬 prod 前必做）

- 镜像 pin：`DBHUB_VERSION` 固定具体版本（勿 latest），随 deploy/docker-prod 目录体系纳管
- `--allowed-hosts '*'` 收敛为显式主机名（若经反代暴露）；auth token 换强值并走 env
- 真实目标库：只改 dbhub.toml DSN（热重载）；目标库侧建只读账号（仅 SELECT + 限库）补齐与工具级 readonly 的双保险（占位 pg source 走 postgres 账号，仅工具级拦截）
