import { request } from '@/service/request';

/** 分页获取预算规则(含读时聚合已用+预警状态) */
export function fetchGetBudgetRuleList(params?: Api.Gateway.BudgetRuleSearchParams) {
  return request<Api.Common.PaginatingQueryRecord<Api.Gateway.BudgetRuleView>>({
    url: '/gateway/budget/list',
    method: 'get',
    params
  });
}

/** 新增预算规则 */
export function fetchCreateBudgetRule(data: Api.Gateway.BudgetRuleOperateParams) {
  return request<Api.Gateway.BudgetRuleView>({
    url: '/gateway/budget',
    method: 'post',
    data
  });
}

/** 修改预算规则 */
export function fetchUpdateBudgetRule(data: Api.Gateway.BudgetRuleOperateParams) {
  return request<Api.Gateway.BudgetRuleView>({
    url: '/gateway/budget',
    method: 'put',
    data
  });
}

/** 批量删除预算规则 */
export function fetchDeleteBudgetRules(ids: CommonType.IdType[]) {
  return request<boolean>({
    url: '/gateway/budget',
    method: 'delete',
    data: { ids }
  });
}

/** 三维度预算汇总(Key/部门/用户) */
export function fetchGetBudgetSummary(params?: { scope?: string }) {
  return request<{ keys: Api.Gateway.BudgetItem[]; depts: Api.Gateway.BudgetRuleView[]; users: Api.Gateway.BudgetRuleView[] }>({
    url: '/gateway/budget/summary',
    method: 'get',
    params
  });
}
