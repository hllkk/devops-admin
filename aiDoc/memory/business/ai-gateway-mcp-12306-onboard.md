# AI 网关·12306-mcp 纳管（协议白名单 patch）

- 日期：2026-09-07
- 状态：dev 全链路已打通（服务侧 patch 完成；平台刷新工具/后端重启修 Bug2 待用户操作）
- 关联：[[ai-gateway-mcp-server]]、[[ai-gateway-mcp-stdio]]（12306 是 http 型部署，stdio 形态禁入不变）、[[ai-gateway-mcp-dbhub-onboard]]（同属自托管 MCP 栈）

## 需求与部署形态

用户纳管 [Joooook/12306-mcp](https://github.com/Joooook/12306-mcp)（车票/经停/中转查询，8 工具）。
stdio 形态（`npx -y 12306-mcp`）走不通（litellm 容器无 npx，禁入不变）；采用官方 Docker-http 模式：
宿主 `docker run -d --name mcp12306 --restart unless-stopped -p 12306:8080 12306-mcp:patched npx 12306-mcp --port 8080`。
镜像 build context 在宿主 `/home/mcp/12306-mcp`（clone 的仓库，Dockerfile 已加 patch 层）。

## 坑：嵌套 SDK 协议白名单不一致（排查 2026-09-07）

**症状链**：平台健康检查落 `health_check_error="ok:"`（=Bug2 误判，探测实际成功）+ 经 LiteLLM 的
`/mcp-rest/test/tools/list` 返回 error，litellm 日志 `httpx.HTTPStatusError: 400`。
`test/connection` 侥幸通过 → 健康"假阴/假阳"都不能代表工具链路可用。

**根因**：12306-mcp 的 `mcp-http-server@1.2.4` **嵌套**了自己专用的
`@modelcontextprotocol/sdk@1.15.1`（顶层 sdk 已是 1.30，白名单含 latest）。
嵌套 1.15.1 的 `SUPPORTED_PROTOCOL_VERSIONS` 止于 2025-06-18，而 LiteLLM 的 MCP 客户端
（python sdk `session.py:187`）**写死以 latest=2025-11-25 发起 initialize**：
协商被宽松回显接受，但后续请求带 `mcp-protocol-version: 2025-11-25` 头被
`streamableHttp.js validateProtocolVersion` 以 400 拒绝（协商接受/校验拒绝自相矛盾）
→ tools/list 与 tools/call 全 400。

**定位方法**：直连（普通 http 客户端逐请求复刻）→ 不带 `mcp-protocol-version` 头全通，
带上即 400 → 用容器内同款 python sdk 复刻完整会话逐头测 → 三组协商版本对照
（2025-06-18 全通 / 2025-11-25 400）实锤。**别只看 litellm 日志的状态码，400 响应体里的
supported versions 列表直接指认白名单来源。**

**patch**（已固化进 Dockerfile 层，上游修复后删该层即可）：
`sed` 把 `"2025-11-25",` 插进嵌套 sdk `dist/{esm,cjs}/types.js` 的
`SUPPORTED_PROTOCOL_VERSIONS` 数组（esm+cjs 两份都要，1.15.1 是多行数组格式），
后接 `grep -q` 校验——将来 mcp-http-server 不再嵌套旧 sdk 时构建会显式失败而非静默丢 patch。

**容器注意**：官方 Dockerfile 无 CMD，`docker run` 必须显式带 `npx 12306-mcp --port 8080`，
否则容器立即退出(0)进 crash loop、logs 全空。

## 平台侧状态（dev）

- 注册参数：serverName=`12306_mcp`、transport=streamable_http、
  url=`http://172.21.10.41:12306/mcp`（LiteLLM 容器视角须用宿主 IP，不能 127.0.0.1）、authType=none
- `litellm_synced=t`（注册同步成功）；工具投影 0 行（刷新工具待用户点）
- 经 LiteLLM 终测：tools/list 8 工具全通 + tools/call(get-current-date)=2026-09-07（2025-11-25 会话）
- 待办：dev 后端重启带 Bug1/Bug2 修复（mcp_info 空对象 + ok 前缀）；生产上线走发版
