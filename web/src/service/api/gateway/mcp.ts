import { request } from '@/service/request';

// ── MCP 服务器 MCPServer(AI 市场 P2) ──

/** 获取用户侧可见 MCP 列表(active+published+按发布可见性过滤;模型广场 MCP 分区用) */
export function fetchGetActiveMcps() {
  return request<Api.Gateway.AvailableMcp[]>({
    url: '/gateway/mcp/active',
    method: 'get'
  });
}

/** 获取可授权 MCP 列表(管理端,全部启用;Key 授权下拉用) */
export function fetchGetAvailableMcps() {
  return request<Api.Gateway.AvailableMcp[]>({
    url: '/gateway/mcp/available',
    method: 'get'
  });
}

/** 分页获取 MCP 服务器列表(支持 name/category/isActive/isPublished/healthStatus 筛选) */
export function fetchGetMCPServerList(params?: Api.Gateway.MCPServerSearchParams) {
  return request<Api.Gateway.MCPServerList>({
    url: '/gateway/mcp/list',
    method: 'get',
    params
  });
}

/** 获取 MCP 分类去重列表(下拉受控数据源) */
export function fetchGetMCPCategories() {
  return request<string[]>({
    url: '/gateway/mcp/categories',
    method: 'get'
  });
}

/** 获取 MCP 服务器详情(含工具列表) */
export function fetchGetMCPServer(mcpServerId: CommonType.IdType) {
  return request<Api.Gateway.MCPServerDetail>({
    url: `/gateway/mcp/${mcpServerId}`,
    method: 'get'
  });
}

/** 注册 MCP 服务器(同步 LiteLLM MCP 网关,allow_all_keys 恒 false) */
export function fetchCreateMCPServer(data: Api.Gateway.MCPServerOperateParams) {
  return request<Api.Gateway.MCPServer>({
    url: '/gateway/mcp',
    method: 'post',
    data
  });
}

/** 修改 MCP 服务器(凭据掩码回传=保留旧明文;停用会回收主 Key 授权) */
export function fetchUpdateMCPServer(data: Api.Gateway.MCPServerOperateParams) {
  return request<Api.Gateway.MCPServer>({
    url: '/gateway/mcp',
    method: 'put',
    data
  });
}

/** 批量删除 MCP 服务器(先删 LiteLLM→回收主 Key 授权→本地软删) */
export function fetchBatchDeleteMCPServer(mcpServerIds: CommonType.IdType[]) {
  return request<boolean>({
    url: `/gateway/mcp/${mcpServerIds.join(',')}`,
    method: 'delete'
  });
}

/** 获取发布设置(含 selected/user 模式的可见部门/用户回显) */
export function fetchGetMCPPublish(mcpServerId: CommonType.IdType) {
  return request<Api.Gateway.MCPPublishView>({
    url: `/gateway/mcp/publish/${mcpServerId}`,
    method: 'get'
  });
}

/** 更新发布设置(发布免审批自动授权按可见档主 Key,双向对齐) */
export function fetchPublishMCPServer(data: Api.Gateway.MCPPublishParams) {
  return request<boolean>({
    url: '/gateway/mcp/publish',
    method: 'put',
    data
  });
}

/** 刷新 MCP 工具列表(远端全量重建,按 tool_name 保留计费配置) */
export function fetchRefreshMCPTools(mcpServerId: CommonType.IdType) {
  return request<Api.Gateway.MCPTool[]>({
    url: `/gateway/mcp/${mcpServerId}/refresh-tools`,
    method: 'post',
    timeout: 30000
  });
}

/** 更新工具级计费(空=继承服务器默认;internalCostPerCall null=继承/同外部价) */
export function fetchUpdateMCPToolBilling(
  mcpToolId: CommonType.IdType,
  data: { billingType: string; externalCostPerCall: number | null; internalCostPerCall: number | null }
) {
  return request<Api.Gateway.MCPTool>({
    url: `/gateway/mcp/tool/${mcpToolId}/billing`,
    method: 'put',
    data
  });
}

/** MCP 服务器健康检查(经 LiteLLM 服务端代理探测) */
export function fetchHealthCheckMCPServer(mcpServerId: CommonType.IdType) {
  return request<Api.Gateway.MCPServer>({
    url: `/gateway/mcp/${mcpServerId}/health-check`,
    method: 'post',
    timeout: 30000
  });
}

/** 获取 MCP 接入配置(用户侧,含主 Key 明文鉴权头的客户端配置 JSON) */
export function fetchGetMCPConnectConfig(mcpServerId: CommonType.IdType) {
  return request<Api.Gateway.MCPConnectConfig>({
    url: `/gateway/mcp/connect-config/${mcpServerId}`,
    method: 'get'
  });
}
