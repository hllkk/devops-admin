# 企业微信扫码登录

> 2026-07-27。对照 `/home/remote/devops-admin` 的企微实现,在当前项目接入企业微信扫码登录(PC 内嵌二维码 + 企微客户端 WebView 免登)。[[social-binding]] 已铺好企微地基(`sys_auth_config` 字段/前端按钮/`WW_verify` 路由全就位),本次补全后端 OAuth 链路与前端扫码组件 + 自动建号。

## 范围

- PC 扫码:登录页内嵌二维码(`qrcode` 库渲染 `oauth2/authorize` URL),`sceneId` + Redis 状态机 + `setTimeout` 链式轮询
- 企微客户端 WebView 免登:`wxwork` UA 检测 → 静默跳授权 → 回调同源设 localStorage + 跳 SPA
- 自动建号:首次扫码按 `WecomDefaultRoleId` 创建 `sys_user` + `sys_social(wecom)` 关联
- 不含组织架构同步(远程的 `/wecom/syncStructure`,独立功能后续再做)

## 关键决策

- 授权方式:`oauth2/authorize` + `scope=snsapi_privateinfo`(能拿 `user_ticket`→手机/邮箱;`qrConnect` iframe 拿不到,已弃用)
- 身份存储**复用 `sys_social`**(`source=wecom`,`open_id`=企微 userid),加 `Mobile` 字段;**不另起** `sys_wecom_users` 表
- token 下发**复用单 `x-token` cookie**(`utils.SetToken`/`TokenNext`),不引入远程的双 cookie
- access_token 缓存:`OPS_CACHE`(Redis 优先降级内存) + `OPS_Concurrency_Control`(singleflight) 防击穿,提前 120s 过期
- 建号策略:**自动建号 + 默认角色**(`WecomDefaultRoleId`,`sys_auth_config` 配置项;为 0 则建号失败)。用户名 `wecom_<userid>`,密码随机不可用,`PasswordUpdatedAt=now` 防过期判定
- **PC 回调来自企微服务器**(非用户浏览器):token 暂存 Redis,前端轮询命中 `confirmed` 时下发 cookie(拿用户真实 IP/UA);WebView 回调即用户浏览器,直接下发
- 与 [[social-binding]] 的"拒绝自动建号"形成对照:企微是企业内部场景,员工扫码即应可用

## 落点

- 后端:`server/utils/wecom.go`(`WecomClient` + access_token 缓存)、`server/api/v1/system/sys_wecom_auth.go`(`QrCodeView`/`QrCodeStatusView`/`WecomCallbackView`/`WecomWebviewLoginView` + 自动建号)、`server/router/system/sys_wecom.go`
- 前端:`web/src/views/_builtin/login/modules/wecom-login.vue`(qrcode canvas + 轮询 + WebView 分支)、`service/api/auth.ts`(3 个 fetch)、`store/modules/auth/index.ts`(`wecomLogin`)、`utils/agent.ts`(`isWecomWebview`)
- model:`sys_social` 加 `Mobile`、`sys_auth_config` 加 `WecomDefaultRoleId`(`int64`,`json:",string"` 对齐前端 `IdType`)
- 配置:系统设置→认证→企业微信 加"扫码建号默认角色"下拉(`auth-setting.vue`,复用 `fetchGetRoleSelect`)
- 用户中心 `social-card.vue` 补 wecom:`autoBind` 标记,未绑定时显示"扫码自动关联"提示(企微不能主动绑定到已有账号)

## 待办/注意

- 真实联调需企微企业 + 自建应用(CorpId/AgentId/Secret/回调地址/可信域名 `WW_verify_*.txt`),配置后才能端到端跑通
- `AutoMigrate` 启动自动加列:`sys_social.mobile`、`sys_auth_config.wecom_default_role_id`
- typecheck 有 2 个**预先存在**错误(`route` i18n 缺 `test` key、`vite.config.ts` 的 `allowedHosts` 类型),与企微无关
- `WecomWebviewLoginView` 返回 `{oauthUrl}` JSON 由前端 `location.replace`(非服务端 302):fetch 跟随跨域 302 到企微域名会触发 CORS
