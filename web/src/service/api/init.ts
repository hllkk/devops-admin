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

/**
 * 测试数据库连接（仅 ping，不建库不落盘）
 *
 * 对应后端 POST /init/conn-test。
 */
export function fetchPingDB(data: Api.Init.PingDBForm) {
  return request<string>({ url: '/init/conn-test', method: 'post', data });
}

/**
 * 测试 Redis 连接（不落盘）
 *
 * 对应后端 POST /init/ping-redis。
 */
export function fetchPingRedis(data: Api.Init.PingRedisForm) {
  return request<string>({ url: '/init/ping-redis', method: 'post', data });
}
