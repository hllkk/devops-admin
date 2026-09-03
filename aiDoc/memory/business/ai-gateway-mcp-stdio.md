# AI 网关·MCP stdio 型服务器支持

- 日期：2026-09-03
- 状态：已实现（build/test+typecheck 全过；LiteLLM stdio 链路经 dev 实测，平台级链路待重启 dev 后端后验证）
- 关联：[[ai-gateway-mcp-server]]（MCP 管理与 LiteLLM 同步）、[[ai-gateway-mcp-usage-sync]]（日志回流，stdio 调用同格式已实测）

## 需求

MCP 管理此前仅支持远程端点（sse/streamable_http），无法纳管本地子进程形态的 MCP server（如 `uvx mcp-server-fetch`、`npx -y @xxx`）。本次支持 stdio 型注册/发布/授权/工具/健康全链路。

## 前置实测结论（dev LiteLLM 1.99.0，2026-09-03）

- `POST /v1/mcp/server` 收 `transport:"stdio"+command/args/env`（`NewMCPServerRequest` schema 已含三字段），注册/读回/删除均通过；
- `/mcp-rest/test/connection`、`/mcp-rest/test/tools/list` body 同 schema，stdio 探测可用（真实 spawn 子进程 + initialize + tools/list 走通）；
- 用户主路径 `/{server_name}/mcp` streamable HTTP → initialize + tools/call 全通（FastMCP echo 实测返回）；
- **stdio command 白名单**：`deno/docker/node/npx/python/python3/uvx`，任意二进制/带路径写法被上游拒绝（这是 LiteLLM 安全边界，平台照抄同清单前置拦截）；
- SpendLogs 回流：stdio 调用同样落 `mcp_namespaced_tool_name`（`server/tool` 斜杠格式）与 `model="MCP: <tool>"`，现有归因/回流零改动兼容；
- dev 容器内运行时：python3（venv）+ mcp py lib + node 在；uvx/npx/deno/docker 不在——纳管 uvx/npx 型需扩镜像或换 command 形态。

## 设计决策

1. **架构零变化**：stdio 子进程由 LiteLLM 容器托管，对外仍暴露 `/{server_name}/mcp` 统一端点。授权（allowed_mcp_servers）、connect-config（用户拿到的仍是 URL 型 mcpServers JSON）、日志回流、计费全部复用，用户侧零部署零感知。
2. **env 即凭据**：stdio 无上游 HTTP 鉴权概念，`authType` 恒 none；env 变量存 `Credentials` 列（AES-256-GCM 同套加密/掩码/合并，`IsSensitiveKey` 对 TOKEN/KEY 类 env 名天然命中掩码）。投影时 stdio 发 `env` 键、不发 `credentials`。
3. **transport 可编辑切换**：投影体显式带全字段（stdio 时 url/auth_type/credentials=null，http 时 command=null），切换形态不留旧字段残留；validateOperate 同步清本地列（stdio 时 url 置空、http 时 command 置空）。
4. **不引入外部桥接层**（supergateway/mcp-proxy）：LiteLLM 原生支持后无必要；也不做"平台只发 stdio 配置片段给用户自跑"的目录模式——绕过网关直跑会丢授权/计量。

## 改动清单

- 后端：`model/gateway/mcp.go`（+`MCPTransportStdio`、`Command`/`Args` 列，AutoMigrate 加列）、`request/mcp.go`（+command/args）、`mcp_payload.go`（NormalizeMCPTransport/litellmMCPTransport 三态、`ValidMCPStdioCommand` 白名单、`BuildMCPEndpointSpec` 探测组装）、`utils/litellm/client.go`（探测签名收 `MCPEndpointSpec` 结构体，stdio 发 command/args/env）、`service/gateway/mcp.go`（validateOperate 分支、Update 凭据合并 stdio 例外——authType 恒 none 不能走"切 none 清空"通道、Updates map +command/args、buildMCPLitellmBody 双形态投影）
- 前端：`gateway.api.d.ts`（MCPTransport 三态+MCPServer+OperateParams）、`MCP_TRANSPORT_OPTIONS`+`MCP_STDIO_COMMAND_OPTIONS` 常量、operate-drawer stdio 分支（command 白名单下拉/args 动态行/env 键值对编辑器，掩码回传保留语义）、i18n 三处同步
- 测试：mcp_payload_test 补 NormalizeMCPTransport(stdio)/ValidMCPStdioCommand/BuildMCPEndpointSpec 用例

## 已知限制 / 待办

- env 键删除不支持（浅 merge 语义，与 http 型凭据键一致的限制）；stdio 也无"清空全部 env"显式通道
- 容器运行时收窄：第一期建议优先 python/python3 型（litellm 镜像原生带）；uvx/npx 型需扩镜像；`command:"docker"` 型（每 server 独立容器隔离）需挂 docker.sock，安全面扩大，单独立项决策
- stdio 单进程单会话，LiteLLM 多用户并发的进程管理语义未明（社区有 issue #19035 报 424），重负载场景上线前需压测
- LiteLLM `${X-HEADER}` 入站头映射子进程 env 机制（stdio 版 per-user 凭证/BYOK）留后续
- 平台级端到端（经 devops-admin 页面/API 注册 stdio server → 发布 → 接入）待 dev 后端重启新二进制后验证
