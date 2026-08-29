# AI 网关 · 批量建场景 Key + 复制主 Key 模板(P2 收尾)

日期：2026-08-28
状态：已实现(后端+前端，待用户运行时验证)
反向链接：[[ai-gateway-mainkey-p0-lifecycle]]、[[ai-gateway-key-scenario]]

## 需求

[[ai-gateway-mainkey-p0-lifecycle]] 遗留 P2 待办：批量开通主 Key 已落地，但场景 Key
仍只能逐个手工建(给一个团队/部门配同款场景 Key 时不可运营)；「复制主 Key 模板」=
以某用户主 Key 的配置(授权/预算/限流)为模板批量复制。参照 AIHelms
batch_create_keys(`{username}`/`{display_name}` 模板替换+循环 create_key)。

## 设计决策

- **后端批量接口**：`POST /gateway/ai-key/batch-scene`，body=AiKeyBatchSceneCreateParams
  (deptId?/userIds? 并集目标 + nameTemplate 必填 + 完整资源配置 models/mcps/skills/
  预算三件/限流/expiresAt)。逐用户调既有 CreateSceneKey(personal_scene)——**复用创建
  管线**(校验/加密/LiteLLM 同步全保留)，渲染名=renderNameTemplate(`{username}`→登录名/
  `{nickname}`→昵称，昵称空回落登录名)。停用用户→失败明细(建了也会被级联停用)；同名冲突
  由 CreateSceneKey 查重报错计入失败明细(**无跳过语义**，与批量开通主 Key 的"已有跳过"
  不同——场景 Key 无唯一性前提)；逐用户独立事务部分成功，复用 BatchCreateMainKeysResult
  (Skipped 恒 0)。casbin 零改动(落密钥管理既有 api_prefix)。
- **复制主 Key 模板=纯前端预填**，后端不加 templateKeyId 参数：密钥列表主 Key 行操作
  「以此为模板建 Key」(copyTemplate 图标)→ 打开批量弹窗并预填该 Key 的
  models/mcps/skills/budgetLimit/budgetHardLimit/budgetDuration/rateLimitMode/tpm/rpm/
  expiresAt(用户可改可再删)。管理员可见最终表单，语义透明。
- 批量弹窗(ai-key-batch-modal)升级双模式：顶部「批量开通个人主 Key / 批量建场景 Key」
  单选(复制模板进入时锁定 scene 模式)；目标选择(部门/用户)两模式共用；scene 模式表单=
  名称模板+场景下拉(可选)+授权三多选+预算三件+限流+过期。

## 实现要点

- renderNameTemplate 纯函数 6 场景单测(占位替换/无占位原样/空模板)；params 绑定单测
  (UserIds 字符串元素+DeptId/ScenarioId 指针 `,string`+skills 字符串数组+expiresAt null)。
- resolveBatchTargets 补取 user_name 列(模板渲染需要，主 Key 批量不受影响)。
- typings：AiKeyBatchSceneCreateParams(RecordNullable)。

## 验证

- go build/vet/test 全过；typecheck/eslint 0 错误；i18n 三处同步
  (batchScene.* 8 key+batchSceneCreate/copyTemplate)。

## 待办(关联)

- P2 至此全部完成。P3 成本效能与预算管控(见 [[ai-gateway-overview]] 分期)。
