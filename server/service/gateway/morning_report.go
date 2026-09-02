package gateway

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"text/template"
	"time"

	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	sysModel "github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// MorningReportService TokenPlan 工作日晨报组装(纯 gateway 数据→通知草稿)。
// 发送(站内/企微应用/群机器人)由 initialize.timer 闭包调 system.NotifySendService 完成，
// 规避 gateway service→system service 反向 import 环(sys_user_manage 已 import gateway)。
// 数据源 gateway_provider_balance 旁路快照(不进成本链路)；无数据返回空列表由调用方跳过。
type MorningReportService struct{}

// MorningReportDraft 单供应商晨报草稿(渠道无关)。
type MorningReportDraft struct {
	ProviderId   int64
	ProviderName string
	Title        string
	Content      string // 纯文本(站内通知+企微应用消息)
	Markdown     string // 群机器人 markdown
}

// MorningTemplateVars 晨报正文模板变量(Go text/template),自定义模板与默认模板共用同一变量集。
type MorningTemplateVars struct {
	ProviderName string  // 供应商名称
	UsedPercent  float64 // 使用率百分数值(如 45.2)
	Surplus      string  // 剩余(中文数量级格式化,如 1.2亿)
	Total        string  // 总量(同上)
	ResetLine    string  // 重置日文案(如 9月5日重置（3 天后）;可能为空)
	Overdrawn    bool    // 是否已超量
}

// 默认模板(与历史硬编码文案保持一致;正文可被 sys_notify_policy.params 的自定义模板替换)
const defaultMorningContentTpl = `{{.ProviderName}} 已使用 {{printf "%.1f" .UsedPercent}}%（剩余 {{.Surplus}} / 总 {{.Total}} Credits）
{{if .ResetLine}}{{.ResetLine}}
{{end}}{{if .Overdrawn}}当前已超量，请临时切换到 MIMO 或其他个人自定义模型。{{else}}如已超量，可临时切换到 MIMO 或其他个人自定义模型。{{end}}`

const defaultMorningMarkdownTpl = `## 【AI 平台晨报】{{.ProviderName}}
已使用 **{{printf "%.1f" .UsedPercent}}%**（剩余 {{.Surplus}} / 总 {{.Total}} Credits）
{{if .ResetLine}}重置日：**{{.ResetLine}}**
{{end}}{{if .Overdrawn}}> 当前已超量，请临时切换到 MIMO 或其他个人自定义模型。{{else}}> 如已超量，可临时切换到 MIMO 或其他个人自定义模型。{{end}}`

// BuildMorningReport 汇总全部 token_plan 余量快照(按供应商)组晨报草稿。
// 口径：坐席+共享包 SUM(total/surplus)；重置日取 MAX(cycle_end)(坐席周期基本一致，取最晚保底)。
// 正文模板取晨报策略 params(场景勾选/目标群由 timer 侧解析,此处只读模板;直查表规避 service 反向依赖)。
func (s *MorningReportService) BuildMorningReport(ctx context.Context) ([]MorningReportDraft, error) {
	type row struct {
		ProviderId   int64
		ProviderName string
		Total        float64
		Surplus      float64
		CycleEnd     *time.Time
	}
	var rows []row
	err := global.OPS_DB.WithContext(ctx).Table("gateway_provider_balance b").
		Select("b.provider_id, p.name AS provider_name, SUM(b.total_value) AS total, SUM(b.surplus_value) AS surplus, MAX(b.cycle_end) AS cycle_end").
		Joins("JOIN gateway_provider p ON p.provider_id = b.provider_id").
		Where("b.plan_type = ?", "token_plan").
		Group("b.provider_id, p.name").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("汇总 TokenPlan 余量失败: %w", err)
	}
	params := s.loadTemplateParams(ctx)
	drafts := make([]MorningReportDraft, 0, len(rows))
	for _, r := range rows {
		if r.ProviderId == 0 || r.Total <= 0 {
			continue
		}
		usedPercent := (r.Total - r.Surplus) / r.Total * 100
		if usedPercent < 0 {
			usedPercent = 0
		}
		if usedPercent > 100 {
			usedPercent = 100
		}
		v := MorningTemplateVars{
			ProviderName: r.ProviderName,
			UsedPercent:  usedPercent,
			Surplus:      formatCredits(r.Surplus),
			Total:        formatCredits(r.Total),
			ResetLine:    morningResetLine(r.CycleEnd),
			Overdrawn:    r.Surplus <= 0,
		}
		drafts = append(drafts, MorningReportDraft{
			ProviderId:   r.ProviderId,
			ProviderName: r.ProviderName,
			Title:        fmt.Sprintf("【AI 平台晨报】%s", r.ProviderName),
			Content:      renderMorningBody(ctx, params.ContentTemplate, defaultMorningContentTpl, "纯文本", v),
			Markdown:     renderMorningBody(ctx, params.MarkdownTemplate, defaultMorningMarkdownTpl, "markdown", v),
		})
	}
	return drafts, nil
}

// loadTemplateParams 读晨报策略 params 取正文模板(渠道勾选在 timer 侧判定,不在此处理)。
// 策略行不存在(未配置过晨报)或 params 解析失败均降级零值(即默认模板),失败记 Warn 不中断发送。
func (s *MorningReportService) loadTemplateParams(ctx context.Context) sysModel.MorningReportParams {
	var policy sysModel.SysNotifyPolicy
	err := global.OPS_DB.WithContext(ctx).
		Where("scene_key = ?", sysModel.NotifySceneTokenPlanMorning).
		First(&policy).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return sysModel.MorningReportParams{}
	}
	if err != nil {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Warn("读取晨报策略失败,正文降级默认模板")
		return sysModel.MorningReportParams{}
	}
	params, err := sysModel.ParseMorningReportParams(policy.Params)
	if err != nil {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Warn("解析晨报模板参数失败,正文降级默认模板")
		return sysModel.MorningReportParams{}
	}
	return params
}

// renderMorningBody 渲染晨报正文:自定义模板优先,留空用默认;解析/执行失败降级默认模板并记 Warn
// (模板坏数据只影响正文形态,不丢当日晨报)。
func renderMorningBody(ctx context.Context, customTpl, defaultTpl, kind string, v MorningTemplateVars) string {
	tplText := customTpl
	if tplText == "" {
		tplText = defaultTpl
	}
	tmpl, err := template.New("morning").Parse(tplText)
	if err != nil {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Warn("晨报" + kind + "模板解析失败,降级默认模板")
		tmpl = template.Must(template.New("morning").Parse(defaultTpl))
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, v); err != nil {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Warn("晨报" + kind + "模板执行失败,降级默认模板")
		sb.Reset()
		tmpl = template.Must(template.New("morning").Parse(defaultTpl))
		_ = tmpl.Execute(&sb, v)
	}
	return sb.String()
}

// morningResetLine 重置日行(cycle_end 为空返回空串整行省略)。日期用服务器本地时区
// (容器 TZ=Asia/Shanghai；快照时间为 UTC，直接取日期避免跨天错位)。
func morningResetLine(cycleEnd *time.Time) string {
	if cycleEnd == nil || cycleEnd.IsZero() {
		return ""
	}
	end := cycleEnd.Local()
	dateLabel := end.Format("1月2日")
	days := int(math.Ceil(end.Sub(time.Now()).Hours() / 24))
	switch {
	case days > 1:
		return fmt.Sprintf("%s重置（%d 天后）", dateLabel, days)
	case days == 1:
		return fmt.Sprintf("%s重置（明天）", dateLabel)
	case days == 0:
		return fmt.Sprintf("%s重置（今天）", dateLabel)
	default:
		return fmt.Sprintf("本周期已于 %s 重置", dateLabel)
	}
}

// formatCredits Credits 中文数量级格式化(亿/万)，避免大数字难读。
func formatCredits(v float64) string {
	switch {
	case math.Abs(v) >= 1e8:
		return fmt.Sprintf("%.2f亿", v/1e8)
	case math.Abs(v) >= 1e4:
		return fmt.Sprintf("%.1f万", v/1e4)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}
