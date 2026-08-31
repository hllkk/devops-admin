import { request } from '@/service/request';

/** 成本总览(KPI 含等长上一期环比 + 按日趋势,随筛选联动) */
export function fetchGetCostOverview(params?: Api.Gateway.CostSearchParams) {
  return request<Api.Gateway.CostOverview>({
    url: '/gateway/cost/overview',
    method: 'get',
    params
  });
}

/** 成本多维明细(六维聚合,服务端分页,排序白名单降序) */
export function fetchGetCostDetail(params?: Api.Gateway.CostSearchParams) {
  return request<Api.Common.PaginatingQueryRecord<Api.Gateway.CostDetailRow>>({
    url: '/gateway/cost/detail',
    method: 'get',
    params
  });
}

/** 部门下钻成员成本(直挂口径,保证部门行=子和) */
export function fetchGetCostScopeUsers(deptId: CommonType.IdType, params?: Api.Gateway.CostSearchParams) {
  return request<Api.Gateway.CostScopeUserRow[]>({
    url: '/gateway/cost/detail/scope-users',
    method: 'get',
    params: { ...params, deptId }
  });
}

/** MCP 维工具子表(指定 server 按工具聚合) */
export function fetchGetCostMcpTools(serverId: CommonType.IdType, params?: Api.Gateway.CostSearchParams) {
  return request<Api.Gateway.CostDetailRow[]>({
    url: '/gateway/cost/detail/mcp-tools',
    method: 'get',
    params: { ...params, serverId }
  });
}
