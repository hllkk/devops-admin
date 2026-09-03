# AI 网关·P3 覆盖率/采用度

- 日期：2026-09-02
- 状态：已实现（go build/vet/test + typecheck 通过，待运行时验证）
- 关联：[[ai-gateway-cost-analysis]]（复用其筛选/归因/环比基建）、[[ai-gateway-plan-litellm-base]]（P3 定义）

## 需求

P3 收尾三件之一：覆盖率/采用度——部门覆盖率/人员激活率/模型分布/日均调用量/人均 token。

## 实现口径（集中定义，规避 AIHelms 同概念三处不一致坑）

- **激活用户** = 期内有任意 LLM（聚合表）或 MCP（日志表）调用且 user_id<>0 的去重用户；Skill 平台内调用不计入
- **分母** = sys_users.status='0' 未软删的全部用户（决策层全员视角，含从未使用者）
- **部门归属** = 直挂口径（复用 costDeptAnchor：部门Key归部门/个人Key归个人主部门）；成员分母按 sys_users.dept_id 单值计数，天然规避 AIHelms 多部门 JOIN 重复计数
- **人均 token** = 总 token/激活用户数（活跃人均）；**覆盖率环比** = 当期-上期百分点差（分母同为当前用户快照）
- **DAU** = (业务日,用户) 粒度 LLM∪MCP 去重后按日计数；调用数为两源之和

## 落地

- 后端四层：service/gateway/adoption.go（AdoptionService）+ api + router + enter 三处注册；request/response 各一文件（**AdoptionSearch 直接别名 = CostAnalysisSearch**，dimension/sort 在本域无意义，零重复绑定结构）
- 4 端点挂 /gateway/adoption/*（overview=KPI+DAU 趋势；departments=全部部门行**含零调用部门**+「未分配」兜底行，覆盖率视角与成本明细只含有消耗组的差异点；departments/:id/users=成员下钻**含未激活成员**兼未使用人员清单，激活在前，部门Key消耗不出现在成员行；models=模型分布仅 LLM 维+占比 Go 算）
- 菜单 route.ai-audit_adoption 挂 AI审计目录 OrderNum 4（ApiPrefix /gateway/adoption, /gateway/adoption/*），user 角色不授；super/admin 循环全量授权零改 sys_role_menu
- 前端 _gateway/ai-audit/adoption/：index + 4 modules（search 复用 cost.preset i18n 语义/overview 4 KPI 卡+DAU 柱线双轴/dept-table 行内 expand 成员子表带 NProgress 覆盖率条/models 占比条）
- i18n page.gateway.adoption.* 三处同步（zh-cn/en-us/app.d.ts）

## 明确不做（本轮拍板）

- 不做重度用户分档（AIHelms 三处口径矛盾的坑源，需求未要求）
- 不做未使用用户独立端点（部门成员下钻已含未激活成员）
- 已有库菜单补丁 SQL 不落（近期实践：dev 重建库种子生效，92b98a0 已清历史补丁；上生产时统一补）
