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

/** 获取供应商套餐余量明细(坐席/共享包,厂商侧快照) */
export function fetchGetProviderBalances(providerId: CommonType.IdType) {
  return request<Api.Gateway.ProviderBalanceDetail>({
    url: `/gateway/provider/${providerId}/balance`,
    method: 'get'
  });
}

/** 读余量采集配置(AK/SK 掩码回显) */
export function fetchGetBalanceConfig(providerId: CommonType.IdType) {
  return request<Api.Gateway.BalanceSyncConfig>({
    url: `/gateway/provider/${providerId}/balance-config`,
    method: 'get'
  });
}

/** 保存余量采集配置(掩码占位保留旧明文) */
export function fetchSaveBalanceConfig(providerId: CommonType.IdType, data: Api.Gateway.BalanceSyncConfig) {
  return request<boolean>({
    url: `/gateway/provider/${providerId}/balance-config`,
    method: 'put',
    data
  });
}

/** 手动同步供应商套餐余量 */
export function fetchSyncProviderBalance(providerId: CommonType.IdType) {
  return request<Api.Gateway.ProviderBalanceSummary>({
    url: `/gateway/provider/${providerId}/balance-sync`,
    method: 'post',
    timeout: 0 // 厂商 OpenAPI 外呼+翻页,不吃默认 10s 超时
  });
}

