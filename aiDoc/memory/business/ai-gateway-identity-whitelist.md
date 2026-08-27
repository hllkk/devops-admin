# AI 网关·用户自身数据接口加入 casbin 白名单（P0 权限缝隙修复）

- 日期：2026-08-26
- 状态：已实现（middleware 单测覆盖）
- 反向链接：[[ai-gateway-identity-home]]、[[ai-gateway-user-key-cascade]]

## 需求

user 角色种子只授 route.home（无 ApiPrefix），而 home「我的AI身份」页依赖的 `GET /gateway/ai-key/identity/my`、`identity/available-models`、`/gateway/dashboard/*(GET)` 均不在 `rbacWhitelistPrivate` 白名单 → 普通用户打开身份页直接 403；若为修复而授予密钥管理菜单的 `/gateway/ai-key/*` 前缀，会连带放开全部管理 CRUD（菜单级前缀通配，无只读子集策略）。参照 AIHelms 用户端/管理端分离（`/ai-keys/my` 仅登录即可），把"自身数据"接口归入认证而非授权。

## 设计决策

- **白名单精确到 6 个 GET path**（middleware/casbin_rbac.go）：`/gateway/ai-key/identity/my`、`/gateway/ai-key/identity/available-models`、`/gateway/dashboard/{overview,trend,top,budget}`。不能写 `/gateway/dashboard` 前缀——该组还有 `POST aggregate`（手动触发聚合，管理操作）。
- **安全前提（已核实）**：identity/my 的 userId 取 JWT（`utils.GetUserID`，仅查本人主 Key 明文）；dashboard 四接口 API 层 `resolveScope` 非超管强制 `scope=self`（api/v1/gateway/dashboard.go:33-39）——数据范围全部由后端锁定，白名单只跳过"菜单授权"不产生越权面。
- **测试守护**：`middleware/casbin_rbac_test.go` 补双向断言——6 个新路径放行 + `/gateway/ai-key/list`、`/gateway/ai-key/1`、`/gateway/ai-key/scenario/list`、`/gateway/dashboard/aggregate` 不被误放行。

## 注意

- 后续 gateway 新增"所有登录用户可用"的自身数据接口时，同样模式：先核实后端 scope 兜底，再精确 path 加白名单条目 + 测试断言；管理操作永不入白名单。
