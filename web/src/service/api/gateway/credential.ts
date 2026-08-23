import { request } from '@/service/request';

/** 分页获取凭证列表(支持 credentialName/providerId/isActive/litellmSynced 筛选) */
export function fetchGetCredentialList(params?: Api.Gateway.CredentialSearchParams) {
  return request<Api.Gateway.CredentialList>({
    url: '/gateway/credential/list',
    method: 'get',
    params
  });
}

/** 获取凭证详情(敏感值掩码，api_base 等非敏感值明文) */
export function fetchGetCredential(credentialId: CommonType.IdType) {
  return request<Api.Gateway.Credential>({
    url: `/gateway/credential/${credentialId}`,
    method: 'get'
  });
}

/** 获取供应商凭证表单字段定义(透传 LiteLLM，供前端动态渲染表单) */
export function fetchGetProviderFields() {
  return request<Api.Gateway.ProviderField[]>({
    url: '/gateway/credential/provider-fields',
    method: 'get'
  });
}

/** 新增凭证(事务内同步 LiteLLM，credentialName 全局唯一) */
export function fetchCreateCredential(data: Api.Gateway.CredentialOperateParams) {
  return request<Api.Gateway.Credential>({
    url: '/gateway/credential',
    method: 'post',
    data
  });
}

/** 修改凭证(credentialName 不可改；敏感值掩码回传=保留旧明文，新值=覆盖) */
export function fetchUpdateCredential(data: Api.Gateway.CredentialOperateParams) {
  return request<boolean>({
    url: '/gateway/credential',
    method: 'put',
    data
  });
}

/** 批量删除凭证(先删 LiteLLM 投影，失败则本地不动) */
export function fetchBatchDeleteCredential(credentialIds: CommonType.IdType[]) {
  return request<boolean>({
    url: `/gateway/credential/${credentialIds.join(',')}`,
    method: 'delete'
  });
}

/** 手动重同步全部凭证到 LiteLLM(漂移兜底) */
export function fetchResyncCredentials() {
  return request<Api.Gateway.ResyncResult>({
    url: '/gateway/credential/resync',
    method: 'post'
  });
}
