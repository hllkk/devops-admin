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

/** 获取成本 Top10(按维度 user/model/aiKey) */
export function fetchGetDashboardTop(params?: DashboardQueryParams & { dimension?: 'user' | 'model' | 'aiKey' }) {
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

/** 手动触发用量聚合(滚动重建 + budget 重算 + 超限停用闭环) */
export function fetchAggregateUsage() {
  return request<{ rebuilt: number; keysRecomputed: number; keysDisabled: number }>({
    url: '/gateway/dashboard/aggregate',
    method: 'post'
  });
}
