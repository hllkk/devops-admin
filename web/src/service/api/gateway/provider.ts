import { request } from '@/service/request';

/** 分页获取供应商列表 */
export function fetchGetProviderList(params?: Api.Gateway.ProviderSearchParams) {
  return request<Api.Gateway.ProviderList>({
    url: '/gateway/provider/list',
    method: 'get',
    params
  });
}

/** 获取供应商详情 */
export function fetchGetProvider(providerId: CommonType.IdType) {
  return request<Api.Gateway.Provider>({
    url: `/gateway/provider/${providerId}`,
    method: 'get'
  });
}

/** 新增供应商(返回创建后的供应商) */
export function fetchCreateProvider(data: Api.Gateway.ProviderOperateParams) {
  return request<Api.Gateway.Provider>({
    url: '/gateway/provider',
    method: 'post',
    data
  });
}

/** 修改供应商 */
export function fetchUpdateProvider(data: Api.Gateway.ProviderOperateParams) {
  return request<boolean>({
    url: '/gateway/provider',
    method: 'put',
    data
  });
}

/** 批量删除供应商 */
export function fetchBatchDeleteProvider(providerIds: CommonType.IdType[]) {
  return request<boolean>({
    url: `/gateway/provider/${providerIds.join(',')}`,
    method: 'delete'
  });
}
