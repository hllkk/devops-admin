# AI 网关·使用场景 KeyScenario（场景 Key 分类字典）

- 日期：2026-08-26
- 状态：已实现（后端 build/路由测试 + 前端 typecheck 通过）
- 反向链接：[[ai-gateway-overview]]、[[ai-gateway-identity-home]]

## 需求

四种 Key（个人主/个人场景/部门主/部门场景）中"场景"原本只是场景 Key 的自由文本 name，无法归一（按场景聚合用量无口径）、无法预设、员工申请无从选择。需要"场景"的增删改查。

## 设计决策

- **场景 = 密钥域内的轻量字典实体**（`gateway_key_scenario`：name/description/is_active），不建顶级菜单、不放 sys_dict 通用字典——场景需要被 Key 引用/停用联动/将来按场景聚合，通用字典承载不了。
- **维护入口 = 密钥管理页左侧菜单**（密钥列表/场景管理；2026-08-26 前为双 Tab，后改左右布局见 [[ai-gateway-ai-key-layout]]），对齐 AIHelms AiKeyManage 的多页签域维护思路；管理员主路径是建场景 Key，表单里场景为下拉必选，名称默认带出场景名可改。
- **区分两个概念**：本实体对齐 AIHelms `key_scenarios`（分类标签）；P4 规划的"业务场景模板+资源配置包"（AIHelms `business_scenarios`）是重概念，不在此范围。
- **同名口径**：未软删行内唯一（应用层查重 + 部分唯一索引 `idx_keyscenario_name WHERE deleted_at IS NULL` 兜底）；停用行占名（防同名二义），软删行不占名（可重建）。规避 AIHelms"DB unique + is_active 软删 → 同名永远建不回"的坑。
- **删除策略**：被未软删密钥引用时拒删；场景删除走 gorm 软删（区别于 AIHelms 的置 is_active=false）。
- casbin 零改动：新接口挂 `/gateway/ai-key/scenario/*`，落在密钥管理菜单既有 api_prefix `/gateway/ai-key, /gateway/ai-key/*` 内。

## 落地

- 后端：`model/gateway/key_scenario.go` + AiKey 加 `scenario_id`（逻辑关联）+ `request/key_scenario.go` + AiKeyOperateParams/AiKeySearch 加 scenarioId + AiKeyView 加 scenarioName + `service/gateway/key_scenario.go`（CRUD+ensureScenarioUsable）+ ai_key.go 联动（Create/Update 校验场景启用、fillScenarioNames 批量回填）+ api/router 挂 ai-key 子资源（静态段在 `:id` 前注册）+ AutoMigrate/初始化器双路注册唯一索引。
- 前端：`key-scenario-panel.vue`（搜索+表格+NModal 增删改）+ index.vue 双 Tab + operate-drawer 场景下拉（value 用字符串，`json:",string"` 传输闭环；选场景带出名称，用户手改后不跟随；编辑回填 "0"→null）+ 密钥列表场景列 + i18n 三处同步（aiKey.tabKeys/tabScenario/col.scenario/form.scenario*、keyScenario.* 全节）。
- 注意：scenarioId 前后端以字符串传输（后端 `json:",string"` 出网为 "123"，前端下拉 value 须 String(id)）；场景 Key 编辑时场景必选（存量无场景 Key 引导补选）。

## 待办（后续阶段）

- P2：员工 home 申请场景 Key 时选同一场景字典 + 审批流。
- P3：用量/成本看板按场景维度聚合（scene 已可归一）。
