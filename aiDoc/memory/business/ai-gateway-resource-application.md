# AI 网关·P2 资源申请审批（模型订阅审批闭环）

- 日期：2026-08-28
- 状态：已实现（go build/vet/test + typecheck/eslint 全过；菜单种子仅新库生效，dev 由用户重新初始化覆盖）
- 反向链接：[[ai-gateway-model-visibility-consumption]]、[[ai-gateway-model-publish]]、[[ai-gateway-mainkey-p0-lifecycle]]

## 需求

模型发布"领用前需审批"档（requiresApproval）自 P1 发布配置落地以来只有开关与前端占位（"审批流即将上线"文案、广场"需审批"状态标），申请/审批闭环完全缺失。P2·AI 市场从本功能切入：它是 MCP/Skill 申请的公共底座（表结构 resource_type 预留），蓝本 AIHelms `resource_applications`，并规避其四坑。

## 后端实现

1. **表 `gateway_resource_application`**（model/gateway/resource_application.go）：OPS_AUDIT_MODEL+雪花；`(user_id, resource_type, resource_id)` 复合唯一索引（gorm tag，**行永不软删**——rejected 再申请=复用原行重置，规避软删行占唯一索引挡重新申请，ModelVisibility 投影表同款考量）；status 一次性单向 `pending → approved/rejected`，无撤回无超时（AIHelms celery 死字段证明超时没人用）；resource_type 枚举 model/mcp/skill（P2 只实现 model）。
2. **Service**（service/gateway/resource_application.go）：
   - `Create` 强校验（规避 AIHelms"只查存在性"坑）：类型合法 → 模型存在+启用+已发布+有路由名+**经 visibleModelScope 对申请人可见** → `requiresApproval=true`（免审批模型拒绝申请"可直接使用"）→ 防重分流（pending 拒"耐心等待"/approved 拒"已拥有"/rejected 复用原行重置清审批字段，条件更新 RowsAffected=0 时递归重读分流防并发）。
   - `Approve/Reject` 共用 `review`：pending 条件更新防并发双审；approve 侧校验模型仍可授（删除/下架/未发布报错让管理员驳回——批准也无法授权），事务内状态+授权原子提交：自建 scope 锁定该用户 personal_main → 复用 `syncModelToMainKeys`（与发布自动授权同管线）；LiteLLM 推送失败 warning 不回滚（每日 ResyncAiKeys 兜底，规避 AIHelms 外呼无容错坑）。reject 仅留痕无回收（驳回时未授权过）。
   - `BatchReview` 逐条串行复用 review（每条独立事务，失败不中断），返回 success/failed 明细。
   - `fillApplicationViews` 每页三次 IN 查询回填 userName/reviewerName/resourceName/resourceKey 防 N+1。
3. **主 Key 自愈差集源扩展**（service/gateway/ai_key.go）——规避 AIHelms"无主 Key 静默 skip 仍 approved"坑（本项目管理员创建制下主 Key 不存在是高频场景）：
   - 新增 `approvedApplicationModelKeys(db, userId)`：JOIN gateway_model（须仍启用+已发布+有路由名）取该用户 approved 申请的 modelKey；下架/删除由发布对齐回收、自愈不回加；**重新发布后自愈补回**（申请仍 approved，语义=批准继续有效）。
   - `loadMainKey` 自愈与 `CreateSceneKey` 主 Key 默认授权（仅 personal_main，dept_main 不涉及用户审批）的差集源改为 `mergeMissingKeys(current, visibleModelKeys(...), approvedApplicationModelKeys(...))`——抽出的纯函数（4 场景单测）统一两处合并口径。主 Key 不存在/停用时授权由后建主 Key 或身份访问自愈补上，批准永不静默丢失。
4. **接口与鉴权**：用户侧 `POST /gateway/application/apply`、`GET /gateway/application/my` 入 rbacWhitelistPrivate（**路径刻意避开 `/gateway/application` 裸前缀**——白名单 HasPrefix 匹配会误放行 list/approve 等管理接口，apply/my 为独立 path 无包含关系）；管理端 list/approve/reject/batch-approve/batch-reject 走菜单 ApiPrefix → casbin（ApiPrefix 枚举具体 path 不用 `/*` 通配）。casbin 白名单单测双向断言（middleware/casbin_rbac_test.go）。
5. **通知**：审批结果定向通知放 **api 层**（`notifyApplicationReviewed`，service/system 已反向 import service/gateway 成环风险故不下沉），复用 SysNotice+SSE：NoticeType=通知/TargetType=users/单人投递，失败仅告警不影响审批结果；service 返回 `ReviewNotification{userId,resourceName,approved,reviewNotes}` 供 api 层组装。
6. **菜单** route.application（sys_menu.go OrderNum 12，`_gateway/application/index`，icon lucide:stamp，照 route.usage 管理页模式 user 角色不授，super/admin 全菜单循环自动获得）；MigrateTable 清单补 ResourceApplication（source/gateway/provider_prefix.go）。

## 前端实现

- **广场页**（_gateway/square/index.vue）：需审批+未授权+未申请卡片加「申请订阅」按钮+理由弹窗（必填/500 字）；已申请态显示「已申请，待审批」tag（appliedIds 本地集合 + onMounted 并行拉 `GET my?status=pending` 回填，刷新状态保持）。
- **home**：`我的申请` 占位卡切真实列表（最近 10 条：资源名+类型 tag+状态 tag+申请时间+审批意见；空态加「去模型广场申请」按钮 routerPushByKey('square')）；需审批 tooltip 文案改「需审批，可在模型广场申请订阅」；发布设置弹窗 requiresApprovalTip 的"审批流即将上线"文案改「在资源申请页处理」。
- **管理审批页**（_gateway/application/index.vue 新建）：状态筛选默认 pending + 类型/申请人（fetchGetUserSelect 懒加载）筛选；表格勾选批量（批量通过/驳回按钮带计数）+ 单条行内通过/驳回；审批弹窗单条/批量共用（mode+targets+可选意见），批量结果「成功 N 失败 M」警告汇总、单条 approve 带同步警告提示兜底文案。
- **契约**：typings/api/gateway.api.d.ts 加 ApplicationItem/SearchParams/CreateParams/ReviewResult/BatchReviewResult；service/api/gateway/application.ts 7 个 fetch（批量 timeout:0）；i18n 三处同步（zh-cn/en-us/app.d.ts：square 申请键 + application 管理页段 + home.identity 申请键）；elegant 路由四件由 vite 插件监听自动生成。

## 设计决策记录

- **rejected 重置复用而非历史多条**：一行一资源一用户，审批历史靠操作日志与通知承载，换取干净的复合唯一索引（并发防重有 DB 兜底）。
- **审批授权域=仅 personal_main**：与发布自动授权"系统自动授权域"口径一致，场景 Key 手工授权不动；停用主 Key 授权照写（DB 层），LiteLLM 侧停用 Key max_budget=0 卡死安全。
- **不回收边界维持**：取消发布/删除模型时发布对齐全量回收（含审批授权）；重新发布后自愈补回 approved 的——宽容语义与 P1 一致，管理员若要作废某用户的审批授权需手工改其主 Key（AIHelms 同样无此能力）。
- **不做**：审批超时过期、员工撤回、多级审批、MCP/Skill 类型落地（实体未建，仅表结构/枚举预留）。

## 待办（关联）

- MCP Server 管理 / Skill 管理（P2 后续切片，本表 resource_type 直接扩展）。
- 批量建场景 Key / 复制主 Key 模板（[[ai-gateway-user-key-cascade]] 待办）。
- 浏览器点触验证（申请→通知→审批→授权→广场状态流转全链路）。
