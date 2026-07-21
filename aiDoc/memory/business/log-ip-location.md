# 登录/操作日志 IP 地点解析（ip2region 离线库）

> 2026-07-21｜承接 [[system-log-models]] 预留的 LoginLocation/OperLocation 字段

## 需求
登录日志 `sys_login_log.login_location`、操作日志 `sys_oper_log.oper_location` 字段此前已建表/建列但**从未填值**（空串）。需基于请求 IP 反查地理位置并落库,让前端日志列表/详情展示「登录地点」「操作地点」。IP 字段（ipaddr/oper_ip）已用 `c.ClientIP()` 填充,仅缺 IP→地点这一步。

## 方案选型
- 离线 IP 库 **ip2region**(方案 A:本地查询、μs 级、无 QPS 限制、IP 不外泄、零成本)优于在线 API 与接入层方案
- 用官方 `binding/golang/service` 包 + **BufferCache**(全内存、单实例 searcher 天生并发安全、无文件 IO)
- **未引入 Redis 缓存**:μs 级查询 + 操作日志已异步,缓存收益不抵复杂度(项目「不过度设计」原则)

## 实现
- 依赖 `github.com/lionsoul2014/ip2region/binding/golang`(go.mod 直接依赖);数据文件 `server/resource/ip2region.xdb`(v4,11MB,随仓库分发,Apache-2.0)
- `utils/ip_location.go`:`LoadIp2Region(dbPath)`(全内存加载)+ `ParseIPLocation(ip)`
  - 私有/回环/链路本地/未指定 IP → `内网IP`(不查库,对齐「内网IP不留空」诉求)
  - 非法 IP / 公网查询不到 / 服务未就绪 → `未知`(不留空);空串 → `""`
  - 公网 → 原生 `国家|区域|城市|ISP|国家码`(如 `中国|北京|北京市|电信|CN`)
- `config/system.go` 加 `ip2region-db-path`(默认 `resource/ip2region.xdb`),config.yaml + config.docker.yaml 同步
- `initialize/other.go` `OtherInit` 启动加载,**失败降级不阻断**;注意此处 OPS_LOG 尚未初始化(main.go 中 Zap 在 OtherInit 之后),改用标准 `log` 落 stderr
- 登录日志:`service/system/sys_login_log.go` `CreateLoginLog` 统一补 `LoginLocation`(CreateLoginLog 是唯一写入口,5 个登录分支全覆盖,**调用方零改动**)
- 操作日志:`service/system/sys_oper_log.go` **异步队列消费处**补 `OperLocation`(对请求零阻塞)
- **仅启用 IPv4**(v6Config=nil),IPv6 地址查询返回 `未知`

## 验证
- `go build ./...` 通过;`go mod tidy` ip2region 转直接依赖
- `go test` 实测:`220.181.38.148`→`中国|北京|北京市|电信|CN`、`114.114.114.114`→`中国|江苏省|南京市|0|CN`、`8.8.8.8`→`United States|California|0|Google LLC|US`、`192.168.1.100`/`10.0.0.1`/`127.0.0.1`/`::1`/`169.254.1.1`→`内网IP`、`abc`→`未知`、空串→`""`
- 前端 `loginLocation`/`operLocation` 字段 + 中英 i18n + `oper-log-view-drawer.vue` 早已就绪,后端填值即生效,**前端零改动**

## 部署注意
- xdb 文件 `server/resource/ip2region.xdb` 须随部署分发;Docker 镜像 Dockerfile 的 COPY 须涵盖 `resource/`(否则容器缺文件降级「未知」)——本次未找到项目 Dockerfile,待确认部署方式

## 待办 / 可选(当前不做)
- IPv6 支持(追加 `ip2region_v6.xdb`,v6Config 非 nil)
- 定期更新 xdb(可加定时任务替换文件,应对 IP 段时效偏差)
- 大流量场景加 Redis 缓存 `ip2region:{ip}`(企业内网 IP 集中,命中率高)

## 关联
- [[system-log-models]]、[[system-oper-log-middleware]]、[[system-log-read-api]]
