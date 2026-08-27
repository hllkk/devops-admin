# AI 网关·个人主 Key 生命周期 P0（批量开通/轮换/过期时间）

- 日期：2026-08-27
- 状态：已实现（go build + gateway 单测 + 前端 typecheck 通过）
- 反向链接：[[ai-gateway-overview]]、[[ai-gateway-user-key-cascade]]、[[ai-gateway-identity-home]]

## 需求

对标 AIHelms 差距分析的 P0 三项（管理员创建制下的效率与安全短板）：①批量开通主 Key（逐个手工建在人数上来后不可运营）；②主 Key 轮换（此前等价操作=删旧建新，换行导致历史用量归因断裂、场景引用丢失）；③expires_at/last_used_at 生命周期字段（AIHelms 有、devops-admin 缺）。

## 设计决策

- **轮换=原地换值保归因**：`POST /gateway/ai-key/rotate/:id`。顺序：LiteLLM DeleteKey(旧) → 复用 `syncKeyToLitellm(create=true)` 建新 Key 并原地 Updates 同一行的 `litellm_key_id/key_value(AES)/key_prefix`。**AiKeyId 与 key_alias 均不变**，用量归因(alias 匹配)与场景引用天然连续——这是相对删旧建新的本质优势。旧 Key 在 LiteLLM 删除成功瞬间失效（轮换安全语义：宁可短暂不可用不留旧值）；建新失败则本地行指向已删 Key，返回错误提示重试（此时该 Key 已不可用，重试即恢复）。管理员视角只回 KeyPrefix，新明文仅 owner 经 identity/my 查看。单机模式（litellm 未配置）/未同步 Key（LitellmKeyId 空）显式报错拒绝。
- **批量开通=逐用户独立事务 + 部分成功语义**：`POST /gateway/ai-key/batch`，body `{deptId?, userIds?}`（deptId 优先取部门下全部用户，两者并集）。目标分类纯函数 `classifyBatchTargets(users, existingSet)`（可单测）：停用用户→失败（建了也会被用户级联停用，判定顺序停用优先于已存在）、已有 personal_main→跳过、其余→待创建（map 去重防同批重复 ID）。逐用户调既有 `CreateSceneKey`（name=main、默认公开模型、独立事务单用户失败不中断）。响应走 OkWithDetailed+data 标记（total/created/skipped/failed 明细），部分失败不用错误码（gva 双提示规避惯例）。
- **expires_at 覆盖式更新 + LiteLLM 原生拦截**：`AiKey.ExpiresAt *time.Time`（nil=永不过期），创建/修改直通（`AiKeyOperateParams.ExpiresAt`，nil=改回永不过期）。下发 LiteLLM `expires_at`：CreateKey 直带；UpdateKey 加 `SyncExpiry=true` 强制刷（含 nil→null 清空，同 SyncBudget 模式）。**过期请求由 LiteLLM 原生拒绝，平台侧不另做校验**（投影原则：LiteLLM 是鉴权中心）。
- **last_used_at 回流回填**：`AiKey.LastUsedAt *time.Time`。usage_sync 的 `touchAiKeyLastUsed(ctx, db, logs)`：按 Key 聚合本批最大 started_at，`UPDATE ... WHERE last_used_at IS NULL OR last_used_at < ?` 仅前推不回退（幂等），回流/对账两路径落库后调用；失败仅告警不影响用量主流程。用途：僵尸 Key 治理（列表「最近使用」列）。

## 实现要点

- 路由：`POST gateway/ai-key/batch`、`POST gateway/ai-key/rotate/:id`（静态段注册在 `:id` 前）；均落在密钥管理菜单既有 api_prefix `/gateway/ai-key, /gateway/ai-key/*` 内，**casbin 零改动**（同 KeyScenario 结论）。
- **坑(2026-08-27 实测)**：请求体内传 ID 列表不能用裸 `[]int64`——前端雪花 id 按项目约定走 `CommonType.IdType=string|number`(实际字符串)，`[]int64` 绑定报 `cannot unmarshal string into ... of type int64`；且 Go `json:",string"` tag 对 slice 无效。已给 `common.Int64StringSlice` 补 `UnmarshalJSON`(字符串/数字/混合元素兼容，复用 Int64String 解析)并用于 `AiKeyBatchCreateParams.UserIds`；`DeptId *int64` 的 `",string"` tag 对指针有效无需处理。单测覆盖(common/basetypes_test.go + ai_key_batch_test.go 绑定测试)。
- 前端：列表「批量开通」按钮（TableHeaderOperation #prefix 插槽）+ `ai-key-batch-modal.vue`（按部门 DeptTreeSelect/按用户 NSelect multiple，提交后弹窗内展示统计+失败明细，关闭时才刷新）；行操作「轮换」（popconfirm 提示旧 Key 立即失效）；operate-drawer 过期时间 NDatePicker（`v-model:formatted-value` + `value-format="yyyy-MM-dd'T'HH:mm:ssXXX"` RFC3339 闭环）；列表过期列（已过期红/7 天内黄/永不灰）+ 最近使用列；home 身份卡 meta 区加「有效期」。i18n 三处同步（zh-cn/en-us/app.d.ts）。
- 单测：`ai_key_batch_test.go` 3 场景（混合分类/停用优先于已存在/同批重复 ID 去重）。

## 待办（关联）

- P1：被动停用标记位（用户重新启用误恢复管理员手动停的 Key，见 [[ai-gateway-user-key-cascade]] 已知取舍）。
- P1：改名级联加固（巡检对账推广到场景 Key）。
- P2：批量建场景 Key（名称模板 `{username}`）/"复制自主 Key"模板/员工申请+审批（审批落主 Key，对齐 AIHelms `_grant_resource`）。
