# 第三方应用绑定 + 社交登录 后端实现设计

> 日期：2026-07-23
> 状态：已批准
> 范围：微信开放平台 / Gitee / GitHub OAuth2 绑定与登录

---

## 1. 概述

### 1.1 背景

项目前端已完成「第三方应用绑定/解绑」UI（个人中心 SocialCard）+ 登录页社交按钮 + OAuth 回调落地页（`/social-callback`），但后端 OAuth 链路 0 实现——前端调用的 4 个接口全部 404。本设计按前端契约从零补后端。

### 1.2 范围

- **绑定**：已登录用户在个人中心关联第三方账号（微信/Gitee/GitHub）
- **登录**：已绑定过的第三方账号可直接登录（未绑定的拒绝，不自动建用户）
- **不包含**：企业微信扫码登录（后续再做）

### 1.3 核心决策

| 决策项 | 选择 | 理由 |
|--------|------|------|
| OAuth 回调架构 | 方案 B：前端直接回调 | 零前端改动，后端最简（4 个 API endpoint），匹配现有前端契约 |
| 新 openid 策略 | 拒绝，要求已注册 | 安全优先；社交登录仅对已绑定过的 openid 生效 |
| Token 存储 | AES-256-GCM 加密落库，`json:"-"` 不返回前端 | 存 token 为未来 API 联动预留，但绝不在列表 API 暴露 |
| State 格式 | base64 JSON 扩展 | 登录: `{domain,source}`；绑定: `{domain,source,userId,intent:"bind"}` |

---

## 2. 数据模型

### 2.1 sys_social 关联表

```go
// server/model/system/sys_social.go

type SysSocial struct {
    global.OPS_AUDIT_MODEL                     // 审计基座
    ID           int64  `gorm:"primarykey;column:id" json:"id,string"`
    UserId       int64  `gorm:"index;comment:本地用户ID" json:"userId,string"`
    Source       string `gorm:"size:32;comment:来源标识" json:"source"`         // "wechat_open"/"gitee"/"github"
    OpenId       string `gorm:"size:128;comment:第三方用户唯一标识" json:"openId"`
    UnionId      string `gorm:"size:128;comment:微信 unionid" json:"unionId"`   // 仅微信有意义
    AuthId       string `gorm:"size:128;comment:认证唯一ID" json:"authId"`      // source+"_"+openId 拼接
    NickName     string `gorm:"size:64;comment:第三方昵称" json:"nickName"`      // 资料快照
    Avatar       string `gorm:"size:512;comment:第三方头像URL" json:"avatar"`    // 资料快照
    Email        string `gorm:"size:128;comment:第三方邮箱" json:"email"`        // 资料快照
    AccessToken  string `gorm:"size:2048;comment:访问令牌(AES加密)" json:"-"`    // 加密存储,不返回前端
    RefreshToken string `gorm:"size:2048;comment:刷新令牌(AES加密)" json:"-"`    // 加密存储,不返回前端
    ExpireIn     int64  `gorm:"comment:令牌有效期(秒)" json:"expireIn"`          // 0=无有效期信息
}

func (SysSocial) TableName() string { return "sys_social" }
```

**唯一约束**：
- `(source, open_id)` — 一个第三方账号只能绑一个本地用户
- `(user_id, source)` — 一个本地用户同一 provider 只能绑一个

**authId 拼接规则**：`source + "_" + openId`（如 `gitee_12345`、`wechat_open_o6_bmjr...`），对齐 JustAuth 风格和前端类型定义。

**UnionId 语义**：仅微信开放平台返回。绑定时若 unionId 非空，查重优先用 unionId（避免同一微信用户在不同应用下被当成两个人）。

### 2.2 字段精简理由

前端 `Api.System.Social` 类型有 20+ 字段（JustAuth 全平台风格），但本项目只用 3 个 source。以下字段**不落库**（返回空值/默认值给前端）：

- `accessCode` / `scope` / `tokenType` / `idToken` / `macAlgorithm` / `macKey` / `code` / `oauthToken` / `oauthTokenSecret` — 小米/Twitter 等平台专用字段
- `userName` — 与 `nickName` 重复，绑定时的昵称快照足够

列表 API 返回的 Social 对象中，这些字段统一为空字符串/0，前端不消费。

---

## 3. API 接口设计

### 3.1 四个 Endpoint（严格对齐前端契约）

| # | Method | URL | 路由组 | Handler | 说明 |
|---|--------|-----|--------|---------|------|
| 1 | GET | `/auth/binding/{source}?domain={host}` | **PublicGroup** | `SocialApi.GetAuthURL` | 返回授权 URL 字符串 |
| 2 | POST | `/auth/social/callback` | **PublicGroup** | `SocialApi.Callback` | 交换 code + 处理绑定/登录 |
| 3 | GET | `/system/social/list` | **PrivateGroup** | `SocialApi.List` | 当前用户已绑定列表 |
| 4 | DELETE | `/auth/unlock/{id}` | **PrivateGroup** | `SocialApi.Unbind` | 解绑指定关联 |

**路由注册**（`SocialRouter.InitSocialRouter(PrivateGroup, PublicGroup)`）：
- `/auth/binding/{source}` 和 `/auth/social/callback` 挂 PublicGroup（登录页免鉴权调用）
- `/system/social/list` 和 `/auth/unlock/{id}` 挂 PrivateGroup（需 JWTAuth）

### 3.2 GET /auth/binding/{source} 详细设计

**功能**：拼出三方授权 URL，返回给前端。前端拿到后 `window.location.href = data` 跳转。

**Handler 实现**：
1. 校验 source 合法（`wechat_open`/`gitee`/`github`）
2. 从 `AuthConfigService.Current(ctx)` 读该 provider 配置，校验 `Enabled=true`
3. **手动尝试解析 JWT cookie**（不用 JWTAuth 中间件，因为这个路由在 PublicGroup）：
   - 如果 cookie 有效 → 从 claims 取 userId → state 加 `userId + intent:"bind"`
   - 如果无效/不存在 → state 只带 `domain + source`
4. state = base64(JSON)，格式：
   - 登录场景: `{"domain":"xxx.com","source":"github"}`
   - 绑定场景: `{"domain":"xxx.com","source":"github","userId":12345,"intent":"bind"}`
5. 拼 authorize URL：
   - GitHub/Gitee: 用 `oauth2.Config.AuthCodeURL(state, oauth2.SetAuthURLParam("redirect_uri", cfg.CallbackUrl))`
   - 微信: 手动拼 `https://open.weixin.qq.com/connect/qrconnect?appid=...&redirect_uri=...&response_type=code&scope=snsapi_login&state=...`
6. 返回 `response.OkWithString(authorizeURL, c)`

**关于 CallbackUrl**：管理员在 `sys_auth_config` 配置的 CallbackUrl 是前端 URL（如 `https://yourdomain.com/social-callback?source=github`），每个 provider 带 `?source=xxx` 区分来源。三方平台回调时保留 query param，前端自然拿到 `code/state/source` 三个参数。

### 3.3 POST /auth/social/callback 详细设计

**请求体**（对齐前端 `Api.Auth.SocialLoginForm`）：

```json
{
  "socialCode": "authorization_code_from_provider",
  "socialState": "base64_state_string",
  "source": "github",
  "grantType": "social",
  "clientId": "optional_frontend_env_value"
}
```

**处理流程**：

```
1. 解码 state → {domain, source, userId?, intent?}
2. 校验 state.source == request.source（防参数篡改）
3. 校验 domain == request.Host（可选,防跨域重放）
4. 根据 source 读 AuthConfig → 取 clientId/clientSecret/callbackUrl
5. 交换 code 换 access_token:
   - GitHub/Gitee: oauth2.Config.Exchange(ctx, socialCode)
   - 微信: 自定义 wechatTokenExchange(appId, secret, code) → 返回 {access_token, openid, unionid}
6. 拉用户信息:
   - GitHub: GET https://api.github.com/user (Authorization: Bearer token)
   - Gitee: GET https://gitee.com/api/v5/user?access_token=token
   - 微信: GET https://api.weixin.qq.com/sns/userinfo?access_token=token&openid=openid
7. 分支处理:
   a. intent="bind" + userId 存在 → 绑定流程
      - 校验 userId 的 JWT 合法（防伪造 userId）
      - 查 (source, openId) 是否已绑 → 已绑别人 → 错误"该第三方账号已被其他用户绑定"
      - 微信额外: 查 unionId 是否已绑别人
      - 未被占用 → 写 sys_social 记录（加密存 token）→ 返回成功
   b. 无 userId → 登录流程
      - 按 (source, openId) 查 sys_social → 找到 userId → 查 SysUser → TokenNext 签 JWT → 返回 LoginToken
      - 未找到 → 返回错误"该账号未绑定本地用户，请先注册"
```

**绑定流程中 userId 校验**：不能盲信 state 里的 userId。需要验证请求中携带的 JWT cookie（如果有）对应的 userId 与 state.userId 一致。如果 state 里有 userId 但请求没有合法 JWT → 视为登录流程（拒绝绑定）。

**TokenNext 复用**：登录流程签发 JWT 时，查出 `SysUser` 全量（含 Roles/Dept），调用现有 `TokenNext(c, user, mustChangePwd)` 完成标准签发。`mustChangePwd` 检查按现有逻辑判断。

### 3.4 GET /system/social/list 详细设计

**功能**：返回当前登录用户的已绑定第三方账号列表。

**Handler 实现**：
1. 从 JWT claims 取 userId
2. `SELECT * FROM sys_social WHERE user_id = userId AND deleted_at IS NULL`
3. 返回 Social 列表（accessToken/refreshToken 字段不返回，`json:"-"`）
4. 前端 social-card.vue 消费 `source/nickName/avatar/createTime/id`

### 3.5 DELETE /auth/unlock/{id} 详细设计

**功能**：解绑指定关联记录。

**Handler 实现**：
1. 从 JWT claims 取 userId
2. 查 sys_social 记录 → 确认 `UserId == claims.userId`（防越权解绑别人的）
3. 解绑校验：
   - 计算该用户的"可登录方式计数"：
     - SysUser.Password 非空且 PasswordUpdatedAt 未过期 → 本地密码可登录（+1）
     - sys_social 中 userId=当前 且 id≠本次解绑的 绑定数（+N）
   - total = 0 → 返回错误"请至少保留一种登录方式"
   - total ≥ 1 → 执行软删除（gorm DeletedAt）
4. 返回成功

---

## 4. 微信开放平台特殊处理

### 4.1 差异点

| 差异点 | 标准 OAuth2 | 微信开放平台 |
|--------|------------|-------------|
| token 请求 | `client_id`+`client_secret` 在 body | `appid`+`secret`+`code`+`grant_type` 在 URL query |
| token 响应 | JSON `{access_token,expires_in,...}` | JSON `{access_token,openid,unionid,expires_in}` |
| userinfo | 只需 access_token | 需要 `access_token`+`openid` |
| scope | `profile` 等 | `snsapi_login` |
| authorize URL | `oauth2.Config.AuthCodeURL()` | 手动拼 `https://open.weixin.qq.com/connect/qrconnect?...` |

### 4.2 实现方案

- **token 交换**：自定义 `wechatTokenExchange(appId, secret, code)` 函数，用 `net/http` 直接请求微信 token endpoint
- **userinfo 拉取**：自定义 `wechatUserInfo(token, openId)` 函数
- **查重优先级**：`unionId > openId`。绑定/查询时，如果 unionId 非空，先查 `(source, unionId)` 是否已有记录

### 4.3 微信用户信息映射

微信 userinfo 响应字段 → sys_social 字段映射：
- `openid` → `OpenId`
- `unionid` → `UnionId`
- `nickname` → `NickName`
- `headimgurl` → `Avatar`
- `email` → 不映射（微信开放平台 userinfo 不返回邮箱）

---

## 5. GitHub / Gitee 实现

### 5.1 OAuth2 Endpoint 配置

```go
// GitHub
oauth2.Config{
    ClientID:     cfg.GithubClientId,
    ClientSecret: cfg.GithubClientSecret,
    RedirectURL:  cfg.GithubCallbackUrl,
    Scopes:       []string{"read:user", "user:email"},
    Endpoint:     oauth2.Endpoint{
        AuthURL:  "https://github.com/login/oauth/authorize",
        TokenURL: "https://github.com/login/oauth/access_token",
    },
}

// Gitee
oauth2.Config{
    ClientID:     cfg.GiteeClientId,
    ClientSecret: cfg.GiteeClientSecret,
    RedirectURL:  cfg.GiteeCallbackUrl,
    Scopes:       []string{"user_info", "emails"},
    Endpoint:     oauth2.Endpoint{
        AuthURL:  "https://gitee.com/oauth/authorize",
        TokenURL: "https://gitee.com/oauth/token",
    },
}
```

### 5.2 GitHub 用户信息

GitHub `/user` API 需要额外请求 `/user/emails` 获取邮箱（主邮箱标记为 `primary=true`）。映射：
- `id` (int) → `OpenId` = `strconv.Itoa(id)`
- `login` → 不存（用 nickName 替代）
- `name` → `NickName`（如果空则用 `login`）
- `avatar_url` → `Avatar`
- `email` → `Email`（从 `/user/emails` 取 primary）

### 5.3 Gitee 用户信息

Gitee `/api/v5/user` 直接返回邮箱。映射：
- `id` (int) → `OpenId` = `strconv.Itoa(id)`
- `name` → `NickName`（如果空则用 `login`）
- `avatar_url` → `Avatar`
- `email` → `Email`

---

## 6. 加密存储方案

### 6.1 Token 加密

`accessToken`/`refreshToken` 用 AES-256-GCM 加密后落库。

**密钥来源**：`global.OPS_CONFIG` 新增 `Social.TokenKey` 字段（32 字节 hex string，即 64 字符）。
- 如果未配置 → 启动时 warn 日志，实际有 token 需加密时报错
- 密钥不能硬编码，必须从配置读

**加密流程**：
- 写入时：`aesGCM.Seal(nil, nonce, plaintextToken, nil)` → base64 → 存 DB
- 读取时：base64 decode → `aesGCM.Open(nil, nonce, ciphertext, nil)` → plaintext token

**Nonce 生成**：每次加密随机生成 12 字节 nonce，与 ciphertext 一起 base64 编码存储（`nonce + ciphertext`）。

### 6.2 列表 API 不返回 Token

`SysSocial.AccessToken` 和 `RefreshToken` 的 json tag 是 `json:"-"`，GORM 序列化时自动排除。前端永远不会收到这些字段。

---

## 7. 错误码

| 场景 | 错误码前缀 | message |
|------|-----------|---------|
| 绑定成功 | 0 | "绑定成功" |
| 绑定-已被别人绑 | 7401 | "该第三方账号已被其他用户绑定" |
| 绑定-自己已绑同 source | 7402 | "您已绑定该平台的账号" |
| 登录成功 | 0 | "登录成功"（+ LoginToken） |
| 登录-未注册 | 7403 | "该账号未绑定本地用户，请先注册" |
| 解绑-无其他登录方式 | 7404 | "请至少保留一种登录方式" |
| 解绑-越权 | 7405 | "无权解绑该账号" |
| 解绑成功 | 0 | "解绑成功" |
| state 校验失败 | 7406 | "授权状态校验失败，请重试" |
| provider 未启用 | 7407 | "该第三方登录未启用" |
| code 交换失败 | 7408 | "第三方授权失败，请重试" |
| source 不合法 | 7409 | "不支持的登录方式" |

错误码具体数值在实现时对齐项目现有 response 体系。核心是 message 明确可展示。

---

## 8. 代码结构

### 8.1 新增文件

| 文件路径 | 说明 |
|----------|------|
| `server/model/system/sys_social.go` | SysSocial 模型 + TableName() |
| `server/model/system/request/sys_social.go` | 请求类型定义 |
| `server/service/system/sys_social.go` | SocialService 全部业务逻辑 |
| `server/api/v1/system/sys_social.go` | SocialApi 4 个 handler |
| `server/router/system/sys_social.go` | SocialRouter 注册 |
| `server/utils/crypto/aes.go` | AES-256-GCM 加密/解密工具（如果项目中没有现成的） |

### 8.2 修改文件

| 文件路径 | 改什么 |
|----------|--------|
| `server/service/system/enter.go` | ServiceGroup 加 `SocialService` |
| `server/api/v1/system/enter.go` | ApiGroup 加 `SocialApi`，加 `socialService` var |
| `server/router/system/enter.go` | RouterGroup 加 `SocialRouter`，加 `socialApi` var |
| `server/initialize/router.go` | 加 `systemRouter.InitSocialRouter(PrivateGroup, PublicGroup)` |
| `server/initialize/gorm.go`（或等价） | AutoMigrate 加 `&system.SysSocial{}` |
| `server/go.mod` | 加 `golang.org/x/oauth2` |
| `server/global/config.go`（或等价） | 加 `Social.TokenKey` 配置结构（可选，如果不从配置读密钥则用环境变量） |

### 8.3 新增依赖

| 依赖 | 用途 |
|------|------|
| `golang.org/x/oauth2` | GitHub/Gitee 标准 OAuth2 code 交换 |

微信不走 oauth2.Config（参数不标准），用 net/http 手动拼 URL。

---

## 9. 前端契约对照

### 9.1 前端已定义的 API 调用

| 前端调用 | 方法/URL | 后端对齐 |
|----------|----------|----------|
| `fetchSocialAuthBinding(source)` | `GET /auth/binding/${source}?domain={host}` → `string` | 1: GetAuthURL |
| `fetchSocialList()` | `GET /system/social/list` → `Social[]` | 3: List |
| `fetchSocialAuthUnbinding(id)` | `DELETE /auth/unlock/${id}` | 4: Unbind |
| `fetchSocialLoginCallback(data)` | `POST /auth/social/callback` → void | 2: Callback |

### 9.2 前端回调流程

`social-callback/index.vue` 的处理：
- 未登录 → `loginByCode(data)` → 调 `authStore.login(data)`（走登录流程）
- 已登录 → `callbackByCode(data)` → 调 `fetchSocialLoginCallback(data)`（走绑定流程）

**后端不依赖前端 `authStore.isLogin` 判断**。后端用 state 里的 userId/intent 权威判断是登录还是绑定。前端分流只是为了选择调用哪个前端函数，两种路径最终都 POST 到同一个 `/auth/social/callback` endpoint。

### 9.3 前端类型对齐

后端返回的 Social 列表 JSON 需对齐 `Api.System.Social` 类型：
- 有值的字段：`id, userId, source, authId, openId, unionId, nickName, avatar, email, expireIn, createTime`
- 空值字段：`accessToken, refreshToken, userName, accessCode, scope, tokenType, idToken, macAlgorithm, macKey, code, oauthToken, oauthTokenSecret` → 返回空字符串/0

前端 `social-card.vue` 实际只消费 `source, nickName, avatar, createTime, id`，其他字段虽在类型里但不影响 UI。

---

## 10. 已知限制 & 后续扩展

1. **企业微信扫码登录**：本设计不包含。`sys_auth_config` 的 wecom 字段保留，未来扩展时复用。
2. **未登录社交登录前端链路**：`authStore.login` 当前把 social 字段丢了（soybean 示例遗留）。绑定功能不受影响，但做登录流程前需要修前端 auth store。本设计后端先按 POST `/auth/social/callback` 实现，登录成功时返回标准 `LoginToken`（httpOnly cookie），前端 `callbackByCode` 路径已经能接住。`loginByCode` 路径需要后续修前端 auth store。
3. **token 过期刷新**：当前设计只存 token，不实现自动刷新。未来如果需要调三方 API，可以加 refresh_token 自动刷新逻辑。
4. **钉钉残留字段**：`sys_auth_config` 的 dingtalk 字段保留（前端已删展示，后端模型残留），不影响本设计。
