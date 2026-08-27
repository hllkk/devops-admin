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

/** 轮换密钥(原地换 Key 值保归因；旧 Key 立即失效，新明文仅 owner 经 identity/my 可查) */
export function fetchRotateAiKey(aiKeyId: CommonType.IdType) {
  return request<Api.Gateway.AiKey>({
    url: `/gateway/ai-key/rotate/${aiKeyId}`,
    method: 'post'
  });
}

/** 查看密钥完整明文(仅管理员/超管，解密 key_value 按需返回；操作日志审计) */
export function fetchRevealAiKeyValue(aiKeyId: CommonType.IdType) {
  return request<Api.Gateway.AiKeyReveal>({
    url: `/gateway/ai-key/value/${aiKeyId}`,
    method: 'get'
  });
}

/** 批量开通个人主 Key(按部门/按用户；已有跳过，部分失败不中断) */
export function fetchBatchCreateMainKeys(data: Api.Gateway.AiKeyBatchCreateParams) {
  return request<Api.Gateway.AiKeyBatchCreateResult>({
    url: '/gateway/ai-key/batch',
    method: 'post',
    data,
    timeout: 0 // 批量按部门开通可能数百用户串行外呼 LiteLLM,不吃默认 10s 超时
  });
}
