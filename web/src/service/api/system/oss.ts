import { request } from '@/service/request';

/** 批量删除文件管理 */
export function fetchBatchDeleteOss(ossIds: CommonType.IdType[]) {
  return request<boolean>({
    url: `/resource/oss/${ossIds.join(',')}`,
    method: 'delete'
  });
}
