# 企微能力内网部署形态（公网单端点回调）定稿

- 日期：2026-09-02
- 状态：方案定稿，部署待实施（仅生产；测试环境不管）
- 关联：[[wecom-contact-sync]]（通讯录同步）、sys_wecom_auth（登录）、sys_notify_send（推送）

## 需求

系统定位内网-only（公网不能访问），但要保留企微 OAuth 登录（PC 扫码 + 客户端免登）与推送能力。

## 方案定稿（用户拍板）

公网单端点白名单 + 同备案域名 split-horizon 分流，**零代码改动**，纯部署侧：

- DMZ 外层 Nginx 仅暴露 `/wecomCallback`（精确匹配）与 `/WW_verify_*`（可信域名归属验证，一次性，验证后删除），其余 `return 444`；`limit_req` 只挂在白名单 location
- 同一个备案域名：公网 DNS → DMZ Nginx（仅回调）；内网 DNS → 内网 Nginx（全量访问）
- 产物：`deploy/docker-prod/nginx/wecom-edge.conf.example`

## 关键结论（为什么必须这么做）

1. **PC 扫码回调由企微侧发起**（非用户浏览器直连，见 `sys_wecom_auth.go` 头注释）→ 内网零入口时 PC 扫码物理不可用，这是引入公网单端点的根本原因；企微客户端 WebView 免登回调方是客户端本身，内网直连即可
2. **两种登录形态共用同一个 redirect_uri**（`wecomClientFromConfig` 单字段）→ 必须同域名分流：异域名会导致免登回调把 httpOnly cookie 落在公网域，`location.replace('/')` 404，内网 SPA 拿不到登录态
3. **推送（应用消息/群机器人 webhook）与通讯录同步是纯出站**，与公网入站无关；自建应用调 cgi-bin API 需在企微管理端配「企业可信 IP」（服务器 NAT 出口 IP，errcode 60020）
4. **外层必须改写 `/proxy-default` 前缀**：内层 Nginx（docker-prod/nginx/nginx.conf）仅该前缀反代到 server:8888 并去前缀，直传 `/wecomCallback` 会掉进 SPA `try_files` 返回 index.html
5. 手机在外网走免登会回调到 DMZ → cookie 落公网域 → 404（合理：系统内网-only，手机在外本就用不了；连办公 WiFi 走内网 DNS 则正常）

## 配套运维项（部署时逐项确认）

- `config.yaml` 的 `system.trusted-proxies` 需追加：内层 Nginx 容器网段（prod-net 172.x）+ DMZ 外层 IP，否则 ClientIP 取到代理 IP，SecurityLimit/登录日志失真
- 证书：同域名一张证书，DMZ 与内网 Nginx 两端共用
- 企微管理端：可信域名（验证时走 DMZ 的 `/WW_verify_*`，验证后删段收窄入口）+ 企业可信 IP；`wecomCallbackUrl` 填 `https://<域名>/wecomCallback`
- 出站放行：服务器 → `qyapi.weixin.qq.com:443`；员工终端 → `open.weixin.qq.com:443`

## 明确不做

- 不做 `WecomScanEnabled` 扫码开关（扫码可用，无需求来源）
- 不给 wecomHTTPClient 加代理支持（服务器可直连出公网）
- 测试环境不部署公网回调
