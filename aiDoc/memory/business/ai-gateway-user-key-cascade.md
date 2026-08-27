# AI 网关·用户生命周期级联启停 AiKey（借鉴 AIHelms P0 补强）

- 日期：2026-08-26
- 状态：已实现（go build/vet/全量测试通过）
- 反向链接：[[ai-gateway-overview]]、[[ai-gateway-identity-home]]

## 需求

用户被禁用/删除后，其名下 AiKey（主 Key+场景 Key）在 LiteLLM 侧仍处启用态——明文 Key 在外部即可继续直连网关调用，仅平台登录被拦。对标 AIHelms `sync_user_keys_active`（`ai_key_service.py:440-462`）补级联闭环，属安全修复（P0）。

## 设计决策

- **级联方法落 gateway 侧**：`AiKeyService.SyncUserKeysActive(ctx, userId, updateBy, active) []string`（service/gateway/ai_key.go）——查 `owner_type=user AND owner_id` 全部未软删 Key，逐个本地置 is_active + 复用 `syncKeyToLitellm`（false 分支，is_active=false → LiteLLM max_budget=0 停用语义现成）。单个失败仅 warning 不中断，返回告警列表。
- **三个钩子**（service/system/sys_user_manage.go）：`UpdateStatus`（changeStatus 接口，'0'/'1' 有效值触发）、`Update`（编辑表单也带 status——事务前读原值，**仅 status 实际变化才级联**，避免全量同步误恢复管理员手动停的 Key）、`Delete`（物理删用户后事务外级联停用；Key 保留承载历史用量归因，不删）。Delete 签名加 updateBy（api 层传 GetUserID）。
- **启用=全量恢复（已知取舍，对齐 AIHelms）**：用户禁用→启用后，此前管理员手动停用的场景 Key 会被一并激活；未做"被动停用"标记位，概率低且可再手动停。
- **级联失败不回滚主操作**：返回 error 提示"N 个密钥级联失败，可重试"（UpdateStatus/Update 幂等，重试即补同步）；Delete 失败提示到密钥管理手动停用。
- **依赖方向**：service/system → service/gateway 首次跨组引用（gateway service 不 import 任何 service 包，无循环）；实例化方式 `(&gateway.AiKeyService{}).SyncUserKeysActive(...)`（空 struct 无状态）。

## 待办（关联）

- P1 模型改名级联已于 2026-08-26 落地，见 [[ai-gateway-model-rename-cascade]]。
- P2 待办：批量建场景 Key（名称模板 `{username}`）/按 user_ids 批量调主 Key；建 Key 表单"复制自主 Key"模板；expires_at/last_used_at 字段。
