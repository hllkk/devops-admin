# AI 网关·P3 成本分析页

- 日期：2026-08-30
- 状态：已实现（build/vet/test+typecheck/eslint 全过，待运行时验证）
- 关联：[[ai-gateway-overview]]（P3 分期）、[[ai-gateway-usage-log-page]]（调用日志页=跳转目的地）、[[ai-gateway-ai-audit-menu]]（AI审计目录）

## 需求

P3「成本效能与预算管控」第一件：多维成本分析页（管理员/决策层视角）——KPI 环比、趋势、按人/部门/模型/Key/供应商/日期六维下钻、部门成员展开、一键跳调用日志。数据全部来自 P1 已有的 `gateway_cost_summary_daily` 聚合缓存（T+5min），不新增表、不改成本口径。

## 用户决策

- **部门归因口径（拍板）**：部门 Key 的消耗归部门（Key 归属优先），个人 Key 归个人主部门（`sys_users.dept_id`）；两者皆无归「未分配」。
- **本期裁剪**：不含 MCP 维度（MCP 调用日志回流是 P3 另立件）、不含导出（效能报告统一做异步导出）、不含项目维度（无此实体）、明细行不做环比（只有 KPI 卡做）。

## 实现

### 后端（`/gateway/cost/*`，CostAnalysisService）

- `GET overview`：KPI（内部/外部成本、结算差额、日均内部成本、内外环比=等长上一期对比、请求/token 量）+ 按日趋势（内/外双线），随筛选联动。
- `GET detail`：六维聚合（department/user/model/aiKey/provider/date），服务端分页（`LimitOffset` 上限100）+ 排序白名单（internal/external/requests/tokens → 聚合别名）；COUNT(DISTINCT 分组表达式) 取组数作 total；label 批量 IN 回填（部门名/昵称/Key 名，防 N+1）。
- `GET detail/scope-users?deptId=`：部门行下钻成员成本，user_id=0 合并为「部门Key/未归因」行。
- **部门读时归因**（不写聚合表，人员调岗不污染历史成本）：
  - 锚点 SQL：`CASE WHEN k.owner_type='dept' THEN k.owner_id ELSE COALESCE(u.dept_id,0) END`，JOIN `sys_users u`(主部门)+`gateway_ai_key k`，`LEFT JOIN` 不丢未归因行；软删 Key 不筛 deleted_at（成本是历史事实，仍按其归属归因）。
  - 部门列表行与下钻同用直挂口径 → 部门行 = 子行和（对账一致）；「含子部门」视角走部门筛选（`expandDeptSubtree` 内存 BFS 子树展开后锚点 IN 下推）。
  - `expandDeptSubtree` 自写而不复用 `system.DataScopeService.ExpandDeptIDs`：service/system 已引 service/gateway（用户级联），反向引用会 import 环。
- 时间：业务日闭区间（`summary_date` 已按 Asia/Shanghai 切桶），缺省本月；环比=等长上一期（`prevCostRange`），上期 0 时环比给 0。
- 纯函数单测：normalizeCostRange/prevCostRange/costChange/维度与排序白名单/分组表达式（5 组）。

### 前端（`_gateway/ai-audit/cost/`）

- `cost-search`：预设时间 Tab（今天/昨天/本月/近7天/近30天，custom=自定义无选中态）+ daterange + 部门树(NTreeSelect)/用户(UserSelect)/模型/供应商筛选。
- `cost-overview`：KPI 4 卡（环比红涨绿跌、口径 tooltip）+ 趋势图（useEcharts 内实线/外虚线双线）。
- `cost-top-users`：人员 Top10（复用 detail dimension=user）。
- `cost-detail-table`：NTabs 六维 + 排序 NRadioGroup + 手写服务端分页 + 部门维行内展开（懒加载缓存 Map）+ 行「日志」按钮 `router.push('/gateway/ai-audit/usage', query)`（维度行收敛为单值筛选，date 维收敛为单日）。
- usage 页补 route query 预填（userId/aiKeyId/model/provider/startDate+endDate 业务日→本地零点/末点 RFC3339），usage-search 加 `initialDateRange` prop 一次性注入。
- i18n：`page.gateway.cost.*`（preset/search/kpi/trend/top/detail）+ `route.ai-audit_cost` 三处同步（zh-cn/en-us/app.d.ts）。

### 菜单与权限

- `route.ai-audit_cost` 挂 AI审计目录 OrderNum 3，ApiPrefix `/gateway/cost, /gateway/cost/*`，Component `_gateway/ai-audit/cost/index`；user 角色不授（管理员视角，防普通用户查全公司人员级成本——规避 AIHelms 读端点零权限坑）。仅新库生效，dev 重新初始化。

## 规避的 AIHelms 坑（调研确认）

读端点零权限｜明细全量返回+SQL LIMIT 500 静默截断｜多部门用户维度视图重复计数（我们按主部门单值归因）｜INNER JOIN 丢无部门用户（LEFT JOIN+未分配）｜部门行数值≠下钻子和（同直挂口径）｜时区切天依赖会话时区（聚合层已显式 Asia/Shanghai）｜环比只算内部（内外都给）｜前端串行三连发（并行刷新）｜模型归因 6 条件 OR LATERAL 模糊反查（直接按聚合表 model 分组）。

## 待办（后续）

- 运行时验证（dev 重新初始化后菜单生效）。
- MCP 维度 tab + overview 拆 LLM/MCP 构成：待 MCP 调用日志回流（P3 另立件）落地后加。
- 导出：待效能报告的异步导出任务统一做。
