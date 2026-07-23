# 第三方应用绑定 + 社交登录 后端

> 2026-07-23。前端「个人中心 SocialCard 绑定/解绑 + 登录页社交按钮 + /social-callback 落地页」早已齐备,后端 OAuth 链路 0 实现(前端 4 接口全 404)。本次按 `docs/superpowers/specs/2026-07-23-social-binding-design.md`(已批准)补后端,微信开放平台/Gitee/GitHub 三平台。前端零改动(仅顺带修 social-card 绑定时间展示)。

## 范围

- 绑定:已登录用户在个人中心关联第三方账号
- 登录:已绑定过的第三方账号可直接登录(未绑定拒绝,不自动建用户)
- 不含企业微信扫码登录(后续再做)

## 关键决策

- OAuth 回调方案 B(前端直接回调):零前端改动,后端 4 endpoint
- 新 openid 策略:拒绝未注册(社交登录仅对已绑定 openid 生效)
- Token 存储:AES-256-GCM 加密落库,`AccessToken`/`RefreshToken` `json:"-"` 不返回前端;密钥从 `config.yaml` 的 `social.token-key` 读(32 字节 hex,`openssl rand -hex 32` 生成)
- state:base64 JSON,用 **URLEncoding 而非 StdEncoding**——StdEncoding 产生的 `+` 经 OAuth provider 走 URL query 回传会被浏览器按 query 规则解码成空格,破坏 state;URLEncoding 用 `-_` 规避,前端 `social-callback/index.vue` 的 `atob` 需兼容 `-_`

## 后端改动(7 新 + 7 改)

新建:
- `utils/crypto/aes.go`:AES-256-GCM 加解密(64 字符 hex 密钥)
- `config/social.go` + `config/config.go` Server 加 `Social.TokenKey` 字段
- `model/system/sys_social.go`:SysSocial(OPS_AUDIT_MODEL + ID 雪花 + token `json:"-"`)
- `model/system/request/sys_social.go`:SocialLoginForm + SocialState
- `service/system/sys_social.go`:GetAuthURL/Callback/List/Unbind + 三平台 token/userinfo + AES + state 编解码
- `api/v1/system/sys_social.go`:4 handler + Swagger;登录成功复用 `BaseApi.TokenNext` 签 JWT
- `router/system/sys_social.go`:InitSocialRouter(双 group)

修改:
- 三处 enter.go(service/api/router 加 SocialService/SocialApi/SocialRouter + var 别名)
- `initialize/router.go`:InitSocialRouter 调用
- `initialize/gorm_biz.go`:`bizModel()` 注册 SysSocial(纯业务表落点,非 gorm.go RegisterTables)
- `go.mod`/`go.sum`:oauth2 转直接依赖

## 契约要点(4 接口)

| 接口 | 方法/URL | group |
|---|---|---|
| GetAuthURL | GET /auth/binding/:source?domain=host → string | Public |
| Callback | POST /auth/social/callback(SocialLoginForm) | Public |
| List | GET /system/social/list → Social[] | Private |
| Unbind | DELETE /auth/unlock/:id | Private |

- `sys_auth_config` 表配 provider(wechat/gitee/github 的 enabled+clientId/clientSecret/callbackUrl,已有系统设置 UI)
- callbackUrl 是前端 URL `https://host/social-callback?source=xxx`,三方回传保留 query,前端自然拿 code/state/source
- 登录/绑定分流:后端按 state 的 `intent=bind+userId`(且与 JWT cookie 一致)权威判断,不依赖前端 isLogin
- 解绑计数:本地密码可登录(IsPasswordExpired 未过期)+ 其它社交绑定,至少保留一种
- GetAuthURL/Callback 在 PublicGroup,安静解析 x-token cookie(不用 GetClaims,避免未登录用户刷 error 日志)

## 建表落点

sys_social 纯业务表(无种子、不走 /initdb)→ `initialize/gorm_biz.go` 的 `bizModel()`,不进 `gorm.go` RegisterTables(那是 JwtBlacklist/SysError 内部散表落点)。详见 backend-layer-rules「表注册与新增 model 的建表维护点」。

## service 层 GORM 查询

bind/login/Unbind 多次查询各用独立 `global.OPS_DB.WithContext(ctx)` tx,不复用被 finisher 污染的 db 变量(防 WHERE 累积/ErrRecordNotFound 残留短路,见 backend-layer-rules GORM 链式查询规则)。

## 运行时配置(用户需做)

1. config.yaml 加 `social: token-key: <openssl rand -hex 32 生成 64 字符 hex>`;未配则绑定报"社交令牌加密密钥未配置"
2. 系统设置→安全配置启 wechat/gitee/github 并填 clientId/clientSecret/callbackUrl
3. 重启服务 AutoMigrate 建 sys_social 表

## 验证

`go build ./...` + `gofmt -l` + `pnpm typecheck` 全通过。

## 已知限制

- 未登录社交登录前端 `authStore.login` 当前丢 social 字段(soybean 示例遗留),登录流程前端需后续修;绑定不受影响。后端已实现 POST /auth/social/callback 登录返回 LoginResponse
- token 过期刷新当前只存不刷;未来调三方 API 可加 refresh 逻辑

## 关联

- 认证链路单 x-token cookie 见 [[httponly-cookie-auth]](蓝图未全量落地,当前简化实现);登录成功复用 BaseApi.TokenNext
- sys_auth_config provider 配置见 [[setting-page-restructure]] 系统设置
- 密码过期判定 IsPasswordExpired 见 [[setting-page-restructure]] 安全配置
- 前端 social-card 顺带修:绑定时间裸 ISO 串改 NaiveUI `<NTime type="datetime">`(非手写 dayjs,合规 frontend-utils 禁止手写日期格式化)
