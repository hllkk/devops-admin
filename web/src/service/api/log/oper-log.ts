import { request } from '@/service/request';

/** 获取操作日志记录列表 */
export function fetchGetOperLogList(params?: Api.Log.OperLogSearchParams) {
  return request<Api.Log.OperLogList>({
    url: '/log/operlog/list',
    method: 'get',
    params
  });
}

/** 批量删除操作日志记录 */
export function fetchBatchDeleteOperLog(operIds: CommonType.IdType[]) {
  return request<boolean>({
    url: `/log/operlog/${operIds.join(',')}`,
    method: 'delete'
  });
}

/** 清理操作日志记录 */
export function fetchCleanOperLog() {
  return request<boolean>({
    url: '/log/operlog/clean',
    method: 'delete'
  });
}
