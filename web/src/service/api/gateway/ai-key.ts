import { request } from '@/service/request';

/** 分页获取密钥列表(管理员视角，不返回 KeyValue) */
export function fetchGetAiKeyList(params?: Api.Gateway.AiKeySearchParams) {
  return request<Api.Common.PaginatingQueryRecord<Api.Gateway.AiKey>>({
    url: '/gateway/ai-key/list',
    method: 'get',
    params
  });
}

/** 获取密钥详情(管理员视角，不返回 KeyValue) */
export function fetchGetAiKey(aiKeyId: CommonType.IdType) {
  return request<Api.Gateway.AiKey>({
    url: `/gateway/ai-key/${aiKeyId}`,
    method: 'get'
  });
}

/** 创建密钥(场景 Key 或管理员手动建部门主 Key) */
export function fetchCreateAiKey(data: Api.Gateway.AiKeyOperateParams) {
  return request<Api.Gateway.AiKey>({
    url: '/gateway/ai-key',
    method: 'post',
    data
  });
}

/** 修改密钥(授权/预算/限流/启停；keyType/ownerType/ownerId 不可改) */
export function fetchUpdateAiKey(data: Api.Gateway.AiKeyOperateParams) {
  return request<boolean>({
    url: '/gateway/ai-key',
    method: 'put',
    data
  });
}

/** 批量删除密钥(先删 LiteLLM，失败则本地不动) */
export function fetchBatchDeleteAiKey(aiKeyIds: CommonType.IdType[]) {
  return request<boolean>({
    url: `/gateway/ai-key/${aiKeyIds.join(',')}`,
    method: 'delete'
  });
}
