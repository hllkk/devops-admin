import { request } from '@/service/request';

/** 获取定时任务列表 */
export function fetchGetTimedTaskList(params?: Api.System.SysTimedTaskSearchParams) {
  return request<Api.System.SysTimedTaskList>({
    url: '/timedTask/getTimedTaskList',
    method: 'get',
    params
  });
}

/** 创建定时任务 */
export function fetchCreateTimedTask(data: any) {
  return request<boolean>({
    url: '/timedTask/createTimedTask',
    method: 'post',
    data
  });
}

/** 更新定时任务 */
export function fetchUpdateTimedTask(data: any) {
  return request<boolean>({
    url: '/timedTask/updateTimedTask',
    method: 'put',
    data
  });
}

/** 删除定时任务 */
export function fetchDeleteTimedTask(data: { ID: number }) {
  return request<boolean>({
    url: '/timedTask/deleteTimedTask',
    method: 'delete',
    data
  });
}

/** 启用/停用定时任务 */
export function fetchToggleTimedTask(data: { ID: number; enabled: boolean }) {
  return request<boolean>({
    url: '/timedTask/toggleTimedTask',
    method: 'post',
    data
  });
}

/** 手动触发定时任务 */
export function fetchTriggerTimedTask(data: { ID: number }) {
  return request<boolean>({
    url: '/timedTask/triggerTimedTask',
    method: 'post',
    data
  });
}

/** 获取执行日志列表 */
export function fetchGetTimedTaskLogList(params?: Api.System.SysTimedTaskLogSearchParams) {
  return request<Api.System.SysTimedTaskLogList>({
    url: '/timedTask/getTimedTaskLogList',
    method: 'get',
    params
  });
}

/** 获取已注册方法列表 */
export function fetchGetRegisteredMethods() {
  return request<Api.System.RegisteredMethodList>({
    url: '/timedTask/getRegisteredMethods',
    method: 'get'
  });
}
