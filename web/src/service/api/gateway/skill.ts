import { request } from '@/service/request';

// ── Skill 技能包(AI 市场 P2 收尾) ──

/** 获取用户侧可见 Skill 列表(active+published+按发布可见性过滤;模型广场 Skill 分区用) */
export function fetchGetActiveSkills() {
  return request<Api.Gateway.AvailableSkill[]>({
    url: '/gateway/skill/active',
    method: 'get'
  });
}

/** 获取可授权 Skill 列表(管理端,全部启用;Key 授权下拉用) */
export function fetchGetAvailableSkills() {
  return request<Api.Gateway.AvailableSkill[]>({
    url: '/gateway/skill/available',
    method: 'get'
  });
}

/** 分页获取 Skill 列表(支持 name/category/isActive/isPublished 筛选) */
export function fetchGetSkillList(params?: Api.Gateway.SkillSearchParams) {
  return request<Api.Gateway.SkillList>({
    url: '/gateway/skill/list',
    method: 'get',
    params
  });
}

/** 获取 Skill 详情 */
export function fetchGetSkill(skillId: CommonType.IdType) {
  return request<Api.Gateway.Skill>({
    url: `/gateway/skill/${skillId}`,
    method: 'get'
  });
}

/** 注册 Skill(仅元数据,zip 包另走上传) */
export function fetchCreateSkill(data: Api.Gateway.SkillOperateParams) {
  return request<Api.Gateway.Skill>({
    url: '/gateway/skill',
    method: 'post',
    data
  });
}

/** 修改 Skill 元数据(发布配置走 publish;停用会回收主 Key 授权) */
export function fetchUpdateSkill(data: Api.Gateway.SkillOperateParams) {
  return request<Api.Gateway.Skill>({
    url: '/gateway/skill',
    method: 'put',
    data
  });
}

/** 批量删除 Skill(回收主 Key 授权+删 zip 包;使用日志保留) */
export function fetchBatchDeleteSkills(skillIds: CommonType.IdType[]) {
  return request<boolean>({
    url: `/gateway/skill/${skillIds.join(',')}`,
    method: 'delete'
  });
}

/** 获取发布设置(含 selected/user 模式的可见部门/用户回显) */
export function fetchGetSkillPublish(skillId: CommonType.IdType) {
  return request<Api.Gateway.SkillPublishView>({
    url: `/gateway/skill/publish/${skillId}`,
    method: 'get'
  });
}

/** 更新发布设置(发布免审批自动授权按可见档主 Key,双向对齐;须先上传 zip 包) */
export function fetchPublishSkill(data: Api.Gateway.SkillPublishParams) {
  return request<boolean>({
    url: '/gateway/skill/publish',
    method: 'put',
    data
  });
}

/** 上传/替换 Skill zip 包(multipart,≤100MB;大文件放宽超时) */
export function fetchUploadSkillPackage(skillId: CommonType.IdType, file: File) {
  const formData = new FormData();
  formData.append('file', file);
  return request<Api.Gateway.Skill>({
    url: `/gateway/skill/${skillId}/package`,
    method: 'post',
    data: formData,
    timeout: 0
  });
}

/** 下载 Skill zip 包(登录态;需审批 Skill 须已授权;返回 blob 由调用方触发保存) */
export async function fetchDownloadSkill(skillId: CommonType.IdType): Promise<Blob> {
  const { data } = await request<Blob, 'blob'>({
    url: `/gateway/skill/download/${skillId}`,
    method: 'get',
    responseType: 'blob'
  });
  const blob = data as unknown as Blob;
  // 错误响应是 JSON(transform 原样返回 blob),按 type 识别抛出可读信息
  if (blob && blob.type.includes('application/json')) {
    const text = await blob.text();
    let msg = '下载失败';
    try {
      msg = JSON.parse(text)?.msg || msg;
    } catch {
      // 非 JSON 文本,保留默认信息
    }
    throw new Error(msg);
  }
  if (!blob) {
    throw new Error('下载失败');
  }
  return blob;
}

/** 分页获取 Skill 使用日志(管理端,回填用户名/技能名) */
export function fetchGetSkillUsageList(params?: Api.Gateway.SkillUsageSearchParams) {
  return request<Api.Gateway.SkillUsageList>({
    url: '/gateway/skill/usage/list',
    method: 'get',
    params
  });
}
