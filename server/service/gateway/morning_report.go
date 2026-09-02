package gateway

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/hllkk/devops-admin/server/global"
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

// BuildMorningReport 汇总全部 token_plan 余量快照(按供应商)组晨报草稿。
// 口径：坐席+共享包 SUM(total/surplus)；重置日取 MAX(cycle_end)(坐席周期基本一致，取最晚保底)。
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
		resetLine := morningResetLine(r.CycleEnd)
		d := MorningReportDraft{
			ProviderId:   r.ProviderId,
			ProviderName: r.ProviderName,
			Title:        fmt.Sprintf("【AI 平台晨报】%s", r.ProviderName),
			Content:      morningContent(r.ProviderName, usedPercent, r.Surplus, r.Total, resetLine, r.Surplus <= 0),
			Markdown:     morningMarkdown(r.ProviderName, usedPercent, r.Surplus, r.Total, resetLine, r.Surplus <= 0),
		}
		drafts = append(drafts, d)
	}
	return drafts, nil
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

// morningContent 纯文本正文(站内+企微应用消息)。
func morningContent(name string, usedPercent, surplus, total float64, resetLine string, overdrawn bool) string {
	s := fmt.Sprintf("%s 已使用 %.1f%%（剩余 %s / 总 %s Credits）", name, usedPercent, formatCredits(surplus), formatCredits(total))
	if resetLine != "" {
		s += "\n" + resetLine
	}
	if overdrawn {
		s += "\n当前已超量，请临时切换到 MIMO 或其他个人自定义模型。"
	} else {
		s += "\n如已超量，可临时切换到 MIMO 或其他个人自定义模型。"
	}
	return s
}

// morningMarkdown 群机器人 markdown 正文。
func morningMarkdown(name string, usedPercent, surplus, total float64, resetLine string, overdrawn bool) string {
	s := fmt.Sprintf("## 【AI 平台晨报】%s\n已使用 **%.1f%%**（剩余 %s / 总 %s Credits）", name, usedPercent, formatCredits(surplus), formatCredits(total))
	if resetLine != "" {
		s += "\n重置日：**" + resetLine + "**"
	}
	if overdrawn {
		s += "\n> 当前已超量，请临时切换到 MIMO 或其他个人自定义模型。"
	} else {
		s += "\n> 如已超量，可临时切换到 MIMO 或其他个人自定义模型。"
	}
	return s
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
