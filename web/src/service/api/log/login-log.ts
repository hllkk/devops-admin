import { request } from '@/service/request';

/** 获取系统访问记录列表 */
export function fetchGetLoginLogList(params?: Api.Log.LoginLogSearchParams) {
  return request<Api.Log.LoginLogList>({
    url: '/log/loginlog/list',
    method: 'get',
    params
  });
}

/** 批量删除系统访问记录 */
export function fetchBatchDeleteLoginLog(infoIds: CommonType.IdType[]) {
  return request<boolean>({
    url: `/log/loginlog/${infoIds.join(',')}`,
    method: 'delete'
  });
}

/** 解锁系统访问记录 */
export function fetchUnlockLoginLog(username: string) {
  return request<boolean>({
    url: `/log/loginlog/unlock/${username}`,
    method: 'get'
  });
}

/** 清空系统访问记录 */
export function fetchCleanLoginLog() {
  return request<boolean>({
    url: '/log/loginlog/clean',
    method: 'delete'
  });
}
