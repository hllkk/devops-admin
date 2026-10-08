import { request } from '@/service/request';

/** 服务器概览统计(各类型数量/监控状态/Agent状态/环境分布) */
export function fetchGetAssetOverview() {
  return request<Api.Server.AssetOverview>({
    url: '/server/asset/overview',
    method: 'get'
  });
}

/** 重启目标机 agent(异步,公钥通道) */
export function fetchRestartAgent(assetId: CommonType.IdType) {
  return request<{ taskId: string }>({
    url: `/server/asset/${assetId}/restart-agent`,
    method: 'post'
  });
}

/** 卸载目标机 agent(异步,公钥通道;平台公钥保留可重装) */
export function fetchUninstallAgent(assetId: CommonType.IdType) {
  return request<{ taskId: string }>({
    url: `/server/asset/${assetId}/uninstall-agent`,
    method: 'post'
  });
}

/** 资产实时快照(agent 心跳指标) */
export function fetchGetAssetSnapshot(assetId: CommonType.IdType) {
  return request<Api.Server.AgentSnapshot>({
    url: `/server/asset/${assetId}/snapshot`,
    method: 'get'
  });
}

/** 触发 agent 自动安装(异步,纯公钥模式:需资产已通过 SSH 录入验证) */
export function fetchInstallAgent(assetId: CommonType.IdType) {
  return request<{ taskId: string }>({
    url: `/server/asset/${assetId}/install-agent`,
    method: 'post'
  });
}

/** 轮询 agent 安装任务状态 */
export function fetchGetInstallStatus(taskId: string) {
  return request<Api.Server.AgentInstallStatus>({
    url: `/server/asset/install-status/${taskId}`,
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
