import { request } from '@/service/request';

/** 获取通知公告列表 */
export function fetchGetNoticeList(params?: Api.System.NoticeSearchParams) {
  return request<Api.System.NoticeList>({
    url: '/system/notice/list',
    method: 'get',
    params
  });
}

/** 新增通知公告 */
export function fetchCreateNotice(data: Api.System.NoticeOperateParams) {
  return request<boolean>({
    url: '/system/notice',
    method: 'post',
    data
  });
}

/** 修改通知公告 */
export function fetchUpdateNotice(data: Api.System.NoticeOperateParams) {
  return request<boolean>({
    url: '/system/notice',
    method: 'put',
    data
  });
}

/** 批量删除通知公告 */
export function fetchBatchDeleteNotice(noticeIds: CommonType.IdType[]) {
  return request<boolean>({
    url: `/system/notice/${noticeIds.join(',')}`,
    method: 'delete'
  });
}

/** 获取当前用户通知列表(未读/历史) */
export function fetchGetUnreadNotice(params: { pageNum: number; pageSize: number; onlyUnread?: boolean }) {
  return request<Api.System.NoticeList>({
    url: '/system/notice/unread',
    method: 'get',
    params
  });
}

/** 标记通知已读(noticeIds 为空=全部已读) */
export function fetchMarkNoticeRead(noticeIds: CommonType.IdType[] = []) {
  return request<boolean>({
    url: '/system/notice/read',
    method: 'put',
    data: { noticeIds }
  });
}
