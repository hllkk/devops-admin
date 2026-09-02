import { request } from '@/service/request';

/** 看板通用查询参数(日期范围 + 范围 self/all) */
export interface DashboardQueryParams {
  startDate?: string;
  endDate?: string;
  scope?: 'all' | 'self';
}

/** 获取看板总览(总成本/请求数/token/预算汇总) */
export function fetchGetDashboardOverview(params?: DashboardQueryParams) {
  return request<Api.Gateway.DashboardOverview>({
    url: '/gateway/dashboard/overview',
    method: 'get',
    params
  });
}

/** 获取成本趋势(按日) */
export function fetchGetDashboardTrend(params?: DashboardQueryParams) {
  return request<Api.Gateway.TrendItem[]>({
    url: '/gateway/dashboard/trend',
    method: 'get',
    params
  });
}

/** 获取 Top10 排行(按维度 user/model/aiKey,排序键 cost/requests/tokens) */
export function fetchGetDashboardTop(
  params?: DashboardQueryParams & { dimension?: 'user' | 'model' | 'aiKey'; sort?: 'cost' | 'requests' | 'tokens' }
) {
  return request<Api.Gateway.TopItem[]>({
    url: '/gateway/dashboard/top',
    method: 'get',
    params
  });
}

/** 获取预算执行率(按 Key) */
export function fetchGetDashboardBudget(params?: { scope?: 'all' | 'self' }) {
  return request<Api.Gateway.BudgetItem[]>({
    url: '/gateway/dashboard/budget',
    method: 'get',
    params
  });
}

/** 手动触发用量聚合(先回流 LLM+MCP 日志再滚动重建 + budget 重算 + 超限停用闭环) */
export function fetchAggregateUsage() {
  return request<{ synced: number; rebuilt: number; keysRecomputed: number; keysDisabled: number }>({
    url: '/gateway/dashboard/aggregate',
    method: 'post',
    // 聚合含回流+60天滚动重建,耗时可能超过 axios 默认 10s,单独放开超时
    timeout: 0
  });
}

/** 获取跨供应商套餐余量汇总(厂商侧旁路口径,非超管返回空) */
export function fetchGetBalanceSummary() {
  return request<Api.Gateway.ProviderBalanceSummary[]>({
    url: '/gateway/dashboard/balance-summary',
    method: 'get'
  });
}
