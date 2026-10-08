import { request } from '@/service/request';

/** 服务器概览统计(各类型数量/监控状态/Agent状态/环境分布) */
export function fetchGetAssetOverview() {
  return request<Api.Server.AssetOverview>({
    url: '/server/asset/overview',
    method: 'get'
  });
}

/** 分页获取资产列表 */
export function fetchGetAssetList(params?: Api.Server.AssetSearchParams) {
  return request<Api.Common.PaginatingQueryRecord<Api.Server.Asset>>({
    url: '/server/asset/list',
    method: 'get',
    params
  });
}

/** 获取资产详情 */
export function fetchGetAsset(assetId: CommonType.IdType) {
  return request<Api.Server.Asset>({
    url: `/server/asset/${assetId}`,
    method: 'get'
  });
}

/** 新增资产 */
export function fetchCreateAsset(data: Api.Server.AssetOperateParams) {
  return request<Api.Server.Asset>({
    url: '/server/asset',
    method: 'post',
    data
  });
}

/** 修改资产 */
export function fetchUpdateAsset(data: Api.Server.AssetOperateParams) {
  return request<boolean>({
    url: '/server/asset',
    method: 'put',
    data
  });
}

/** 批量删除资产 */
export function fetchBatchDeleteAssets(assetIds: CommonType.IdType[]) {
  return request<boolean>({
    url: `/server/asset/${assetIds.join(',')}`,
    method: 'delete'
  });
}
