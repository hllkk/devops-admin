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

/** 发送测试邮件（使用当前表单值，无需先保存） */
export function fetchTestEmail(data: {
  emailHost: string;
  emailPort: number;
  emailUsername: string;
  emailPassword: string;
  emailFromAddr: string;
  emailFromName: string;
  emailSSLMode: string;
  testTo: string;
}) {
  return request<boolean>({
    url: '/system/setting/notify/test-email',
    method: 'post',
    data
  });
}

/** 发送企微应用消息测试（凭证取已保存的认证配置企微段，目标用户须已绑定企微） */
export function fetchTestWecomApp(data: { testUserId: CommonType.IdType; redirectBase: string }) {
  return request<boolean>({
    url: '/system/setting/notify/test-wecom-app',
    method: 'post',
    data
  });
}

/** 发送企微群机器人测试（使用当前表单 webhook，无需先保存） */
export function fetchTestWecomBot(data: { webhookUrl: string }) {
  return request<boolean>({
    url: '/system/setting/notify/test-wecom-bot',
    method: 'post',
    data
  });
}
