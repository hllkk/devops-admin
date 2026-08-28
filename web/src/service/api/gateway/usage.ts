import { request } from '@/service/request';

/** 分页获取用量日志(管理员视角，按用户/密钥/渠道/模型/供应商/时间过滤) */
export function fetchGetUsageLogList(params?: Api.Gateway.UsageLogSearchParams) {
  return request<Api.Common.PaginatingQueryRecord<Api.Gateway.UsageLog>>({
    url: '/gateway/usage/list',
    method: 'get',
    params
  });
}

/** 手动触发用量回流(归因+成本重算，复合游标增量幂等) */
export function fetchSyncLLMLogs() {
  return request<Record<string, number>>({
    url: '/gateway/usage/sync',
    method: 'post',
    timeout: 0
  });
}

/** 手动触发对账回灌 LiteLLM 漏单(近30天 NOT EXISTS 兜底) */
export function fetchReconcileLLMLogs() {
  return request<Record<string, number>>({
    url: '/gateway/usage/reconcile',
    method: 'post',
    timeout: 0
  });
}
