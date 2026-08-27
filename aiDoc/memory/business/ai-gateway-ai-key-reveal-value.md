# AI 网关·密钥列表查看/复制完整 Key（value/:id 按需明文）

- 日期：2026-08-27
- 状态：已实现（go build/test、路由注册测试、typecheck 通过）
- 反向链接：[[ai-gateway-ai-key-list-columns]]、[[ai-gateway-user-key-cascade]]

## 需求

密钥列表 Key 列原本只显 `keyPrefix`（前 8 位+****）且点击仅复制前缀。管理员/超管悬停该列时展示「眼睛 + 复制」两个图标：点眼睛切换显示完整 Key 明文，点复制把完整 Key 复制给用户转发。

## 设计决策

- **不打破"列表/详情不返回 KeyValue"的默认安全边界**：明文经独立接口 `GET /gateway/ai-key/value/:id` 按需解密返回（`AiKeyRevealView{keyValue}`），单次单条，不随列表批量出网；列表/详情/轮换的 KeyPrefix 语义不变。
- **权限后端硬校验**：service 层 `keyRevealAllowed`——JWT `claims.SuperAdmin` 直接放行；其余查启用角色（`Preload("Roles","status=0")` 与 getUserInfo 口径一致），任一角色 SuperAdmin 或 RoleKey=admin 即视为管理员。未过校验返回业务错误，不依赖 casbin 兜底。
- **审计**：挂 PrivateGroup（OperationRecord 在 JWT 后），查看明文行为自动落操作日志（谁查看了谁的 Key）。
- **单机模式边界**：`litellm_key_id` 为空（未同步 LiteLLM）的行返回"无可用明文"错误，与轮换同口径。
- **前端瞬态行字段**：`AiKeyRow = AiKey & { fullKeyValue?; keyRevealed? }`，data 深层 reactive 直接改行对象触发单元格重渲；明文只缓存在内存行内，翻页/刷新自然重置，不落任何持久层。
- **交互**：默认仅前缀；图标 `hidden group-hover:flex`（触屏 `lt-sm:flex` 恒显）；眼睛按 revealed 切换 visibility/visibility-off 图标与 tooltip；明文首次拉取后行内缓存，复制复用 `handleCopy`（自带成功 toast）。

## 落地

- `server/model/gateway/response/ai_key.go`：`AiKeyRevealView`。
- `server/service/gateway/ai_key.go`：`RevealAiKeyValue` + `keyRevealAllowed`（新增 systemReq import）。
- `server/api/v1/gateway/ai_key.go`：`RevealAiKey` handler（swagger 注释）。
- `server/router/gateway/ai_key.go`：`g.GET("value/:id", ...)`（对齐 `rotate/:id` 静态前缀风格，`TestAiKeyRouterRegistration` 验证不 panic）。
- `web/src/service/api/gateway/ai-key.ts`：`fetchRevealAiKeyValue`。
- `web/src/typings/api/gateway.api.d.ts`：`Api.Gateway.AiKeyReveal`。
- `web/src/views/_gateway/ai-key/modules/ai-key-list-panel.vue`：keyPrefix 列 hover 图标改造 + `ensureFullKey/toggleKeyReveal/copyFullKey`。
- i18n：`page.gateway.aiKey.viewKey/hideKey/copyKey`（zh-cn/en-us/app.d.ts 三处同步），列 minWidth 180→240。

## 复制不成功排查（2026-08-27 追记）

现象：点复制无任何反应、无 toast。后端链路逐一验证正常（value/:id 路由已在运行进程、`gateway_ai_key.litellm_key_id/key_value` 非空、super/admin 角色数据正常），根因在 `web/src/utils/copy.ts`：

1. dev 经 `http://<IP>:9527` 访问（远程开发机，非 localhost 非 https）→ 非安全上下文 → `navigator.clipboard === undefined`，走 execCommand 降级分支；
2. 降级分支引用 `document.getElementById('tokenDetailInput')` —— 该 ID **全库无任何组件定义**（上游拷来的死引用），`range.selectNode(null)` 抛 TypeError，未捕获 → 静默失败；
3. `if (!isSupported)` 判断的是 Ref 对象本身（恒 truthy），"不支持剪贴板"保护形同虚设。

修复：`useClipboard({ legacy: true })`（vueuse 内部临时 textarea 自动降级，不依赖页面元素）+ `isSupported.value` 正确判断 + try/catch 失败兜底提示，删除死代码分支。此修复惠及全项目所有 `handleCopy` 调用点。
