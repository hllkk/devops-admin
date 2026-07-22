import { request } from '@/service/request';

/** 获取错误日志列表 */
export function fetchGetErrorLogList(params?: Api.Log.ErrorLogSearchParams) {
  return request<Api.Log.ErrorLogList>({
    url: '/log/sysError/getSysErrorList',
    method: 'get',
    params
  });
}

/** 删除错误日志 */
export function fetchDeleteErrorLog(id: string) {
  return request<boolean>({
    url: '/log/sysError/deleteSysError',
    method: 'delete',
    params: { ID: id }
  });
}

/** 批量删除错误日志 */
export function fetchBatchDeleteErrorLog(ids: string[]) {
  return request<boolean>({
    url: '/log/sysError/deleteSysErrorByIds',
    method: 'post',
    data: ids
  });
}

/** 查询错误日志详情 */
export function fetchGetErrorLogDetail(id: string) {
  return request<Api.Log.ErrorLog>({
    url: '/log/sysError/findSysError',
    method: 'get',
    params: { ID: id }
  });
}

/** 触发AI处理 */
export function fetchGetErrorLogSolution(id: string) {
  return request<{ msg: string }>({
    url: '/log/sysError/getSysErrorSolution',
    method: 'get',
    params: { id }
  });
}
