import { request } from '@/service/request';

/** 效能报告分页列表(不带内容大字段) */
export function fetchGetReportList(params?: Api.Gateway.ReportSearchParams) {
  return request<Api.Common.PaginatingQueryRecord<Api.Gateway.EfficiencyReportView>>({
    url: '/gateway/report/list',
    method: 'get',
    params
  });
}

/** 效能报告详情(结构化内容+Markdown) */
export function fetchGetReport(id: CommonType.IdType) {
  return request<Api.Gateway.EfficiencyReportView>({
    url: `/gateway/report/${id}`,
    method: 'get'
  });
}

/** 手动生成效能报告(weekly/monthly 取上一完整周期,custom 须带起止) */
export function fetchGenerateReport(data: Api.Gateway.ReportGenerateParams) {
  return request<Api.Gateway.EfficiencyReportView>({
    url: '/gateway/report/generate',
    method: 'post',
    data
  });
}
