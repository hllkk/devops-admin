# go-captcha 行为验证码登录（go-captcha Login）

> 类型：登录安全增强 · 状态：已重建（2026-07-18）

## 需求

登录集成验证码：双引擎可切换——`image` 传统图形验证码（base64Captcha）与 `click|slide|rotate` 行为验证码（[go-captcha](https://github.com/wenlng/go-captcha)）。前端用配套 [go-captcha-vue](https://github.com/wenlng/go-captcha-vue)。

### 背景

2026-07-14 首次实现（嵌套 `captcha.go-captcha` 配置 + Redis/local_cache 双写），重构 `1d632d9` 删除整个 `/auth/*` 后悬空。2026-07-18 按前端既有契约重建，配置收敛为**扁平字段**（去嵌套、去重复、复用现有图形验证码参数），并增 `enable` 总开关与 `tolerance` 容差。

### 已确认决策（2026-07-18 重建）

- **总开关 `captcha.enable`**：`SysSecurityConfig.CaptchaEnabled`，false 时验证码完全关闭（吸收历史 `trigger.mode=off`）
- **单一扁平 `captcha.type`**：枚举 `image | click | slide | rotate`，一个字段同时表达引擎与 go-captcha 形态；不嵌套
- **复用现有配置（零重复）**：触发阈值复用 `CaptchaOpen`（`0`=每次都要 / `N`=失败N次后触发，吸收历史 `trigger.mode`+`fail-threshold`）；窗口复用 `CaptchaTimeout`（吸收历史 `fail-window`）；命中容差 `captcha.tolerance`（`CaptchaTolerance`，click/slide 像素、rotate 角度）；`KeyLong` 同时作 image 验证码长度与 click 文字点选字符数；image 复用 `ImgWidth/ImgHeight`
- **落库热更新**：`CaptchaEnabled/CaptchaType/CaptchaTolerance` 均落 `SysSecurityConfig`，与现有 captcha 字段一致
- **go-captcha 资源单例 + builder 动态构建**：字体/背景/拼图经 `sync.Once` 加载一次，builder 按 当前 `KeyLong` 每次 Generate 现场构建，保证安全配置改 key-long 即时生效
- **存储统一 `global.OPS_CACHE`**：Redis 优先、不可用降级 memory（复用 `ops_cache.Cache` 抽象，不再自造 Redis+local_cache 双写）；key 前缀 `gocaptcha:`
- **校验先于密码、一次性消费**：`Verify` 供 login 在密码校验前调用，校验即删、TTL 过期、单 captchaId 失败上限 3 次防暴力试答案

### 关键契约（前端 `web/src/typings/api/auth.d.ts` 已就绪，不可变）

- 端点：`GET /auth/captcha?username=`（public），返回 `{captchaEnabled,type,captchaId,masterImage,tileImage,thumbImage,thumbX,thumbY,thumbWidth,thumbHeight,angle,thumbSize}`；`captchaEnabled=false` 表示当前无需验证码
- `POST /auth/login` 请求体携带 `captchaId` + `captcha`（用户答案 JSON：click `[{x,y}]` / slide `{x,y}` / rotate `{angle}` / image 文本）
- 各 type 字段：image→masterImage+captchaId；click→masterImage+thumbImage；slide→masterImage+tileImage+thumbX/Y/W/H；rotate→masterImage+thumbImage+angle(0)+thumbSize

## 实现（2026-07-18 重建）

- 后端：
  - `config/captcha.go`：`Captcha` 增 `Enable/Type/Tolerance`（扁平，替代历史嵌套 `GoCaptcha`；`KeyLong` 复用扩展到 click）
  - `config.yaml`/`config.docker.yaml`：`captcha.{enable,type,tolerance,key-long,...}`
  - `model/system/sys_security_config.go`：增 `CaptchaEnabled/CaptchaTolerance`（落库热更新）+ `DefaultSecurityConfig` 设值
  - `model/system/sys_captcha.go`：`CaptchaResult`（对齐前端契约）
  - `service/system/sys_captcha.go`：`CaptchaService`（provider 接口 + image/click/slide/rotate 四实现 + 资源单例+builder 动态构建 + `OPS_CACHE` 存储 + `Get/Verify/RecordLoginFail/ResetLoginFail`）
  - `api/v1/system/sys_captcha.go`：`CaptchaApi.Captcha` handler
  - `router/system/sys_captcha.go`：`InitCaptchaRouter(PublicGroup)` 注册 `GET /auth/captcha`
  - `service/api/router` 三个 `enter.go` 嵌入 + `initialize/router.go` 调用注册
- 依赖：`github.com/wenlng/go-captcha/v2`、`github.com/wenlng/go-captcha-assets`、`github.com/mojocn/base64Captcha`
- 前端：无改动（`go-captcha-vue` + `fetchCaptcha` + `CaptchaResult` 契约 2026-07-14 已就绪）

## 范围（本次不做）

- `POST /auth/login`/`refreshToken`/`logout`/`getUserInfo`、双 cookie、JWT → 属 [[httponly-cookie-auth]]
- 失败计数累加调用点（login 失败时 `RecordLoginFail`）→ 随 login 重建接入；`CaptchaService` 方法已就绪
- 前端 image 模式传统验证码输入框 UI → 前端任务，后端契约已预留

## 关联

- 承接 [[httponly-cookie-auth]] 登录链路，`Verify` 待 login 重建时在密码校验前接入
- 契约同步进 `aiDoc/frontend-backend/boundary.md`
