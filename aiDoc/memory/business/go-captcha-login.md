# go-captcha 行为验证码登录（go-captcha Login）

> 类型：登录安全增强 · 状态：已实现（2026-07-14）

## 需求

登录集成 [go-captcha](https://github.com/wenlng/go-captcha) 行为验证码（click 点选 / slide 滑动 / rotate 旋转），从安全与性能多视角落地：防暴力破解、配置化、低摩擦体验。替换项目原有未启用的 GIF 图片验证码逻辑。前端用配套 [go-captcha-vue](https://github.com/wenlng/go-captcha-vue)。

### 已确认决策

- **三类型可切换**：`config.captcha.go-captcha.type` 切 click | slide | rotate；前端按 type 动态渲染 `go-captcha-vue` 对应组件
- **触发策略**：`trigger.mode` = threshold（默认，账号/IP 连续失败达 `fail-threshold` 次才弹）/ always / off
- **替换原 GIF**：移除 `/auth/code` 占位与前端 GIF 渲染，新增 `/auth/captcha`
- **校验先于密码**：`Login` 在密码校验前先校验验证码，防绕过暴力破解
- **一次性 + TTL**：答案存 Redis（`gocaptcha:` 前缀），校验后立即删；Redis 不可用自动降级进程内 `local_cache`
- **资源单例**：字体/背景/拼图经 `sync.Once` 懒加载，避免每次生成重读内嵌资源（go-captcha-assets）
- **失败计数**：账号与 IP 双维度，登录成功清零、失败各 +1（窗口 `fail-window`）

### 关键契约

- 端点：`GET /auth/captcha?username=`（public），返回 `{captchaEnabled,type,captchaId,masterImage,tileImage,thumbImage,thumbX,thumbY,thumbWidth,thumbHeight,angle,thumbSize}`；`captchaEnabled=false` 表示当前无需验证码
- `POST /auth/login` 请求体携带 `captchaId` + `captcha`（用户答案 JSON：click 为 `[{x,y}]`、slide 为 `{x,y}`、rotate 为 `{angle}`）
- 验证码校验失败 / 密码错误均记失败计数；登录成功清零

## 实现

- 后端：`config/captcha.go` 扩展 `GoCaptcha` 结构；`config.yaml` / `config.docker.yaml` 的 `captcha.go-captcha` 段；`service/system/sys_captcha.go`（Provider 接口 + 三实现 + builder 单例 + Redis/local_cache 存储 + 触发计数）；`api/v1/system/sys_captcha.go` 改占位为生成接口；路由 `auth/captcha`；`Login` 集成先校验验证码
- 前端：`go-captcha-vue@^2`；`service/api/auth.ts` `fetchCaptcha`；`PwdLoginForm` 加 `captchaId/captcha`；auth store `login` 透传（修复原 code/uuid 未进请求体）；`pwd-login.vue` 用 NModal 按 type 渲染组件、confirm 序列化答案、阈值刷新；`register.vue` 移除 GIF；i18n `page.login.captcha.*`
- 依赖：`github.com/wenlng/go-captcha/v2`、`github.com/wenlng/go-captcha-assets`

## 安全与性能要点

- 安全：验证码先于密码、一次性 key 防重放、TTL 过期、失败计数触发、复用既有 IP 限流
- 性能：builder 资源单例、答案存 Redis 不进 DB、Redis 挂降级内存、JPEG 主图压缩

## 关联

- 承接 [[httponly-cookie-auth]] 登录链路，在 `/auth/login` 前置验证码校验
- 契约同步进 `aiDoc/frontend-backend/boundary.md`
