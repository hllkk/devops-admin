import { request } from '@/service/request';

/** 分页获取使用场景列表 */
export function fetchGetKeyScenarioList(params?: Api.Gateway.KeyScenarioSearchParams) {
  return request<Api.Common.PaginatingQueryRecord<Api.Gateway.KeyScenario>>({
    url: '/gateway/ai-key/scenario/list',
    method: 'get',
    params
  });
}

/** 获取启用中的场景全量(建 Key 表单下拉) */
export function fetchGetAllKeyScenarios() {
  return request<Api.Gateway.KeyScenario[]>({
    url: '/gateway/ai-key/scenario/all',
    method: 'get'
  });
}

/** 新增使用场景(name 未软删行内唯一) */
export function fetchCreateKeyScenario(data: Api.Gateway.KeyScenarioOperateParams) {
  return request<Api.Gateway.KeyScenario>({
    url: '/gateway/ai-key/scenario',
    method: 'post',
    data
  });
}

/** 修改使用场景(改名查重；停用后新建 Key 不可选) */
export function fetchUpdateKeyScenario(data: Api.Gateway.KeyScenarioOperateParams) {
  return request<Api.Gateway.KeyScenario>({
    url: '/gateway/ai-key/scenario',
    method: 'put',
    data
  });
}

/** 批量删除使用场景(被密钥引用时拒删) */
export function fetchBatchDeleteKeyScenario(scenarioIds: CommonType.IdType[]) {
  return request<boolean>({
    url: `/gateway/ai-key/scenario/${scenarioIds.join(',')}`,
    method: 'delete'
  });
}
