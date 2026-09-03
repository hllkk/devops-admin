# AI 网关·P3 效能报告

- 日期：2026-09-02
- 状态：已实现（go build/vet/test + typecheck/eslint 0 警告通过，待运行时验证）
- 关联：[[ai-gateway-p3-adoption]]、[[ai-gateway-p3-health]]（同日三件收尾 P3）

## 需求

P3 收尾三件之三：效能报告——周/月自动生成 + 多维下钻 + 导出。用户拍板：模板化数据报告（不调 LLM）+ 复用现有 excel 同步导出（不做异步导出任务中心）。

## 核心决策：不照搬 AIHelms

AIHelms 的报告是**半成品**：POST /reports 只插 summary="报告生成中..."、content_md="" 的占位记录，全仓无任何后台任务/LLM 调用填充内容，报告永远停在"生成中"；"导出 PDF"是无 click 处理的死按钮；suggestions 双存储冗余。本项目走模板化真闭环：复用 AdoptionService/CostAnalysisService 取数组装结构化 JSON + Markdown，零 LLM 依赖零成本。

## 落地

- **新表 gateway_efficiency_report**（report_type weekly/monthly/custom + period_start/end + summary + content JSONB[ReportContent:Kpi/部门Top20/模型Top20/用户Top10] + content_md + created_by[0=定时任务]），注册 AutoMigrate
- **ReportService**：GenerateReport（weekly=上周一~周日 LastWeekRange/monthly=上月 LastMonthRange/custom 显式起止；**同类型同起始日幂等**——已存在直接返回，定时重跑安全）+ GetReportList（分页不带大字段）+ GetReport（content JSON 解析+生成人回填）+ ExportReport（excelize 三 sheet：部门覆盖率/模型分布/用户Top）
- **通知防 import 环**（service/gateway 不得引 service/system）：`BuildReportNotice` 产草稿（目标=super/admin 角色启用用户 SQL）→ 发送在 timer 闭包执行（同预算告警 BudgetAlertNotices 模式）；手动生成不通知（管理员自己在看）
- **定时**：GenerateWeeklyEfficiencyReport 周一 08:13（`13 8 * * 1`）+ GenerateMonthlyEfficiencyReport 每月 1 号 08:23（`23 8 1 * *`），种子仅新库生效
- **端点**：GET /gateway/report/list、GET /gateway/report/:id、POST /gateway/report/generate（utils.GetUserID 取生成人）、POST /gateway/report/export/:id（POST 对齐前端 useDownload 表单下载模式）；writeReportXlsx 复制自 sys_export.go 的 writeXlsx（gateway 首例，注释互指不提公共包）
- 菜单 route.ai-audit_report 挂 AI审计目录 OrderNum 6，user 角色不授
- **前端** `_gateway/ai-audit/report/`：列表卡片（类型筛选 NRadioButton all 哨兵+分页+行点击载详情）+ 详情卡片（NAlert 摘要+8 KPI 卡+NTabs 三明细表+复制 Markdown useClipboard+导出 useDownload）+ report-generate-modal（类型三选+custom 时 daterange）；i18n page.gateway.report.* 三处同步，明细列文案复用 adoption/cost 既有 key
- 坑规避：i18n 的 $t 模板字符串 key 需联合类型收窄（typeLabel 参数用字面量联合）；模板内联 row-key 箭头注 Api 类型触发 vue/no-undef-properties warning（函数移 script 消除）

## 明确不做

- LLM 生成文字总结与建议（AIHelms suggestions 形态，依赖网关自身可用+有成本，需要时另立）
- 异步导出任务中心/下载中心（用户拍板同步 xlsx 足够）
- 日报/季报（AIHelms 前端有后端 422 的死选项，不学）
