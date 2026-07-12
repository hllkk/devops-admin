import { request } from '../request';

/**
 * 检查数据库是否需要初始化
 *
 * 对应后端 POST /init/checkdb，返回 { needInit }
 */
export function fetchCheckDB() {
  return request<Api.Init.CheckDBResult>({
    url: '/init/checkdb',
    method: 'post'
  });
}

/**
 * 初始化数据库
 *
 * 对应后端 POST /init/initdb；耗时操作，调用方需自行控制 loading。
 */
export function fetchInitDB(data: Api.Init.InitDBForm) {
  return request<string>({
    url: '/init/initdb',
    method: 'post',
    data
  });
}
