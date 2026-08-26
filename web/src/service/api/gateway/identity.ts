import { request } from '@/service/request';

/** 获取我的 AI 身份(管理员创建制：未开通 opened=false；已开通返回主 Key 明文 + 场景 Key 列表 + 可用模型) */
export function fetchGetMyIdentity() {
  return request<Api.Gateway.MyIdentity>({
    url: '/gateway/ai-key/identity/my',
    method: 'get'
  });
}

/** 获取可授权模型列表(发布+激活，含 anthropic 变体标注) */
export function fetchGetAvailableModels() {
  return request<Api.Gateway.AvailableModel[]>({
    url: '/gateway/ai-key/identity/available-models',
    method: 'get'
  });
}
