# Soybean 前端 ↔ devops-admin 后端 接口差异体检报告

> 体检时间:2026-07-18
> 对照三方:Soybean Admin 前端(`/home/devops-admin/web`,`soybean-admin@2.2.0`)、当前后端(`/home/devops-admin/server`)、对标源 gin-vue-admin 3.0-beta(`/home/remote/gin-vue-admin/server`)
> 用途:作为"实现后台管理功能时,结构借鉴 GVA、接口对齐 Soybean 前端"的依据。详细的前端接口清单与 GVA 四件套清单见会话历史或重新分析。

---

## 0. 核心结论(TL;DR)

| 维度 | 状态 | 说明 |
|---|---|---|
| 统一响应封装 | ✅ 已对齐 | `response.go` 已改为 `Code string`、`SUCCESS="0000"`/`ERROR="7"`,字段 `code/data/msg`。**不是 GVA 原生的 `code int 0/7`** |
| 登录/验证码/初始化路径 | 🟡 半对接 | `/base/login`、`/base/captcha`、`/init/*` 路径与前端一致;但登录返回体仍带 `user/token`,而前端只要 `{expiresAt}`、token 应走 httpOnly Cookie |
| 用户信息 `getUserInfo` | ❌ 路径+结构不符 | 后端 `GET /user/getUserInfo`;前端要 `GET /auth/getUserInfo`,返回 `{user, roles[], permissions[]}` |
| 业务模块 CRUD(用户/角色/菜单/部门/岗位/字典/日志) | ❌ **最大缺口** | 后端是 GVA 动词风格(`/user/getUserList`、`/authority/createAuthority`);前端是 RuoYi RESTful 风格(`/system/user/list`、`/system/role`)。**两套几乎不重叠,前端调不通** |
| 鉴权方式 | ⚠ 待确认 | 前端用 httpOnly Cookie + `Clientid` header,不放 Authorization;后端当前用 JWT(需确认 SetToken 是否写 httpOnly Cookie) |
| 请求加密 | ⚠ 缺中间件 | 前端 `resetPwd`/`updatePwd` 走 RSA-512 + AES-ECB 加密;后端需补解密中间件 |
| 分页结构 | ❌ 不符 | 后端返回 `{list,total,page,pageSize}`(GVA 风格);前端要 `{rows,total,pageNum,pageSize}` |

**最重要的方向修正**:前端 Soybean 是 **RuoYi 系接口契约**(RESTful + `/list` 后缀 + 按钮权限 `perms` + `menuType M/C/F` + 状态值字符串 `"0"/"1"`),与 GVA 的接口设计**几乎完全不同**。因此:
- **后端代码分层/业务逻辑实现** → 借鉴 GVA 四件套(api/router/service/model)✅
- **接口路径、请求/响应字段、鉴权、分页** → **必须按 Soybean 前端实现**,GVA 接口设计**基本不可直接复用**,只能当"业务逻辑参考"。

---

## 1. 响应封装(已对齐,保持现状)

| 项 | GVA 3.0-beta | 当前后端 | Soybean 前端要求 |
|---|---|---|---|
| code 类型 | `int`(0/7) | `string`("0000"/"7") | `string`("0000") |
| 成功码 | `0` | `"0000"` | `"0000"` |
| 失败码 | `7` | `"7"` | 任意非 "0000" |
| 字段 | code/data/msg | code/data/msg | code/data/msg + `rows?`/`total?`(分页) |
| HTTP 状态 | 200(401 鉴权) | 200(401 鉴权) | 200 |

⚠ **分页接口**:前端 `transform` 在 `response.data.rows` 存在时返回整个 body,否则取 `data`。所以**列表分页接口的 data 必须形如 `{rows,total,pageNum,pageSize}`**,不能用当前 `response.PageResult{List,Total,Page,PageSize}`。见 §6。

---

## 2. 鉴权与加密约定(前端强约束)

### 2.1 鉴权:httpOnly Cookie + Clientid(非 Authorization)
- 前端 `getAuthorization()` 返回 `null`,**不放 Authorization 头**;登录成功靠后端 `Set-Cookie`(httpOnly)写 access/refresh token。
- 登录响应体**只回 `{expiresAt: 毫秒时间戳}`**,前端在 token 寿命 80% 时主动调 `/auth/refreshToken`。
- 每个请求额外注入 header:
  - `Clientid: e5cd7e4891bf95d1d19206ce24a7b32e`(注意 header 名是 `Clientid`,驼峰无连字符)
  - `Content-Language: zh_CN`/`en_US`(下划线)
  - `X-Request-Id: nanoid`
- ⚠ **跨域风险**:前端 axios 未设 `withCredentials`。同源/开发代理 OK;跨域部署需前端补 `withCredentials:true` + 后端 `Access-Control-Allow-Credentials:true` 且 Origin 不能为 `*`。

> 当前后端 `Login` 仍返回 `{user, token, expiresAt, needChangePassword}` 并调 `utils.SetToken`。**需要确认 `utils.SetToken` 是否写 httpOnly Cookie**;若是写 header/返回体,需改为 Set-Cookie。

### 2.2 业务状态码约定(前端拦截器行为)
| code | 前端行为 |
|---|---|
| `0000` | 成功 |
| `8888`,`8889` | 静默登出(未登录态) |
| `7777`,`7778` | 弹窗"登录过期"后登出(已登录态) |
| `9999`,`9998`,`3333` | token 过期 → 自动 refreshToken 重放 |

### 2.3 请求加密(RSA-512 + AES-ECB-PKCS7)
- 总开关 `VITE_APP_ENCRYPT=Y`,header flag `encrypt-key`。
- 流程:客户端生成 32 字节 AES key → RSA 公钥加密后放 `encrypt-key` 头 → body 用 AES-ECB-PKCS7 加密。
- **当前仅 `resetPwd`、`updatePwd` 两个接口显式开 `isEncrypt`**;登录密码当前**明文**(未开 isEncrypt)。
- 后端需:**请求解密中间件**(从 `encrypt-key` 头用 RSA 私钥解出 AES key,解密 body);可选响应加密中间件。RSA 密钥对必须与 `.env.test` 的公私钥严格配对。

### 2.4 其他前端约定
- 防重复提交:POST/PUT 默认 500ms 内同 URL+body 拦截。
- SSE:仅 `VITE_APP_SSE=Y` 开关,前端**无 SSE 客户端实现**,后端 SSE 端点暂不上线。
- 路由模式:`VITE_AUTH_ROUTE_MODE=static`,实际调 `/route/getConstantRoutes`(返回数组);dynamic 模式才调 `/route/getUserRoutes`。

---

## 3. 接口现状对照(已实现部分)

当前后端 `RouterPrefix=""`,已注册路由:

| 路径 | method | handler | 对接前端 | 状态 |
|---|---|---|---|---|
| `/base/login` | POST | Login | `/base/login` | 🟡 路径对,返回体待精简 |
| `/base/captcha` | POST | Captcha | `/base/captcha`(GET!) | 🟡 前端是 **GET**,当前是 POST |
| `/init/initdb` | POST | InitDB | `/init/initdb` | ✅ |
| `/init/checkdb` | POST | CheckDB | `/init/checkdb` | ✅ |
| `/init/conn-test` | POST | PingDB | `/init/conn-test` | ✅ |
| `/init/ping-redis` | POST | PingRedis | `/init/ping-redis` | ✅ |
| `/user/admin_register` | POST | Register | 无对应(前端用 `/system/user` POST) | ❌ |
| `/user/getUserList` | POST | GetUserList | 前端 `GET /system/user/list` | ❌ 风格不符 |
| `/user/getUserInfo` | GET | GetUserInfo | 前端 `GET /auth/getUserInfo` | ❌ 路径+结构不符 |

> 注:`router/system/sys_user.go` 中 changePassword/deleteUser/setUserInfo/setUserAuthorities 等大量路由**已被注释**,说明业务模块尚未真正实现。

### 3.1 验证码协议差异(go-captcha,非传统字符图)
前端 `GET /base/captcha?username=xxx` 期望返回:
```
{ code:"0000", data:{
  captchaEnabled, type:"click|slide|rotate", captchaId,
  masterImage(base64), tileImage, thumbImage,
  thumbX/Y/Width/Height, angle, thumbSize
}}
```
当前后端 `Captcha` 接口需确认是否返回该结构(基于 GVA 的 `SysCaptchaResponse`,字段可能不一致)。**且 method 要从 POST 改 GET**。

### 3.2 getUserInfo 差异(高优先级)
前端 `GET /auth/getUserInfo` 期望:
```
{ code:"0000", data:{
  user:{ userId, deptId, userName, nickName, avatar, roles:[...], ... },
  roles:["admin","super"],          // roleKey 字符串数组(不是 roleName)
  permissions:["*:*:*","system:user:list"]  // perms 字符串数组
}}
```
当前后端 `GET /user/getUserInfo` 返回 `gin.H{"userInfo":...}`,**缺顶层 roles/permissions、路径前缀错位**。

---

## 4. 接口缺口清单(按前端业务模块)

> 这是**最大缺口区**。前端清单来自 `web/src/service/api/`。下表"后端现状"为空表示**完全未实现**;"GVA 模板"指可在 `/home/remote/gin-vue-admin/server` 借鉴业务实现的模块。

### 4.1 auth(认证)
| 前端接口 | method | 后端现状 | GVA 模板 |
|---|---|---|---|
| `/base/captcha` | GET | ❌(POST,结构待核) | sys_base/captcha |
| `/base/login` | POST | 🟡 路径有 | sys_base/Login |
| `/auth/getUserInfo` | GET | ❌(路径/结构不符) | sys_user/GetUserInfo |
| `/auth/refreshToken` | POST | ❌ | sys_jwt |
| `/auth/logout` | POST | ❌ | sys_base(登出+黑名单) |
| `/auth/register`,`/auth/social/*` | - | — | 示例保留,可不做 |

### 4.2 route(动态菜单)
| 前端接口 | method | 后端现状 | GVA 模板 |
|---|---|---|---|
| `/route/getConstantRoutes` | GET | ❌ | sys_menu(GetMenu) |
| `/route/getUserRoutes` | GET | ❌ | sys_menu |
| `/route/isRouteExist` | GET | ❌ | — |

> ⚠ 前端 static 模式仍调 `getConstantRoutes`;路由 `name/path/component` 必须严格匹配 `elegant-router.d.ts` 生成的 `RouteKey`,否则前端路由解析失败。GVA 的菜单结构与 Soybean 的 `MenuRoute` 结构差异大,需做转换层。

### 4.3 system/user(用户管理)
| 前端接口 | method | 后端现状 | GVA 模板 |
|---|---|---|---|
| `/system/user/list` | GET | ❌ | sys_user.GetUserList |
| `/system/user` | POST/PUT | ❌(register 风格不符) | Register/SetUserInfo |
| `/system/user/changeStatus` | PUT | ❌ | — |
| `/system/user/{userIds}` | DELETE | ❌ | DeleteUser |
| `/system/user/{userId}` | GET | ❌ | GetUserInfo |
| `/system/user/deptTree` | GET | ❌ | sys_department |
| `/system/user/resetPwd` | PUT(**加密**) | ❌ | ResetPassword |
| `/system/user/profile` | PUT | ❌ | SetSelfInfo |
| `/system/user/profile/updatePwd` | PUT(**加密**) | ❌ | ChangePassword |
| `/system/user/profile/avatar` | POST(FormData) | ❌ | 头像上传 |

### 4.4 system/role(角色)
| 前端接口 | method | 后端现状 | GVA 模板 |
|---|---|---|---|
| `/system/role/list` | GET | ❌ | sys_authority.GetAuthorityList |
| `/system/role` | POST/PUT | ❌ | CreateAuthority/UpdateAuthority |
| `/system/role/changeStatus` | PUT | ❌ | — |
| `/system/role/{roleIds}` | DELETE | ❌ | DeleteAuthority |
| `/system/role/authUser/*` | GET/PUT | ❌ | SetRoleUsers |

> GVA 的 `authority` ≈ RuoYi 的 `role`;字段差异:GVA 用 `authorityId`,前端用 `roleId/roleKey/roleSort/menuCheckStrictly/menuIds`。

### 4.5 system/menu(菜单+按钮)
| 前端接口 | method | 后端现状 | GVA 模板 |
|---|---|---|---|
| `/system/menu/list` | GET | ❌ | sys_menu.GetMenuList |
| `/system/menu` | POST/PUT | ❌ | AddBaseMenu/UpdateBaseMenu |
| `/system/menu/{menuId}` | DELETE | ❌ | DeleteBaseMenu |
| `/system/menu/treeselect` | GET | ❌ | GetBaseMenuTree |
| `/system/menu/roleMenuTreeselect/{roleId}` | GET | ❌ | GetMenuAuthority |

> 字段差异(关键):前端菜单 `menuType:'M'|'C'|'F'`、`perms:"system:user:add"`、`visible/status/isFrame/isCache` 全是字符串 `"0"/"1"`。GVA 的 `SysBaseMenu` 结构不同,需重新设计 model。

### 4.6 system/dept(部门)
| 前端接口 | method | 后端现状 | GVA 模板 |
|---|---|---|---|
| `/system/dept/list`(树) | GET | ❌ | sys_department |
| `/system/dept` | POST/PUT | ❌ | CreateDepartment |
| `/system/dept/{deptIds}` | DELETE | ❌ | DeleteDepartment |
| `/system/dept/optionselect` | GET | ❌ | — |

### 4.7 system/post(岗位)
| 前端接口 | method | 后端现状 | GVA 模板 |
|---|---|---|---|
| `/system/post/list` | GET | ❌ | sys_position |
| `/system/post` | POST/PUT | ❌ | CreatePosition |
| `/system/post/{postIds}` | DELETE | ❌ | DeletePosition |
| `/system/post/optionselect` | GET | ❌ | — |

### 4.8 system/dict(字典类型+数据)
| 前端接口 | method | 后端现状 | GVA 模板 |
|---|---|---|---|
| `/system/dict/data/type/{dictType}` | GET | ❌ | sys_dictionary_detail |
| `/system/dict/type/list` | GET | ❌ | sys_dictionary |
| `/system/dict/type` | POST/PUT | ❌ | CreateSysDictionary |
| `/system/dict/type/{dictIds}` | DELETE | ❌ | DeleteSysDictionary |
| `/system/dict/data/list` | GET | ❌ | sys_dictionary_detail |
| `/system/dict/data` | POST/PUT | ❌ | — |
| `/system/dict/data/{dictCodes}` | DELETE | ❌ | — |
| `/system/dict/type/refreshCache` | DELETE | ❌ | — |

> ⚠ 字典数据 `listClass` 必须是 NaiveUI 枚举:`default|primary|info|success|warning|error`。`dictValue`/`dictLabel`/`listClass` 三元组。

### 4.9 其他模块
| 模块 | 前端路径前缀 | 后端现状 | GVA 模板 |
|---|---|---|---|
| notice 通知 | `/system/notice` | ❌ | (GVA 无直接对应) |
| setting 设置 | `/system/setting` | ❌ | sys_system/sys_security_config |
| oss 存储 | `/resource/oss` | ❌(media 模块注释) | — |
| loginlog 登录日志 | `/log/loginlog` | ❌ | sys_login_log |
| operlog 操作日志 | `/monitor/operlog` | ❌ | sys_operation_record |
| online 在线设备 | `/monitor/online` | ❌ | (jwt 多点登录) |
| social 社交绑定 | `/auth/binding`、`/system/social` | ❌ | (示例保留) |

---

## 5. 关键字段级差异(易错点)

1. **分页响应**:`{rows, total, pageNum, pageSize}`(非 `list/records/items`)。
2. **状态值**:菜单/部门/字典的 `status`/`visible`/`isFrame`/`isCache` 统一字符串 `"0"/"1"`(非 bool/int)。
3. **角色标识**:顶层 `roles` 是 `roleKey` 字符串数组(`["admin","super"]`),与 `user.roles` 对象数组同名不同型。
4. **权限标识**:`permissions` 用三段式 `模块:资源:动作`(`system:user:add`),超管 `*:*:*`。
5. **超管判定**:static 模式下 `roles` 含 `"super"`(`VITE_STATIC_SUPER_ROLE`)即全路由。
6. **字典 listClass**:NaiveUI 6 种主题色枚举之一。
7. **`fetchGetMenuTreeSelect` URL 缺前导斜杠**(`system/menu/treeselect`),后端路由要兼容。
8. **登录响应**:只回 `{expiresAt}`,token 走 Cookie。

---

## 6. 通用分页响应结构改造建议

当前:`model/common/response/common.go` 的 `PageResult{List,Total,Page,PageSize}`。
前端要:`{rows,total,pageNum,pageSize}`。

**建议**:新增一个 Soybean 风格分页结构(或改造现有),所有对接前端的列表接口统一返回:
```go
type RowResult struct {
    Rows     interface{} `json:"rows"`
    Total    int64       `json:"total"`
    PageNum  int         `json:"pageNum"`
    PageSize int         `json:"pageSize"`
}
```
同时入参从 GVA 的 `PageInfo{Page,PageSize,Keyword}` 调整为前端的 `CommonSearchParams{pageNum,pageSize,orderByColumn,isAsc,params}`。

---

## 7. 重构与对接路线图

### P0 — 打通登录到首页(最高优先级)
1. **验证码** `/base/captcha`:改 GET,返回 go-captcha 结构(masterImage/captchaId/type 等)。
2. **登录** `/base/login`:返回体精简为 `{expiresAt, needChangePassword?}`;确认 `utils.SetToken` 改为写 httpOnly Cookie;补 `Clientid` 校验。
3. **用户信息** `/auth/getUserInfo`:新增路径,返回 `{user, roles[], permissions[]}`;从 GVA 的 `sys_user/GetUserInfo` 借鉴取数逻辑,但重新组装结构。
4. **登出** `/auth/logout`:清 Cookie + JWT 入黑名单。
5. **刷新 token** `/auth/refreshToken`:Cookie 续期。
6. **常量路由** `/route/getConstantRoutes`:返回 elegant-router 兼容的 `MenuRoute[]`。

### P1 — 后台管理核心 CRUD(按前端页面逐个补)
按顺序:用户管理 → 角色管理 → 菜单管理 → 部门 → 岗位 → 字典。每个模块:
- 按 RuoYi 路径设计 `router/system/sys_xxx.go`
- model 按前端 typings 字段设计(状态字符串、perms、menuType 等)
- service 业务逻辑借鉴 GVA 对应模块
- 列表接口返回 `RowResult{rows,total,...}`

### P2 — 辅助功能
登录日志、操作日志、通知公告、系统设置、在线设备、OSS。

### P3 — 可选
加密中间件(resetPwd/updatePwd)、refreshCache、社交登录、动态路由(dynamic 模式)。

---

## 8. 代码组织建议(借鉴 GVA 四件套)

- **保留 GVA 分层**:`api/v1/system`、`router/system`、`service/system`、`model/system`(+`request`/`response`),三层 `enter.go` 聚合,包级 `xxxApi`/`xxxService` 注入。
- **响应封装保持现状**(已对齐 "0000")。
- **路由前缀**:前端路径已带 `/system`、`/auth`、`/log`、`/monitor` 前缀。建议后端 group 按 `/system`、`/auth` 等分,而不是 GVA 的 `/user`、`/authority`。
- **OperationRecord 中间件**:写操作挂,读操作不挂;`sys_operation_record` 自身不挂(防递归)——沿用 GVA 做法。
- **多对多**:用显式连接表 struct + `TableName()`(GVA 风格),不依赖 GORM 隐式建表。
- **Casbin**:GVA 的 Casbin 策略基于 API path;前端按钮权限用 `perms` 字符串。需设计 perms 与 Casbin 策略的映射,或改用前端 `permissions` 下发 + 后端按 perms 校验。

> ⚠ **代码生成器(sys_auto_code)**:GVA 的代码生成器产出的四件套骨架是 GVA 风格接口,**不能直接用于对接 Soybean**。可用它生成 service/model 骨架,但 router/api 的路径与字段必须手改成 RuoYi 风格。

---

## 附:GVA 3.0-beta 模块与前端页面对照(供借鉴业务实现)

| 前端功能 | GVA group | GVA 关键 model | 借鉴点 |
|---|---|---|---|
| 登录/验证码 | base | SysUser/SysSecurityConfig | 安全配置+锁定+登录日志+JWT黑名单 |
| 用户 | user | SysUser(含 Authorities/Departments/Positions m2m) | 关联表设计 |
| 角色 | authority | SysAuthority(DataScope/DefaultRouter) | 数据权限档位 |
| 菜单 | menu | SysBaseMenu/SysBaseMenuBtn | 菜单树+按钮 |
| 权限 | casbin | (casbin_rule) | 策略管理 |
| 部门 | department | SysDepartment(Ancestors) | 祖级链树 |
| 岗位 | position | SysPosition | 与角色正交 |
| 字典 | sysDictionary+Detail | SysDictionary/Detail(Level/Path) | 自树形+冗余 |
| 操作日志 | sysOperationRecord | SysOperationRecord | 中间件写入 |
| 登录日志 | sysLoginLog | SysLoginLog | 登录内部调用 |
| 系统配置 | system/securityConfig | System/SysSecurityConfig | 单行表+atomic缓存 |
| 数据权限审计 | dataAccessLog | SysDataAccessLog | 异步批量落表 |
| 安全策略 | securityConfig | SysSecurityConfig | 验证码/密码/限流/锁定/过期 |
