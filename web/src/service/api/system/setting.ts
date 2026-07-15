import { request } from '@/service/request';

/** 获取公开系统设置（登录页使用，无需登录） */
export function fetchGetPublicSetting() {
  return request<Api.System.PublicSetting>({
    url: '/system/setting/public',
    method: 'get'
  });
}

/** 获取系统设置（管理员） */
export function fetchGetSetting() {
  return request<Api.System.Setting>({
    url: '/system/setting',
    method: 'get'
  });
}

/** 更新系统设置（管理员） */
export function fetchUpdateSetting(data: Api.System.Setting) {
  return request<boolean>({
    url: '/system/setting',
    method: 'put',
    data
  });
}
