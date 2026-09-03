package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	"github.com/hllkk/devops-admin/server/model/system"
)

// ReportService 效能报告(P3)。模板化数据报告：复用 AdoptionService/CostAnalysisService
// 取数组装结构化 JSON + Markdown，不调 LLM 生成文字(AIHelms 的 LLM 报告是半成品——
// POST /reports 只插"生成中"占位记录、无任何后台任务填充,报告永远停在"生成中";
// 本项目保证真闭环)。周/月报定时生成幂等(同类型同起始日已存在直接返回)。
type ReportService struct{}

// 报告明细截断(部门/模型 Top 20,用户 Top 10)
const (
	reportDeptLimit  = 20
	reportModelLimit = 20
	reportUserLimit  = 10
)

// GenerateReport 生成报告并落库；weekly/monthly 幂等(已存在返回既有报告)。
// createdBy=0 表示定时任务生成。
func (s *ReportService) GenerateReport(ctx context.Context, reportType, startStr, endStr string, createdBy int64) (gatewayResp.EfficiencyReportView, error) {
	start, end, err := normalizeReportRange(reportType, startStr, endStr, time.Now())
	if err != nil {
		return gatewayResp.EfficiencyReportView{}, err
	}

	// 周报/月报幂等：同类型同起始日已存在直接返回
	if reportType != gateway.ReportTypeCustom {
		var exist gateway.EfficiencyReport
		if err := global.OPS_DB.WithContext(ctx).
			Where("report_type = ? AND period_start = ?", reportType, start).First(&exist).Error; err == nil {
			return s.toDetailView(ctx, exist)
		}
	}

	f := &gatewayReq.AdoptionSearch{StartDate: start, EndDate: end}
	adoptionSvc := AdoptionService{}
	ov, err := adoptionSvc.GetAdoptionOverview(ctx, f)
	if err != nil {
		return gatewayResp.EfficiencyReportView{}, err
	}
	deptRows, err := adoptionSvc.GetAdoptionDepartments(ctx, f)
	if err != nil {
		return gatewayResp.EfficiencyReportView{}, err
	}
	if len(deptRows) > reportDeptLimit {
		deptRows = deptRows[:reportDeptLimit]
	}
	modelRows, err := adoptionSvc.GetAdoptionModels(ctx, f)
	if err != nil {
		return gatewayResp.EfficiencyReportView{}, err
	}
	if len(modelRows) > reportModelLimit {
		modelRows = modelRows[:reportModelLimit]
	}
	userSearch := &gatewayReq.CostAnalysisSearch{
		StartDate: start, EndDate: end, Dimension: dimensionUser,
	}
	userSearch.PageNum, userSearch.PageSize = 1, reportUserLimit
	topUsers, _, err := (&CostAnalysisService{}).GetCostDetail(ctx, userSearch)
	if err != nil {
		return gatewayResp.EfficiencyReportView{}, err
	}

	ca := CostAnalysisService{}
	curInt, curExt, _, _, err := ca.sumCost(ctx, f, start, end)
	if err != nil {
		return gatewayResp.EfficiencyReportView{}, err
	}
	mInt, mExt, _, err := sumMcpCost(ctx, f, start, end)
	if err != nil {
		return gatewayResp.EfficiencyReportView{}, err
	}

	content := gatewayResp.ReportContent{
		Kpi: gatewayResp.ReportKpi{
			AdoptionKpi: ov.Kpi, InternalCost: curInt + mInt, ExternalCost: curExt + mExt,
			CostDiff: curExt + mExt - curInt - mInt,
		},
		DeptRows: deptRows, ModelRows: modelRows, TopUsers: topUsers,
	}
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return gatewayResp.EfficiencyReportView{}, err
	}

	kpi := ov.Kpi
	summary := fmt.Sprintf("覆盖率 %.1f%%(环比%+.1fpp)，激活 %d/%d 人，调用 %d 次，内部成本 ¥%.2f",
		kpi.Coverage, kpi.CoverageChange, kpi.ActiveUsers, kpi.TotalUsers,
		kpi.TotalRequests, content.Kpi.InternalCost)

	row := gateway.EfficiencyReport{
		ReportType: reportType, PeriodStart: start, PeriodEnd: end,
		Summary: summary, Content: contentJSON,
		ContentMd: renderReportMarkdown(reportType, start, end, &content),
		CreatedBy: createdBy,
	}
	if err := global.OPS_DB.WithContext(ctx).Create(&row).Error; err != nil {
		return gatewayResp.EfficiencyReportView{}, err
	}
	return s.toDetailView(ctx, row)
}

// GetReportList 分页列表(不带 content/content_md 大字段)，类型筛选，新在前。
func (s *ReportService) GetReportList(ctx context.Context, q *gatewayReq.ReportSearch) ([]gatewayResp.EfficiencyReportView, int64, error) {
	db := global.OPS_DB.WithContext(ctx).Model(&gateway.EfficiencyReport{})
	if q.ReportType != "" {
		db = db.Where("report_type = ?", q.ReportType)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []gateway.EfficiencyReport
	limit, offset := q.LimitOffset()
	if err := db.Select("report_id, report_type, period_start, period_end, summary, created_by, create_time").
		Order("period_start DESC, report_id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return s.toListView(ctx, rows), total, nil
}

// GetReport 详情(解析 content + 生成人回填)。
func (s *ReportService) GetReport(ctx context.Context, id int64) (gatewayResp.EfficiencyReportView, error) {
	var row gateway.EfficiencyReport
	if err := global.OPS_DB.WithContext(ctx).Where("report_id = ?", id).First(&row).Error; err != nil {
		return gatewayResp.EfficiencyReportView{}, err
	}
	return s.toDetailView(ctx, row)
}

// ExportReport 构建 xlsx(三 sheet:部门覆盖率/模型分布/用户Top)。
func (s *ReportService) ExportReport(ctx context.Context, id int64) (*string, []byte, error) {
	view, err := s.GetReport(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	content := view.Content
	if content == nil {
		content = &gatewayResp.ReportContent{}
	}

	f := excelize.NewFile()
	sheetDept, sheetModel, sheetUser := "部门覆盖率", "模型分布", "用户Top"
	_, _ = f.NewSheet(sheetModel)
	_, _ = f.NewSheet(sheetUser)
	_ = f.SetSheetName("Sheet1", sheetDept)

	// 部门覆盖率
	_ = f.SetSheetRow(sheetDept, "A1", &[]interface{}{
		"部门", "成员数", "激活数", "覆盖率(%)", "调用数", "总Token", "内部成本(¥)"})
	for i, r := range content.DeptRows {
		_ = f.SetSheetRow(sheetDept, fmt.Sprintf("A%d", i+2), &[]interface{}{
			r.DeptName, r.MemberCount, r.ActiveCount, round1(r.Coverage), r.Requests, r.TotalTokens, round4(r.InternalCost)})
	}

	// 模型分布
	_ = f.SetSheetRow(sheetModel, "A1", &[]interface{}{
		"模型", "调用数", "调用占比(%)", "总Token", "内部成本(¥)", "成本占比(%)", "活跃用户"})
	for i, r := range content.ModelRows {
		_ = f.SetSheetRow(sheetModel, fmt.Sprintf("A%d", i+2), &[]interface{}{
			r.Model, r.Requests, round1(r.RequestShare), r.TotalTokens, round4(r.InternalCost), round1(r.CostShare), r.ActiveUsers})
	}

	// 用户 Top(成本口径)
	_ = f.SetSheetRow(sheetUser, "A1", &[]interface{}{
		"用户", "调用数", "总Token", "内部成本(¥)", "外部成本(¥)", "结算差额(¥)"})
	for i, r := range content.TopUsers {
		_ = f.SetSheetRow(sheetUser, fmt.Sprintf("A%d", i+2), &[]interface{}{
			r.Label, r.Requests, r.TotalTokens, round4(r.InternalCost), round4(r.ExternalCost), round4(r.CostDiff)})
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, nil, err
	}
	name := fmt.Sprintf("效能报告_%s_%s~%s", reportTypeName(view.ReportType), view.PeriodStart, view.PeriodEnd)
	return &name, buf.Bytes(), nil
}

// LastWeekRange 上周一~上周日(本地时区业务日,周一为一周开始)。
func LastWeekRange(now time.Time) (string, string) {
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday
	}
	thisMonday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(weekday - 1))
	lastMonday := thisMonday.AddDate(0, 0, -7)
	return lastMonday.Format("2006-01-02"), thisMonday.AddDate(0, 0, -1).Format("2006-01-02")
}

// LastMonthRange 上月 1 号~月末(本地时区业务日)。
func LastMonthRange(now time.Time) (string, string) {
	firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastOfPrev := firstOfThisMonth.AddDate(0, 0, -1)
	firstOfPrev := time.Date(lastOfPrev.Year(), lastOfPrev.Month(), 1, 0, 0, 0, 0, now.Location())
	return firstOfPrev.Format("2006-01-02"), lastOfPrev.Format("2006-01-02")
}

// ReportNoticeDraft 报告生成通知草稿(发送由调用方执行——service/gateway 不得引
// service/system 防环，同预算告警 BudgetAlertNotices 的草稿模式)。
type ReportNoticeDraft struct {
	Title   string
	Content string
	Url     string
	UserIds []int64
}

// BuildReportNotice 组装报告生成通知(目标=super/admin 角色启用用户)；无可达目标返回 nil。
func (s *ReportService) BuildReportNotice(ctx context.Context, view gatewayResp.EfficiencyReportView) *ReportNoticeDraft {
	var userIds []int64
	if err := global.OPS_DB.WithContext(ctx).Table("sys_users u").
		Select("DISTINCT u.id").
		Joins("JOIN sys_user_role ur ON ur.sys_user_id = u.id").
		Joins("JOIN sys_role r ON r.role_id = ur.sys_role_id").
		Where("r.role_key IN ? AND u.status = ? AND u.deleted_at IS NULL", []string{"super", "admin"}, "0").
		Scan(&userIds).Error; err != nil || len(userIds) == 0 {
		return nil
	}
	return &ReportNoticeDraft{
		Title:   fmt.Sprintf("AI 网关%s已生成(%s~%s)", reportTypeName(view.ReportType), view.PeriodStart, view.PeriodEnd),
		Content: view.Summary, Url: "/gateway/ai-audit/report", UserIds: userIds,
	}
}

// ───────────────── 内部辅助 ─────────────────

// normalizeReportRange 报告周期归一：weekly=上周、monthly=上月、custom=显式起止。
func normalizeReportRange(reportType, startStr, endStr string, now time.Time) (string, string, error) {
	switch reportType {
	case gateway.ReportTypeWeekly:
		start, end := LastWeekRange(now)
		return start, end, nil
	case gateway.ReportTypeMonthly:
		start, end := LastMonthRange(now)
		return start, end, nil
	default:
		if startStr == "" || endStr == "" {
			return "", "", fmt.Errorf("自定义报告必须指定起止业务日")
		}
		start, end, _ := normalizeCostRange(startStr, endStr, now)
		return start, end, nil
	}
}

// toDetailView 详情视图(content JSON 解析+生成人回填)。
func (s *ReportService) toDetailView(ctx context.Context, row gateway.EfficiencyReport) (gatewayResp.EfficiencyReportView, error) {
	view := gatewayResp.EfficiencyReportView{
		ReportId: row.ReportId, ReportType: row.ReportType,
		PeriodStart: row.PeriodStart, PeriodEnd: row.PeriodEnd,
		Summary: row.Summary, ContentMd: row.ContentMd,
		CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt,
	}
	var content gatewayResp.ReportContent
	if len(row.Content) > 0 {
		if err := json.Unmarshal(row.Content, &content); err != nil {
			return view, fmt.Errorf("报告内容解析失败: %w", err)
		}
		view.Content = &content
	}
	if row.CreatedBy != 0 {
		var user system.SysUser
		if err := global.OPS_DB.WithContext(ctx).Select("id, nick_name").
			Where("id = ?", row.CreatedBy).First(&user).Error; err == nil {
			view.CreatorName = user.NickName
		}
	}
	return view, nil
}

// toListView 轻量列表视图(生成人昵称批量回填)。
func (s *ReportService) toListView(ctx context.Context, rows []gateway.EfficiencyReport) []gatewayResp.EfficiencyReportView {
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		if r.CreatedBy != 0 {
			ids = append(ids, r.CreatedBy)
		}
	}
	names := map[int64]string{}
	if len(ids) > 0 {
		var users []system.SysUser
		if err := global.OPS_DB.WithContext(ctx).Select("id, nick_name").Where("id IN ?", ids).Find(&users).Error; err == nil {
			for i := range users {
				names[users[i].UserId] = users[i].NickName
			}
		}
	}
	list := make([]gatewayResp.EfficiencyReportView, 0, len(rows))
	for _, r := range rows {
		list = append(list, gatewayResp.EfficiencyReportView{
			ReportId: r.ReportId, ReportType: r.ReportType,
			PeriodStart: r.PeriodStart, PeriodEnd: r.PeriodEnd,
			Summary: r.Summary, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt,
			CreatorName: names[r.CreatedBy],
		})
	}
	return list
}

// renderReportMarkdown Markdown 文本(复制/通知用,数字表 + 摘要)。
func renderReportMarkdown(reportType, start, end string, c *gatewayResp.ReportContent) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# AI 网关效能%s（%s ~ %s）\n\n", reportTypeName(reportType), start, end)
	kpi := c.Kpi
	fmt.Fprintf(&b, "覆盖率 **%.1f%%**（环比 %+.1fpp），激活 %d/%d 人，新增活跃 %d\n\n", kpi.Coverage, kpi.CoverageChange, kpi.ActiveUsers, kpi.TotalUsers, kpi.NewActiveUsers)
	fmt.Fprintf(&b, "调用 %d 次（日均 %.0f），人均 Token %d，内部成本 ¥%.2f（外部 ¥%.2f，差额 ¥%.2f）\n\n",
		kpi.TotalRequests, kpi.DailyRequests, kpi.PerCapitaTokens, kpi.InternalCost, kpi.ExternalCost, kpi.CostDiff)

	b.WriteString("## 部门覆盖率\n\n| 部门 | 成员 | 激活 | 覆盖率 | 调用 | 内部成本(¥) |\n|---|---|---|---|---|---|\n")
	for _, r := range c.DeptRows {
		fmt.Fprintf(&b, "| %s | %d | %d | %.1f%% | %d | %.2f |\n", r.DeptName, r.MemberCount, r.ActiveCount, r.Coverage, r.Requests, r.InternalCost)
	}
	b.WriteString("\n## 模型分布\n\n| 模型 | 调用 | 占比 | 内部成本(¥) |\n|---|---|---|---|\n")
	for _, r := range c.ModelRows {
		fmt.Fprintf(&b, "| %s | %d | %.1f%% | %.2f |\n", r.Model, r.Requests, r.RequestShare, r.InternalCost)
	}
	b.WriteString("\n## 用户 Top10(内部成本)\n\n| 用户 | 调用 | 总Token | 内部成本(¥) |\n|---|---|---|---|\n")
	for _, r := range c.TopUsers {
		fmt.Fprintf(&b, "| %s | %d | %d | %.2f |\n", r.Label, r.Requests, r.TotalTokens, r.InternalCost)
	}
	return b.String()
}

// reportTypeName 报告类型中文名。
func reportTypeName(t string) string {
	switch t {
	case gateway.ReportTypeWeekly:
		return "周报"
	case gateway.ReportTypeMonthly:
		return "月报"
	default:
		return "报告"
	}
}

// round1/round4 数值精度。
func round1(v float64) float64  { return float64(int64(v*10+0.5)) / 10 }
func round4(v float64) float64 { return float64(int64(v*10000+0.5)) / 10000 }
