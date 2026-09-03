import { request } from '@/service/request';

/** 健康检查汇总(四卡:MCP上游/模型部署/基础组件/数据回流新鲜度+三块明细) */
export function fetchGetHealthSummary() {
  return request<Api.Gateway.HealthSummary>({
    url: '/gateway/health/summary',
    method: 'get'
  });
}

/** 手动巡检全部模型部署(按路由组探测落库,返回检查的路由组数) */
export function fetchHealthCheckDeployments() {
  return request<number>({
    url: '/gateway/health/check-deployments',
    method: 'post'
  });
}
