import { request } from '@/service/request';

/**
 * 配额 API（第1期 mock）
 * ⚠️ 后端 /disk/quota 就绪后把 USE_MOCK 改 false（与 file.ts 的 USE_MOCK 同步切换）
 */
const USE_MOCK = true;

/** 获取用户配额信息 */
export function fetchGetQuota() {
  if (USE_MOCK) {
    // mock：已用 ~11.5GB / 上限 ~93GB（约 12%）
    const data: Api.Disk.QuotaInfo = {
      usedSpace: 12_345_678_900,
      quota: 100_000_000_000,
      unlimited: false,
      quotaSource: 'personal'
    };
    return new Promise<{ data: Api.Disk.QuotaInfo; error: null }>(resolve => {
      setTimeout(() => resolve({ data, error: null }), 200);
    });
  }

  return request<Api.Disk.QuotaInfo>({
    url: '/disk/quota',
    method: 'get'
  });
}
