# AI 网关·密钥列表列重排（用户名/资源/限流）

- 日期：2026-08-27
- 状态：已实现（go build/test、typecheck/eslint 通过）
- 反向链接：[[ai-gateway-ai-key-layout]]、[[ai-gateway-overview]]

## 需求

管理员视角的密钥列表按用户指定顺序重排：**用户名 → 密钥类型 → Key → 归属 → 资源 → 预算 → 限流 → 状态 → 过期时间 → 最近使用 → 操作**。其中：

- "用户名"= 用户登录本项目的 username（新增列，需后端联表）。
- "归属"只展示 nickname/部门名，不再拼`（用户）`类型后缀。
- "授权模型"改名"**资源**"——为 P2 授权 skill/mcp 预留语义（模型只是资源的一种）。
- 新增"限流"列（原列表未展示限流配置）。
- 保留过期时间与最近使用两列（用户明确要求）。

## 设计决策

- **后端联表扩展**：`AiKeyView` 新增 `OwnerUsername`（user→`sys_users.user_name`，dept 留空）；`fillOwnerNames` 一次 IN 查询同时取 nick_name+user_name（局部 `userBrief` 结构体，不引入 N+1）。注意 SysUser 登录名 Go 字段是 `UserName` 不是 `Username`。
- **场景信息并入密钥类型列**：场景 Key 在类型 NTag 下方以 12px 灰字带出 scenarioName（类型+场景同属"这是什么 Key"），移除独立"场景"列。
- **移除"密钥名称"列**：主 Key 名称恒为 main 无信息量，其归属展示职责已由独立的用户名/归属列承担（原列存在的临时理由消解）。
- **移除"创建时间"列**：信息密度低，表格聚焦（用户清单未包含且未要求保留）。
- **资源列**展示 `N 个模型`（i18n 带参 `aiKey.modelCount`），P2 扩 skill/mcp 后叠加计数。
- **限流列**三态：none→灰字"不限流"；total→"总限流"tag+`TPM x / RPM y` 小字；per_model→"分模型限流"tag+模型项数。
- **i18n**：`col.username/rateLimit` 新增、`col.keyPrefix`(Key前缀→Key)/`col.models`(授权模型→资源)/`col.budget`(预算执行→预算) 改文案（zh-cn/en-us/app.d.ts 三处同步）；`col.name/scenario` 保留（搜索表单与编辑抽屉仍引用）。

## 落地

- `server/model/gateway/response/ai_key.go`：AiKeyView + `OwnerUsername`。
- `server/service/gateway/ai_key.go`：fillOwnerNames 填充登录名。
- `web/src/typings/api/gateway.api.d.ts`：AiKey + `ownerUsername`。
- `web/src/views/_gateway/ai-key/modules/ai-key-list-panel.vue`：列序重排+新列渲染，删 name/scenario/createTime 列与 `ownerTypeLabelKey`。
