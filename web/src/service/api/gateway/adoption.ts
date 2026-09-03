import { request } from '@/service/request';

/** 覆盖率总览(KPI 含环比 + DAU 按日趋势,随筛选联动) */
export function fetchGetAdoptionOverview(params?: Api.Gateway.AdoptionSearchParams) {
  return request<Api.Gateway.AdoptionOverview>({
    url: '/gateway/adoption/overview',
    method: 'get',
    params
  });
}

/** 部门覆盖率明细(全部部门含零调用,激活/成员/消耗) */
export function fetchGetAdoptionDepartments(params?: Api.Gateway.AdoptionSearchParams) {
  return request<Api.Gateway.AdoptionDeptRow[]>({
    url: '/gateway/adoption/departments',
    method: 'get',
    params
  });
}

/** 部门成员明细下钻(含未激活成员,兼未使用人员清单) */
export function fetchGetAdoptionDeptUsers(deptId: CommonType.IdType, params?: Api.Gateway.AdoptionSearchParams) {
  return request<Api.Gateway.AdoptionUserRow[]>({
    url: `/gateway/adoption/departments/${deptId}/users`,
    method: 'get',
    params
  });
}

/** 模型分布(LLM 维,调用/成本占比) */
export function fetchGetAdoptionModels(params?: Api.Gateway.AdoptionSearchParams) {
  return request<Api.Gateway.AdoptionModelRow[]>({
    url: '/gateway/adoption/models',
    method: 'get',
    params
  });
}
