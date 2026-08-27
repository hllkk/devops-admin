# AI 网关·模型改名级联 Key 授权（借鉴 AIHelms P1 补强）

- 日期：2026-08-26
- 状态：已实现（go build/vet/全量测试 + 4 个纯函数单测通过）
- 反向链接：[[ai-gateway-user-key-cascade]]、[[ai-gateway-overview]]

## 需求

模型 `model_key` 改名（UpdateModel）此前只级联重建部署投影（cascadeRebuildModelDeployments），Key 侧 `models`/`model_budgets`/`model_limits` 三个 JSONB 仍存旧名——改名后用户调新名被 LiteLLM 拒（key 无该模型授权）、按模型预算/限流随旧名漂移。对标 AIHelms `_sync_keys_after_model_rename`（model_service.py:183-215）补级联。

## 设计决策

- **落位**：`cascadeRenameKeyModels(ctx, db, oldKey, newKey)` + 纯函数 `renameKeyReferences`（service/gateway/ai_key.go），挂在 UpdateModel 部署级联旁、仅 `newKey != m.ModelKey` 时触发；与部署级联同口径**尽力而为**（单个失败仅 warning 不阻断改名）。
- **全量拉取内存过滤**：AiKey 是小表（主Key+场景Key 数量级），不做 JSON 查询条件（跨库方言），Find 全表后在 Go 内按三处 JSONB 过滤。
- **DB 存原始 modelKey**：`(Anthropic)` 变体仅在 syncKeyToLitellm 下发时扩展，改名只需 oldKey→newKey 一对映射，同步时自动带出新名变体。
- **值保值改名**：models 列表元素替换（保序）、budgets/limits map 键替换值不动；`renameKeyReferences` 返回 changed 标志，未引用的 Key 跳过（零写入）。
- **边界**：只处理主 Key 与场景 Key 一视同仁（都是 DB 引用）；取消发布**不回收**授权（维持既有边界，loadMainKey 自愈按 publicModelKeys 差集不会加回取消发布的模型）。
- **测试**：`ai_key_cascade_test.go` 四场景——全引用/未引用/仅 models 引用（nil map 原样返回）/仅 budgets·limits 引用。

## 关联

- 用户级联（禁用/删除）：[[ai-gateway-user-key-cascade]]
- P2 待办（批量建 Key/复制主 Key 模板/expires_at+last_used_at）见 [[ai-gateway-user-key-cascade]] 待办节
