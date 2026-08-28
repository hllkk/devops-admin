import { request } from '@/service/request';

/** 提交资源申请(用户侧,暂仅模型;需审批+对申请人可见才可申请) */
export function fetchSubmitApplication(data: Api.Gateway.ApplicationCreateParams) {
  return request<Api.Gateway.ApplicationItem>({
    url: '/gateway/application/apply',
    method: 'post',
    data
  });
}

/** 我的申请列表(用户侧,强制本人) */
export function fetchGetMyApplications(params?: Api.Gateway.ApplicationSearchParams) {
  return request<Api.Common.PaginatingQueryRecord<Api.Gateway.ApplicationItem>>({
    url: '/gateway/application/my',
    method: 'get',
    params
  });
}

/** 分页获取申请列表(管理端审批列表) */
export function fetchGetApplicationList(params?: Api.Gateway.ApplicationSearchParams) {
  return request<Api.Common.PaginatingQueryRecord<Api.Gateway.ApplicationItem>>({
    url: '/gateway/application/list',
    method: 'get',
    params
  });
}

/** 审批通过(授权模型到申请人个人主 Key) */
export function fetchApproveApplication(data: { applicationId: CommonType.IdType; reviewNotes?: string }) {
  return request<Api.Gateway.ApplicationReviewResult>({
    url: '/gateway/application/approve',
    method: 'put',
    data
  });
}

/** 审批驳回(仅留痕,无授权动作) */
export function fetchRejectApplication(data: { applicationId: CommonType.IdType; reviewNotes?: string }) {
  return request<Api.Gateway.ApplicationReviewResult>({
    url: '/gateway/application/reject',
    method: 'put',
    data
  });
}

/** 批量审批(approve=true 批量通过,false 批量驳回;逐条独立事务,单条失败不中断) */
export function fetchBatchReviewApplications(
  data: { applicationIds: CommonType.IdType[]; reviewNotes?: string },
  approve: boolean
) {
  return request<Api.Gateway.BatchReviewResult>({
    url: approve ? '/gateway/application/batch-approve' : '/gateway/application/batch-reject',
    method: 'put',
    data,
    timeout: 0
  });
}
