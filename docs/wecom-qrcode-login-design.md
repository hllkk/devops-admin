# 企业微信扫码登录 设计文档

> 2026-07-27。对照 `/home/remote/devops-admin` 的企微实现，在当前项目接入企业微信扫码登录。
> 业务记忆见 `aiDoc/memory/business/wecom-qrcode-login.md`；本文为完整设计记录。

## 1. 背景与目标

当前项目已有通用社交登录框架（`sys_social` 表 + OAuth2 方案 B 前端回调），覆盖 wechat_open/gitee/github。
企业微信登录此前未实现，但地基已就位：`sys_auth_config` 预留 wecom 全套字段、登录页企微按钮已渲染、
`WW_verify_*.txt` 可信域名校验路由已注册。

**目标**：接入企业微信扫码登录（PC 内嵌二维码 + 企微客户端 WebView 免登），企业员工扫码即登录。

## 2. 远程实现调研（/home/remote/devops-admin）

远程企微登录是**一条独立专用链路**：

- 独立 API（`sys_wecom_auth.go`）、独立表（`sys_wecom_users`）、独立前端组件（`wecom-login.vue`）
- sceneId/Redis 状态机 + 前端 `setTimeout` 链式轮询
- `oauth2/authorize` + `scope=snsapi_privateinfo`（拿 `user_ticket` → 手机/邮箱）
- access_token 缓存（Redis + 内存降级 + sync.Mutex）
- 双 cookie（access 24h + refresh 7d）、自动建号（用户名=企微 userid，默认角色）

## 3. 方案选型

| 维度 | 远程 | 当前项目（选定） |
|---|---|---|
| 身份存储 | 独立 `sys_wecom_users` | **复用 `sys_social`**（source=wecom） |
| token 下发 | 双 cookie | **单 `x-token` cookie**（复用 `TokenNext`） |
| access_token 缓存 | Redis + 内存 + Mutex | **`OPS_CACHE` + singleflight**（项目统一设施） |
| 交互 | 内嵌二维码 + 轮询 + WebView | **同远程**（借鉴体验） |
| 建号 | 自动建号 | **自动建号**（企微是企业内部场景） |
| 默认角色 | `GetDefaultRole`（角色码兜底） | **`SysGeneralConfig.DefaultRoleId`**（常规配置通用项） |

**核心原则**：交互体验借鉴远程，基础设施复用当前项目，不另起平行链路（避免维护两套身份/token 模型）。

## 4. 架构与时序

### 4.1 PC 扫码

```
前端 wecom-login.vue        后端                       企业微信
  │ GET /auth/wecomLogin      │
  │ ← {sceneId, oauthUrl}     │ (sceneId 写 Redis, TTL 2min)
  │ 渲染二维码(qrcode canvas)  │
  │ 轮询 GET /auth/qrCodeStatus?sceneId= (3s→扫码后 1s)
  │                            │
  │              (用户企微 App 扫码确认)
  │                            ← GET /wecomCallback?code=&state=sceneId
  │                            ├ code→userid+user_ticket→手机/邮箱
  │                            ├ 查/建 sys_social(wecom) + sys_user
  │                            ├ 签发 JWT(暂存 Redis confirmed+token)
  │                            → HTML"请在电脑端继续"
  │ 轮询命中 confirmed:        │
  │   ← Set-Cookie: x-token    │ (Del sceneId 防重放)
  │   ← {expiresAt}            │
  │ authStore.wecomLogin → getUserInfo → 跳首页
```

**关键**：PC 回调来自企业微信服务器（非用户浏览器），token 不能在回调响应直接下发，
暂存 Redis 由前端轮询命中 `confirmed` 时下发 cookie（拿用户真实 IP/UA）。

### 4.2 企微客户端 WebView 免登

检测 `wxwork` UA → 调 `/auth/wecomWebviewLogin` 拿 `{oauthUrl}` → `location.replace` →
企微静默授权 → 回调 `/wecomCallback`（回调即用户浏览器）→ 直接下发 cookie +
返回 HTML（同源设 `localStorage` + 跳 SPA）。

## 5. 关键决策

1. **授权方式**：`oauth2/authorize` + `snsapi_privateinfo`。能拿 `user_ticket` 换手机/邮箱；
   `qrConnect` iframe 方式拿不到，已弃用。
2. **身份存储复用 `sys_social`**：`source=wecom`，`open_id`=企微 userid，加 `Mobile` 字段。
   不另起 `sys_wecom_users` 表，用户中心 `social-card` 统一管理。
3. **token 单 `x-token`**：复用 `utils.SetToken`/`TokenNext`，不引入双 cookie（当前项目
   `boundary.md` 的双 cookie 是未落地图纸）。
4. **access_token 缓存**：`OPS_CACHE`（Redis 优先降级内存）+ `OPS_Concurrency_Control`
   （singleflight）防击穿，提前 120s 过期。
5. **自动建号**：企微是企业内部场景，员工扫码即应可用。与 [[social-binding]] 的"拒绝自动建号"
   形成对照。用户名 `wecom_<userid>`，密码随机不可用，`PasswordUpdatedAt=now` 防过期判定。
6. **默认角色通用化**：`SysGeneralConfig.DefaultRoleId`（常规配置），供企微/LDAP/后续注册复用，
   非企微专属。前端复用改造后的 `RoleSelect` 组件（加 `multiple` prop 支持单选）。

## 6. 默认角色设计（通用化）

默认角色从企微专属（`sys_auth_config.WecomDefaultRoleId`）提升为**常规配置通用项**
（`SysGeneralConfig.DefaultRoleId`），理由：

- 后续 LDAP 建号、注册默认角色等功能可复用同一配置
- 集中在「系统设置 → 常规配置」管理，符合"通用默认值"语义
- 前端复用 `RoleSelect` 组件（原多选，改造加 `multiple` prop 支持单选，常规配置单选用）

**迁移**：`sys_auth_config.WecomDefaultRoleId` 删除；`sys_general_config` 加 `DefaultRoleId`
（`int64`，`json:",string"` 对齐前端 `IdType`）。

## 7. 落地清单

### 后端

- `server/utils/wecom.go`（新）：`WecomClient` + access_token 缓存 + 企微 API 封装
- `server/api/v1/system/sys_wecom_auth.go`（新）：`QrCodeView`/`QrCodeStatusView`/`WecomCallbackView`/`WecomWebviewLoginView` + 自动建号
- `server/router/system/sys_wecom.go`（新）：公开路由 + 限流
- `server/model/system/sys_social.go`：加 `Mobile`
- `server/model/system/sys_general_config.go`：加 `DefaultRoleId`
- `enter.go`/`router.go`：注册入口

### 前端

- `web/src/views/_builtin/login/modules/wecom-login.vue`（新）：二维码 + 轮询 + WebView
- `web/src/components/custom/role-select.vue`：加 `multiple` prop 支持单选
- `web/src/views/_admin/system/setting/modules/general-setting.vue`：加默认角色下拉
- `web/src/store/modules/auth/index.ts`：`wecomLogin`
- `web/src/service/api/auth.ts`：3 个 fetch（`fetchWecomQrCode`/`fetchQrCodeStatus`/`fetchWecomWebviewLogin`）
- `web/src/utils/agent.ts`：`isWecomWebview`
- i18n 三处同步（`page.login.wecomLogin.*`）、`social-card` 补 wecom（`autoBind`）

## 8. 配置与部署

1. 「系统设置 → 常规配置」选「默认角色」（企微扫码建号用，必填，未选则建号失败）
2. 「系统设置 → 认证 → 企业微信」填 CorpId/AgentId/Secret/回调地址，启用
3. 企业微信管理端配置可信域名（`WW_verify_*.txt` 由系统自动响应 `/WW_verify_:name.txt`）
4. `AutoMigrate` 启动自动加列：`sys_social.mobile`、`sys_general_config.default_role_id`

## 9. 待办

- 真实企微联调（需企微企业 + 自建应用配置）
- 组织架构同步（远程 `/wecom/syncStructure`，独立功能后续）
