import { request } from '@/service/request';

// ── 模型 Model ──

/** 获取用户侧可见模型列表(active+published+按发布可见性过滤;模型广场用,超管看全部) */
export function fetchGetActiveModels() {
  return request<Api.Gateway.ActiveModel[]>({
    url: '/gateway/model/active',
    method: 'get'
  });
}

/** 分页获取模型列表(支持 name/modelKey/category/isActive/isPublished 筛选) */
export function fetchGetModelList(params?: Api.Gateway.ModelSearchParams) {
  return request<Api.Gateway.ModelList>({
    url: '/gateway/model/list',
    method: 'get',
    params
  });
}

/** 获取模型详情(含部署列表) */
export function fetchGetModel(modelId: CommonType.IdType) {
  return request<Api.Gateway.Model & { deployments: Api.Gateway.Deployment[] }>({
    url: `/gateway/model/${modelId}`,
    method: 'get'
  });
}

/** 新增模型 */
export function fetchCreateModel(data: Api.Gateway.ModelOperateParams) {
  return request<Api.Gateway.Model>({
    url: '/gateway/model',
    method: 'post',
    data
  });
}

/** 修改模型(改模型 ID/改类触发关联部署在 LiteLLM 侧级联重建) */
export function fetchUpdateModel(data: Api.Gateway.ModelOperateParams) {
  return request<boolean>({
    url: '/gateway/model',
    method: 'put',
    data
  });
}

/** 批量删除模型(软删三连：部署先禁用→部署停用→模型软删) */
export function fetchBatchDeleteModel(modelIds: CommonType.IdType[]) {
  return request<boolean>({
    url: `/gateway/model/${modelIds.join(',')}`,
    method: 'delete'
  });
}

/** 获取模型发布设置(含 selected 模式的可见部门) */
export function fetchGetModelPublish(modelId: CommonType.IdType) {
  return request<Api.Gateway.ModelPublishView>({
    url: `/gateway/model/publish/${modelId}`,
    method: 'get'
  });
}

/** 更新发布设置(发布公开模型会自动授权到所有启用中的主 Key) */
export function fetchPublishModel(data: Api.Gateway.ModelPublishParams) {
  return request<boolean>({
    url: '/gateway/model/publish',
    method: 'put',
    data
  });
}

// ── 部署 Deployment ──

/** 分页获取部署列表(按 modelId/credentialId 过滤) */
export function fetchGetDeploymentList(params?: Api.Gateway.DeploymentSearchParams) {
  return request<Api.Gateway.DeploymentList>({
    url: '/gateway/model/deployment/list',
    method: 'get',
    params
  });
}

/** 新增部署(litellmParams 必含 model 键) */
export function fetchCreateDeployment(data: Api.Gateway.DeploymentOperateParams) {
  return request<Api.Gateway.Deployment>({
    url: '/gateway/model/deployment',
    method: 'post',
    data
  });
}

/** 修改部署(敏感键掩码回传=保留旧明文) */
export function fetchUpdateDeployment(data: Api.Gateway.DeploymentOperateParams) {
  return request<boolean>({
    url: '/gateway/model/deployment',
    method: 'put',
    data
  });
}

/** 批量删除部署 */
export function fetchBatchDeleteDeployment(deploymentIds: CommonType.IdType[]) {
  return request<boolean>({
    url: `/gateway/model/deployment/${deploymentIds.join(',')}`,
    method: 'delete'
  });
}

/** 部署连通性测试(经 LiteLLM 数据面,返回成功/延迟/错误分类;技术详情已脱敏) */
export function fetchTestDeployment(data: Api.Gateway.DeploymentTestParams) {
  return request<Api.Gateway.DeploymentTestResult>({
    url: '/gateway/model/deployment/test',
    method: 'post',
    data,
    timeout: 30000
  });
}

// ── 路由策略 RouterSettings ──

/** 获取全局路由策略(单例,同步自 LiteLLM /router/settings) */
export function fetchGetRouterSettings() {
  return request<Api.Gateway.RouterSettings>({
    url: '/gateway/router/settings',
    method: 'get'
  });
}

/** 更新全局路由策略(写库 + 同步 LiteLLM /router/settings 热更新) */
export function fetchUpdateRouterSettings(data: Api.Gateway.RouterSettingsParams) {
  return request<Api.Gateway.RouterSettings>({
    url: '/gateway/router/settings',
    method: 'put',
    data
  });
}
