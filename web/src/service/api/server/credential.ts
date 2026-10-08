import { request } from '@/service/request';

/** 分页获取凭据列表 */
export function fetchGetCredentialList(params?: Api.Server.CredentialSearchParams) {
  return request<Api.Common.PaginatingQueryRecord<Api.Server.Credential>>({
    url: '/server/credential/list',
    method: 'get',
    params
  });
}

/** 获取凭据详情 */
export function fetchGetCredential(credentialId: CommonType.IdType) {
  return request<Api.Server.Credential>({
    url: `/server/credential/${credentialId}`,
    method: 'get'
  });
}

/** 获取凭据下拉选项(仅启用中,type 过滤,资产表单用) */
export function fetchGetCredentialOptions(type?: Api.Server.CredentialType) {
  return request<Api.Server.CredentialOption[]>({
    url: '/server/credential/options',
    method: 'get',
    params: type ? { type } : {}
  });
}

/** 新增凭据 */
export function fetchCreateCredential(data: Api.Server.CredentialOperateParams) {
  return request<Api.Server.Credential>({
    url: '/server/credential',
    method: 'post',
    data
  });
}

/** 修改凭据(敏感值掩码 ****** 回传=未修改保留旧明文) */
export function fetchUpdateCredential(data: Api.Server.CredentialOperateParams) {
  return request<Api.Server.Credential>({
    url: '/server/credential',
    method: 'put',
    data
  });
}

/** 批量删除凭据(被资产关联时拒删) */
export function fetchBatchDeleteCredentials(credentialIds: CommonType.IdType[]) {
  return request<boolean>({
    url: `/server/credential/${credentialIds.join(',')}`,
    method: 'delete'
  });
}
