# AI 网关·模型发布可见性消费闭环（用户侧展示 + 定向自动授权）

- 日期：2026-08-27
- 状态：已实现（go build/vet/test + vue-tsc + eslint 通过）
- 反向链接：[[ai-gateway-model-publish]]、[[ai-gateway-identity-whitelist]]、[[ai-gateway-model-rename-cascade]]

## 需求

模型发布三档可见性（all/selected/user）已落库+投影表，但**消费侧整条链路缺失**：发布后用户侧无任何展示（`GET /gateway/model/active` 是无前端的孤儿接口且无可见性过滤），selected/user 档发布后密钥资源数永远不 +1（原设计仅 all 档免审批自动授权，「订阅入口后续落地」从未落地）。对照 AIHelms 深度分析后确认：**AIHelms 的可见性表同样只写不读**（`_sync_published_model_to_main_keys` 不检查 visibility_type，selected 档发布也进所有主 Key，越权全员可见），其用户可用模型的事实源是主 Key `models` 数组 + `resource_applications` 审批流闭环——本项目取其「授权即资源数/模型广场交互/审批流」骨架，规避其「可见性不过滤/部门物化展开/授权只加不收」三坑。

## 已实现（后端）

1. **`visibleModelScope(db, userId, deptId)`**（service/gateway/model.go 内部工具）：三档可见过滤条件——all 直通 ∨ selected 命中部门投影（用户归属部门=**主部门 dept_id ∪ 多部门 sys_user_departments**，与 sys_notice 部门展开口径对称）∨ user 命中用户投影。投影表带软删基座，手写 EXISTS 子查询显式 `deleted_at IS NULL`。
2. **`GetActiveModels(ctx, userId)`** 改造：加 userId 参数，查可见模型；anthropic 联查抽 `annotateAnthropic` 共用（三处逐行重复消重）。`GET /gateway/model/active` 入 `rbacWhitelistPrivate` 白名单（照 identity/my 先例）+ casbin 单测双向断言（model/list、model/:id、model/publish 不得误放行）。**2026-08-27 修正：初版给超管加了 userId=0 旁路（看全部），导致"指定用户可见"发布后管理员在模型广场仍见全部、且与 GetMyIdentity（无旁路）两入口口径不一致——已移除旁路，所有用户含超管统一按可见性过滤，超管看全部走管理端 GetModelList。**
3. **`PublishModel` 三档定向自动授权**：免审批+有 modelKey 时按 `mainKeyScopeOf(visibility, deptIds, userIds)` 构造目标集合——all=全部活跃主 Key；selected=可见部门成员（sys_users 主部门∪多部门，排除软删）的 personal_main + 可见部门的 dept_main；user=指定用户的 personal_main。`syncPublicModelToMainKeys` 泛化为 `syncModelToMainKeys(ctx, tx, modelKey, scope)`。需审批模型仍不自动授权（申请流 P2）。
4. **自愈差集源扩展**：`publicModelKeys`（仅 all 档）→ `visibleModelKeys(db, userId, deptId)`（对 owner 可见的免审批模型；个人传 (userId,主部门) 三档全生效，部门传 (0,deptId) 仅 all/selected——userId=0 不命中 user 投影天然正确）。两处消费同步：`loadMainKey` 自愈、`CreateSceneKey` 建主 Key 默认授权（部门主 Key 按部门可见）。新加 `userDeptIdOf`。
5. **`GetMyIdentity`**：availableModels 从 `GetAvailableModels`（全量）切 `GetMyVisibleModels`（可见过滤，经 `listModelsAsAvailable(scope)` 与全量版共用组装）；`MyIdentityView` 加 `GatewayUrl`（`OPS_CONFIG.Litellm.PublicURL` 下发，客户端 Base URL）。`GetAvailableModels` 保持管理员全量（建 Key 下拉在用），注释标注勿混。

## 已实现（前端）

6. **home 我的资源卡**（home/index.vue）：`identity` ref 全量保存（未开通 opened=false 也带可见模型），`mainKey` 变 computed；「可用模型」卡从渲染 `mainKey.models`（已授权）升级为 `identity.availableModels`（可见模型）——已授权 primary 实色 / 未授权灰虚线（NTooltip 提示未授权/需审批），计数改「已授权 X / 可见 Y」；未开通用户也可见可开通的模型（MCP/Skill 占位卡仍需开通后显示）。
7. **模型广场页**（views/_gateway/square/index.vue，新）：卡片网格（provider logo/名称/modelKey/类别/能力≤3/描述两行截断）+ 本地搜索（名称/路由名）+ 已授权✓/未授权/需审批状态标 + 「查看接入」弹窗（modelKey + modelKeyAnthropic + Base URL gatewayUrl + 主 Key 明文掩码/显隐/复制）；未开通时 NAlert 提示+按钮禁用。菜单：sys_menu.go 顶层 C 菜单 `route.square`（path=square，Component=_gateway/square/index，ApiPrefix=/gateway/model/active 仅关联语义）+ sys_role_menu.go user 角色授权（与 home 同「全员基础页」模式，**仅新库生效，已有库需手动补菜单+授权**）。service 加 `fetchGetActiveModels`，typings 加 `ActiveModel`/`MyIdentity.gatewayUrl`，i18n 三处同步。
8. **发布设置弹窗对齐 AIHelms**（model-publish-dialog.vue，2026-08-27）：标题「发布设置」+ 副标题说明；「是否发布」开关改「发布到用户端」；可见范围（三档）与「领用前需要审批」（requiresApprovalTip 标注审批流即将上线，占位，申请流 P2）仅发布开启时显示；autoGrantTip 文案更新为三档自动授权口径（免审批+已设 modelKey → 授权到可见范围内用户的主 Key）。

## 设计决策记录

- **「授权即资源数」保持极简口径**：不建 key↔model 关联表、不加 count 字段，`models` JSONB 长度即资源数（AIHelms 同款，本项目密钥列表列已是该口径）。
- **不回收边界维持**：取消发布/收窄可见范围/用户调部门，已授权 Key 里的 modelKey 不回收（与既有决策一致，宽容语义）；自愈差集只加不删。
- **dept_main 无惰性自愈**：与原设计一致，靠发布时 syncModelToMainKeys 追加 + 建 Key 时默认授权。
- **部门匹配语义**：精确匹配（不做子部门递归），用户侧=主部门∪多部门连接表，与 sys_notice 展开口径对称。
- **超管视角**：用户端接口（model/active、identity）对所有用户含超管一律按可见性过滤，不做管理视角旁路（曾引入超管旁路后因口径混乱移除，2026-08-27）；超管看全部走管理端 GetModelList。
- **借鉴与规避**（AIHelms）：取模型广场交互（isOwned 前端交集判定）、审批流蓝图（resource_applications，P2 落地时参照）；规避 visibility 不参与过滤、部门投影物化展开成用户投影（僵尸数据）、审批授权无主 Key 静默 skip（本项目管理员创建制下高频，P2 必须显式处理）。

## 关联

- 发布三档与投影表：[[ai-gateway-model-publish]]
- identity 白名单先例：[[ai-gateway-identity-whitelist]]
- 改名级联（models 键改写）：[[ai-gateway-model-rename-cascade]]
