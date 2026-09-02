import { request } from '@/service/request';

/** 同步企业微信通讯录(异步启动:全量首同步含逐用户新建,耗时长,后端 goroutine 执行,立即返回 started) */
export function fetchSyncWecomStructure() {
  return request<Api.System.WecomSyncStatus>({
    url: '/system/wecom/syncStructure',
    method: 'post'
  });
}

/** 查询企微通讯录同步状态(进度/最近结果/错误),供异步轮询 */
export function fetchWecomSyncStatus() {
  return request<Api.System.WecomSyncStatus>({
    url: '/system/wecom/syncStatus',
    method: 'get'
  });
}
