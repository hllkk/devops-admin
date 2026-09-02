package system

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// NotifySendService 通知三渠道统一发送:站内(复用 NoticeService,SSE+已读跟踪)、
// 企微应用消息(sys_social 映射+分批 message/send)、企微群机器人(webhook markdown)。
//
// 架构约束:service/system 已被 gateway service 反向依赖(sys_user_manage import gateway),
// 场景组装(gateway 数据→通知草稿)必须留在 gateway service 或调用方,本服务只做"草稿→渠道"。
// 渠道总开关/事件开关取自 sys_notify_config,企微凭证取自 sys_auth_config(均内存缓存热读)。
type NotifySendService struct{}

// SendChannels 指定各渠道是否发送。
type SendChannels struct {
	InApp    bool // 站内通知(NoticeService.CreateNotice,SSE 实时+已读跟踪)
	WecomApp bool // 企微应用消息(按人)
	WecomBot bool // 企微群机器人(markdown 进群)
}

// SendRequest 统一发送请求(渠道无关的场景草稿)。
type SendRequest struct {
	Title      string   // 标题(站内标题+应用消息 textcard 标题)
	Content    string   // 正文(站内正文+应用消息 textcard 描述,纯文本)
	Markdown   string   // 群机器人 markdown 正文(空则降级用 Content)
	Url        string   // 应用消息跳转地址(以/开头时拼接配置的跳转 base;空则 textcard 降级纯文本)
	TargetType string   // users/depts(depts 含子部门展开为成员)
	UserIds    []int64  // TargetType=users 时的目标用户
	DeptIds    []int64  // TargetType=depts 时的目标部门
	Channels   SendChannels
}

// Send 按渠道逐个发送。单渠道失败记日志不中断其余渠道(外部推送 best-effort),
// 站内失败记 Error 返回(站内是主渠道,失败应让调用方感知)。
func (s *NotifySendService) Send(ctx context.Context, req SendRequest) error {
	if req.TargetType != system.NotifyPolicyTargetAll && req.TargetType != system.NotifyPolicyTargetDepts {
		req.TargetType = system.NotifyPolicyTargetUsers // 非法值归一到 users
	}
	// 展开目标用户(应用消息按人发送用):users/depts 走 notice 同款展开,
	// all 展开为全量启用用户(CreateNotice 的 all 是广播语义,与此独立)
	targetUserIds := (&NoticeService{}).expandTargetUserIDs(ctx, req.TargetType, req.UserIds, req.DeptIds)
	if req.TargetType == system.NotifyPolicyTargetAll {
		var ids []int64
		// sys_users.status '0'=正常 '1'=停用(data_scope:skip 旁路数据权限查全量)
		global.OPS_DB.WithContext(ctx).Set("data_scope:skip", true).
			Model(&system.SysUser{}).Where("status = ?", "0").Pluck("id", &ids)
		targetUserIds = ids
	}

	if req.Channels.InApp {
		if err := s.SendInApp(ctx, req.Title, req.Content, req.TargetType, req.UserIds, req.DeptIds); err != nil {
			logger.WithCtx(ctx).Mod("system").Err(err).Error("站内通知发送失败")
			return err
		}
	}
	if req.Channels.WecomApp {
		if err := s.SendWecomApp(ctx, targetUserIds, req.Title, req.Content, s.resolveRedirect(ctx, req.Url)); err != nil {
			logger.WithCtx(ctx).Mod("system").Err(err).Error("企微应用消息发送失败")
		}
	}
	if req.Channels.WecomBot {
		md := req.Markdown
		if md == "" {
			md = req.Content
		}
		if err := s.SendWecomBot(ctx, md); err != nil {
			logger.WithCtx(ctx).Mod("system").Err(err).Error("企微群机器人发送失败")
		}
	}
	return nil
}

// SendInApp 站内定向通知(复用 NoticeService:落库+SSE 弹窗+已读跟踪)。
func (s *NotifySendService) SendInApp(ctx context.Context, title, content, targetType string, userIds, deptIds []int64) error {
	return (&NoticeService{}).CreateNotice(ctx, noticeOperateOf(title, content, targetType, userIds, deptIds), 0)
}

// SendWecomApp 企微应用消息:sys_social(wecom) 取企微 userid → 未绑定跳过 → 截断上限 →
// 分批(≤1000) SendMessage。部分用户未绑定属正常(未走企微登录),仅 Warn 计数。
func (s *NotifySendService) SendWecomApp(ctx context.Context, userIds []int64, title, desc, url string) error {
	cfg := (&NotifyConfigService{}).Current(ctx)
	if !cfg.WecomPushEnabled {
		return nil
	}
	client := wecomClientFromCfg((&AuthConfigService{}).Current(ctx))
	if client.CorpID == "" || client.CorpSecret == "" || client.AgentID == 0 {
		return errWecomNotConfigured
	}
	if len(userIds) == 0 {
		return nil
	}
	// 一次 IN 查绑定表:user_id → 企微 userid
	type socialRow struct {
		UserId int64
		OpenId string
	}
	var socials []socialRow
	if err := global.OPS_DB.WithContext(ctx).
		Model(&system.SysSocial{}).
		Select("user_id", "open_id").
		Where("user_id IN ? AND source = ?", userIds, "wecom").
		Scan(&socials).Error; err != nil {
		return err
	}
	openIds := make([]string, 0, len(socials))
	for _, r := range socials {
		if r.OpenId != "" {
			openIds = append(openIds, r.OpenId)
		}
	}
	if skipped := len(userIds) - len(openIds); skipped > 0 {
		logger.WithCtx(ctx).Mod("system").Warn("企微应用消息: " + itoa(skipped) + " 个目标用户未绑定企业微信,已跳过")
	}
	if len(openIds) == 0 {
		return nil
	}
	maxTargets := cfg.WecomPushMaxTargets
	if maxTargets <= 0 {
		maxTargets = 1000
	}
	if len(openIds) > maxTargets {
		logger.WithCtx(ctx).Mod("system").Warn("企微应用消息目标超出上限,截断: " + itoa(len(openIds)) + " → " + itoa(maxTargets))
		openIds = openIds[:maxTargets]
	}
	card := utils.WecomTextCard{Title: title, Description: desc, URL: url, Btntxt: "前往查看"}
	for i := 0; i < len(openIds); i += utils.WecomMessageSendBatch {
		end := i + utils.WecomMessageSendBatch
		if end > len(openIds) {
			end = len(openIds)
		}
		if err := client.SendMessage(ctx, openIds[i:end], card); err != nil {
			return err // 配置级错误重试无意义,放弃后续批次
		}
	}
	return nil
}

// SendWecomBot 企微群机器人 markdown 消息(发送到 sys_notify_config 配置的 webhook)。
func (s *NotifySendService) SendWecomBot(ctx context.Context, markdown string) error {
	cfg := (&NotifyConfigService{}).Current(ctx)
	if !cfg.WecomBotEnabled || cfg.WecomBotWebhook == "" {
		return nil
	}
	return (&utils.WecomClient{}).SendBotMessage(ctx, cfg.WecomBotWebhook, markdown)
}

// resolveRedirect 应用消息跳转地址:相对路径(以/开头)拼配置 base,空/绝对地址原样返回。
// base 未配置时相对路径置空(textcard 降级纯文本,适配未配可信域名的部署)。
func (s *NotifySendService) resolveRedirect(ctx context.Context, url string) string {
	if url == "" || !strings.HasPrefix(url, "/") {
		return url
	}
	base := strings.TrimSuffix((&NotifyConfigService{}).Current(ctx).WecomPushRedirectBase, "/")
	if base == "" {
		return ""
	}
	return base + url
}

// --- 内部 helper ---

// errWecomNotConfigured 企微凭证缺失(sys_auth_config 未配置 CorpId/AgentId/Secret)。
var errWecomNotConfigured = errors.New("企业微信 CorpId/AgentId/Secret 未配置(系统设置→认证配置)")

// wecomClientFromCfg 从 sys_auth_config 构造企微客户端(service 层版,api 层 wecomClientFromConfig 同款)。
func wecomClientFromCfg(cfg system.SysAuthConfig) *utils.WecomClient {
	return &utils.WecomClient{
		CorpID:      cfg.WecomCorpId,
		AgentID:     cfg.WecomAgentId,
		CorpSecret:  cfg.WecomClientSecret,
		RedirectURI: cfg.WecomCallbackUrl,
	}
}

// noticeOperateOf 组装站内通知入参(系统发送:类型=通知、状态=发布)。
func noticeOperateOf(title, content, targetType string, userIds, deptIds []int64) systemReq.NoticeOperateParams {
	return systemReq.NoticeOperateParams{
		NoticeTitle:   title,
		NoticeType:    system.NoticeTypeNotice,
		NoticeContent: content,
		Status:        "0",
		TargetType:    targetType,
		TargetUserIds: userIds,
		TargetDeptIds: deptIds,
	}
}

// itoa 简写,避免多处 strconv 重复。
func itoa(n int) string { return strconv.Itoa(n) }

// SendTestWecomApp 向指定用户发企微应用消息测试(凭证取已保存的 sys_auth_config;
// redirectBase 为前端当前表单值,未保存也可测跳转,空则降级纯文本)。跳转目标为网关看板。
func (s *NotifySendService) SendTestWecomApp(ctx context.Context, userId int64, redirectBase string) error {
	var social system.SysSocial
	if err := global.OPS_DB.WithContext(ctx).
		Where("user_id = ? AND source = ?", userId, "wecom").
		First(&social).Error; err != nil {
		return errors.New("该用户未绑定企业微信(需先经企微扫码登录)")
	}
	client := wecomClientFromCfg((&AuthConfigService{}).Current(ctx))
	if client.CorpID == "" || client.CorpSecret == "" || client.AgentID == 0 {
		return errWecomNotConfigured
	}
	card := utils.WecomTextCard{
		Title:       "devops-admin 测试消息",
		Description: "收到本条消息说明企业微信应用消息推送配置正确。",
		Btntxt:      "前往查看",
	}
	if redirectBase != "" {
		card.URL = strings.TrimSuffix(redirectBase, "/") + "/gateway"
	}
	return client.SendMessage(ctx, []string{social.OpenId}, card)
}

// SendTestWecomBot 用当前表单 webhook 发群机器人测试消息(未保存也可测)。
func (s *NotifySendService) SendTestWecomBot(ctx context.Context, webhookURL string) error {
	return (&utils.WecomClient{}).SendBotMessage(ctx, webhookURL,
		"## devops-admin 测试消息\n收到本条消息说明企业微信群机器人配置正确。")
}
