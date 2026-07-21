# 不主动启用 casbin

> 长期约束 · 用户明确指示 · 2026-07-20

除非用户明确要求"使用 casbin"，否则不启用 casbin 权限中间件。

## 现状

- `server/initialize/router.go` PrivateGroup 当前仅挂 `JWTAuth + OperationRecord + MustChangePwdGuard + DataScope`，`CasbinHandler()` 被注释（约 line 86）。
- 接口级权限校验未启用；前端 `hasAuth(...)` 仅用于控制按钮显隐，不是后端强制。

## 应用

- 写路由/接口不主动取消 casbin 注释、不主动配 casbin 权限码做强制校验。
- 需要访问控制时走现有数据权限/角色判断，不引 casbin。
- 权限方案讨论时，不把"启用 casbin"作为默认建议。
- 仅当用户明确说"启用/使用 casbin"时才动。
