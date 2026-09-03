import { request } from '@/service/request';

/** 版本信息（「关于」弹窗展示；登录即可） */
export function fetchGetVersion() {
  return request<Api.System.UpgradeVersionInfo>({
    url: '/system/upgrade/version',
    method: 'get'
  });
}

/** 检查更新（拉发布服务器 manifest 与当前版本比对；登录即可） */
export function fetchCheckUpdate() {
  return request<Api.System.UpgradeCheckResult>({
    url: '/system/upgrade/check',
    method: 'get'
  });
}

/** 触发在线升级（转发 updater 执行；进度轮询 fetchUpgradeStatus；需 system:setting:upgrade 权限） */
export function fetchStartUpgrade() {
  return request<Api.System.UpgradeStartResult>({
    url: '/system/upgrade/start',
    method: 'post'
  });
}

/** 升级状态机查询（升级中 3-5s 轮询；dev 环境无 updater 返回 unreachable） */
export function fetchUpgradeStatus() {
  return request<Api.System.UpgradeStateInfo>({
    url: '/system/upgrade/status',
    method: 'get'
  });
}
