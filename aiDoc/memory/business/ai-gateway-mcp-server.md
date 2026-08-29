# AI 网关 · MCP Server 管理(P2)

日期：2026-08-28
状态：已实现(后端+前端，待用户运行时验证)
（菜单结构后续已变更：本页改名「MCP 管理」并挂入「AI能力」目录，见 [[ai-gateway-ai-capability-menu]]）

## 需求

P2「AI 市场」第二切片：MCP Server 统一注册/发布/授权/工具/健康，打通
`resource_type=mcp` 审批公共底座与 `gateway_ai_key.mcps` P1 预留字段。
参照 AIHelms(`/home/remote/AIHelms` apps/api/v1/mcp.py + services/mcp_service.py)。

## 后端设计要点

- **四张新表**：`gateway_mcp_server`(主表,server_name=LiteLLM 路由键唯一禁`-`拒绝改名,
  litellm_server_id 归因锚点自定义 `gw_mcp_{雪花}`,credentials AES-GCM 密文列 json:"-")
  + `gateway_mcp_tool`(远端投影表,(server_id,tool_name) 唯一,namespaced={serverName}_{toolName})
  + `gateway_mcp_visibility` / `gateway_mcp_visibility_user`(三档可见性投影,物理删重建,
  与模型同构)。RegisterTables 已注册。
- **LiteLLM 客户端新增**：CreateMCPServer(POST /v1/mcp/server)/UpdateMCPServer(PUT)/
  DeleteMCPServer(DELETE)/ListMCPServers/TestMCPConnection 与 ListMCPToolsFromServer
  (POST /mcp-rest/test/*,litellm 对探测失败也回 200+status=error 须查 body)。
  KeyUpdateReq 加 SyncMCPServers 强制刷(含空数组→LiteLLM 清权,已实测空数组推送后
  allowed_mcp_servers=None)。
- **授权架构(规避 AIHelms 三坑)**：LiteLLM 侧 server 恒 `allow_all_keys=false`，
  Key.allowed_mcp_servers 是唯一授权凭证；mcps JSONB 存 **serverName 字符串**(与 models 存
  modelKey 同构,免解析、改名本就拒绝)；syncKeyToLitellm 恒推(SyncMCPServers=true)。
- **发布三档可见性+需审批**：`PublishMCPServer` 与 PublishModel 同构(校验/投影重建/
  mainKeyScopeOf 复用)；`alignMCPAuthorization` 收口：发布+免审批+启用=sync+revoke(scope)，
  否则 revoke(all)。回收扫全部主 Key 含停用；场景 Key 手工授权不动。
- **启停联动**：UpdateMCPServer 停用=全量回收授权，恢复=按发布档重授(Update 内查投影表现值)。
- **删除**：先 LiteLLM DELETE(失败中止本地不动；未同步跳过)→事务内 revoke all+软删主行+
  物理删工具/可见性行。
- **工具刷新**：RefreshMCPTools 远端全量重建(Unscoped 物理删防唯一索引占位)，按 tool_name
  保留计费配置(CollectMCPToolBilling/BuildMCPTools 纯函数)，刷新后重推 mcp_info。
- **计费投影**：`MCPCostInfo` 组装 mcp_info.mcp_server_cost_info{default_cost_per_query,
  tool_name_to_cost_per_query}，¥÷汇率(6位舍入)→USD；free 或无价不下发。MCP 调用成本
  回流入账留 P3。
- **健康检查**：经 LiteLLM 服务端代理(平台不直连)，结果落 health_status/last_health_check/
  health_check_error。
- **审批接入**：resource_application.go Create 拆 `validateResourceVisible`(model/mcp 各自
  三档可见性+需审批校验)；review 通过按类型分派 syncModelToMainKeys/syncMcpToMainKeys；
  fillApplicationViews 批量回填 MCP 名(四次 IN 查询防 N+1)。
- **identity/广场**：MyIdentityView 加 mcps/availableMcps；GET /gateway/mcp/active(可见列表)、
  /gateway/mcp/connect-config/:id(客户端配置 JSON,主 Key 明文作 Bearer)入 casbin 登录白名单；
  GET /gateway/mcp/available(管理端全量)与 available 走菜单授权。
- **菜单**：sys_menu 加顶层 C `route.mcp`(path=mcp,Component=_gateway/mcp/index,
  ApiPrefix=/gateway/mcp, /gateway/mcp/*,OrderNum=13)。仅新库生效,已有库手动补。
- **凭据语义**：编辑掩码回传=保留旧明文(MergeCredentialValues 复用)；切 authType=none=
  显式清空(前端 null→后端清,PUT LiteLLM credentials:null 已实测接受)；更新用显式
  Updates(map) 不用 Save(防审计字段零值覆盖)。
- 纯函数层 `service/gateway/mcp_payload.go`(5 单测)：ValidMCPServerName(`^[a-zA-Z0-9_]+$`)
  /NormalizeMCPTransport(空→streamable_http,streamableHttp|http→streamable_http,下发映射 http)
  /MCPCostInfo/CollectMCPToolBilling/BuildMCPTools。

## 前端要点

- 管理页 `views/_gateway/mcp/`(index 表格页+mcp-search+mcp-operate-drawer+
  mcp-publish-dialog 克隆模型发布弹窗+mcp-tools-drawer 工具计费)；常量
  MCP_TRANSPORT/AUTH_TYPE/BILLING/HEALTH_OPTIONS 收口 constants/business/gateway.ts。
- ai-key 抽屉加 MCP 授权多选(value=serverName,label 附展示名)。
- 审批页 resourceType 列/筛选项支持 mcp。
- home 身份页 MCP 区切真实(可见 tag+授权态,未开通也展示)；模型广场面板加
  模型/MCP 单选切换,MCP 卡片+接入信息弹窗(mcpUrl/工具清单/配置 JSON 复制不回显,
  Base URL+掩码 Key 与模型共用区块)；appliedIds 改 `${type}:${id}` 键防两类资源撞车。
- i18n 三处同步：route.mcp、page.gateway.mcp.*、page.home.square.{filterModels,filterMcps,
  accessMcpUrl,accessMcpConfig,accessMcpConfigTip,copyConfig,toolsCount}、
  page.gateway.application.typeMcp、page.gateway.aiKey.{col.mcps,form.mcpsPlaceholder}；
  typings gateway.api.d.ts 加 MCPServer/MCPTool/AvailableMcp/MCPConnectConfig 等,
  AiKey.mcps 修正为 string[](P1 误标 number[])。
- 路由类型 `mcp` 由运行中的 vite dev 的 elegant-router 插件自动再生成(RouteMap/routes.ts)。

## 验证

- go build/vet/test 全过(含 mcp_payload 5 单测+路由冒烟 mcp_test.go)。
- pnpm typecheck/eslint(oxlint) 0 错误。
- LiteLLM 1.98.0 实测(dev)：/v1/mcp/server CRUD 201/202、自定义 server_id 接受、
  PUT credentials:null 清凭据、key allowed_mcp_servers 空数组=清权、/mcp-rest/test/* 可用。

## 待办(关联)

- 用户运行时点触验证(菜单需补:已有库手动加 route.mcp 菜单或重置菜单种子)。
- swag 注释已写 @Tags GatewayMCP,待 swag 重生成。
- MCP resync 全量投影比对兜底(credential 有先例)未做,记技术债。
- P2 剩余：Skill 管理；批量建场景 Key/复制主 Key 模板。
