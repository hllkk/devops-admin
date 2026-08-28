# AI 网关·P1 收尾加固（被动停用标记 + 密钥 resync 兜底）

- 日期：2026-08-27
- 状态：已实现（go build/vet/test 全过）
- 反向链接：[[ai-gateway-user-key-cascade]]、[[ai-gateway-mainkey-p0-lifecycle]]、[[ai-gateway-model-rename-cascade]]

## 需求

P1 两个已知取舍/待办收口：①用户重新启用时全量恢复名下 Key，会把管理员手动停用的 Key 一并激活（授权隐患）；②模型改名级联 `cascadeRenameKeyModels` 已覆盖场景 Key（全表内存过滤），但其 `syncKeyToLitellm` 失败后场景 Key 无补偿路径——主 Key 有 loadMainKey 自愈（身份访问驱动）顺带重推，场景 Key 漂移即永久。

## 设计决策

### 被动停用标记位（A1）

- `AiKey.DisabledByCascade bool`（default:false，AutoMigrate 自动加列）。语义=**用户生命周期级联停用**专属标记；管理员手动启停与超限停用（enforceBudgetHardLimit）均不打标。
- 筛选纯函数 `filterCascadeKeys(keys, active)`（ai_key_cascade_test.go 3 场景单测）：停用=仅选 `is_active=true`（打标），恢复=仅选 `disabled_by_cascade=true`（清标恢复）。两向都不碰手动停/超限停的 Key。
- `SyncUserKeysActive` 查 owner 全量后经纯函数筛，逐个 Updates `is_active + disabled_by_cascade + update_by` 并同步 LiteLLM；三钩子（UpdateStatus/Update/Delete）签名零改动。
- `UpdateAiKey` 显式传 `IsActive` 时清标（管理员显式意志接管，此后用户重新启用不再联动该 Key）。
- 边界确认：超限停用 Key 若被用户级联恢复误激活，聚合任务每 5 分钟扫 `is_active=true` 超限 Key 会再次停用，闭环安全。
- 存量库行为变化：既有"已被动停"的 Key 无标记→用户重新启用时不恢复（宁少恢复不多恢复，管理员可手动启用）。

### 密钥 resync 兜底（A2）

- `AiKeyService.ResyncAllKeys(ctx)`：全量未软删 Key，`litellm_key_id != ''` 的逐个 `syncKeyToLitellm`（create=false 全量下发，幂等）。返回复用凭证域 `ResyncResult{total/pushed/skipped/failed}`。
- **与凭证 resync 的差异**：凭证侧做投影比对（ListCredentials 远端真值可靠）；Key 侧无条件重推——LiteLLM `/key/info` 有显示缓存（slice4 实测），远端无可靠真值可比。
- 入口三路：`POST /gateway/ai-key/resync`（密钥列表「重新同步」按钮，casbin 零改动落既有 api_prefix）、定时任务 `ResyncAiKeys`（每日 03:37，timer.go 注册 + timed_task.go 种子）、兜住改名级联/授权对齐/任何 DB 成功但 LiteLLM 失败的漂移。

## 待办（关联）

- ProviderBalance 录入百炼 AK/SK 实测（[[ai-gateway-provider-balance]]）。
- 复制主 Key 模板/批量建场景 Key（P2，[[ai-gateway-user-key-cascade]] 待办节）。
