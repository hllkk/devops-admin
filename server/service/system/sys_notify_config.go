package system

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"sync/atomic"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils/logger"
	"gorm.io/gorm"
)

type NotifyConfigService struct{}

// notifyConfigCache 进程内当前生效配置 热读
var notifyConfigCache atomic.Value

func setNotifyConfigCache(cfg system.SysNotifyConfig) {
	notifyConfigCache.Store(cfg)
}

func getNotifyConfigCache() system.SysNotifyConfig {
	if v := notifyConfigCache.Load(); v != nil {
		return v.(system.SysNotifyConfig)
	}
	return system.SysNotifyConfig{}
}

// Get 读取单行配置,不存在则按默认创建并返回
func (s *NotifyConfigService) Get(ctx context.Context) (system.SysNotifyConfig, error) {
	var cfg system.SysNotifyConfig
	if global.OPS_DB == nil {
		return system.DefaultNotifyConfig(), errors.New("数据库未初始化")
	}
	err := global.OPS_DB.WithContext(ctx).Where("id = ?", 1).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cfg = system.DefaultNotifyConfig()
		cfg.ID = 1
		if err = global.OPS_DB.WithContext(ctx).Create(&cfg).Error; err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	return cfg, err
}

// Set 持久化配置,刷新内存缓存
func (s *NotifyConfigService) Set(ctx context.Context, cfg system.SysNotifyConfig) error {
	prev, err := s.Get(ctx)
	if err != nil {
		return err
	}
	cfg.OPS_MODEL = prev.OPS_MODEL
	if err = global.OPS_DB.WithContext(ctx).Save(&cfg).Error; err != nil {
		return err
	}
	setNotifyConfigCache(cfg)
	return nil
}

// Current 返回内存缓存当前配置,未加载则惰性 Get
func (s *NotifyConfigService) Current(ctx context.Context) system.SysNotifyConfig {
	if v := notifyConfigCache.Load(); v != nil {
		return v.(system.SysNotifyConfig)
	}
	cfg, err := s.Get(ctx)
	if err == nil {
		setNotifyConfigCache(cfg)
	}
	return cfg
}

// LoadAll 启动时加载配置入内存缓存
func (s *NotifyConfigService) LoadAll(ctx context.Context) {
	cfg, err := s.Get(ctx)
	if err != nil {
		logger.WithCtx(ctx).Mod("biz").Error("加载通知配置失败!")
		return
	}
	setNotifyConfigCache(cfg)
}

// SendTestEmail 使用给定配置发送测试邮件到 testTo 地址。
// cfg 为前端传入的当前表单值（尚未持久化），不读 DB。
func (s *NotifyConfigService) SendTestEmail(cfg system.SysNotifyConfig, testTo string) error {
	if cfg.EmailHost == "" {
		return errors.New("SMTP 服务器地址不能为空")
	}
	if testTo == "" {
		return errors.New("收件人地址不能为空")
	}

	addr := net.JoinHostPort(cfg.EmailHost, fmt.Sprintf("%d", cfg.EmailPort))
	auth := smtp.PlainAuth("", cfg.EmailUsername, cfg.EmailPassword, cfg.EmailHost)

	msg := s.buildTestMessage(cfg, testTo)

	switch strings.ToLower(cfg.EmailSSLMode) {
	case "ssl":
		return s.sendWithTLS(addr, auth, cfg.EmailUsername, []string{testTo}, msg)
	case "starttls":
		return s.sendWithStartTLS(addr, cfg.EmailHost, auth, cfg.EmailUsername, []string{testTo}, msg)
	default: // none
		return smtp.SendMail(addr, auth, cfg.EmailUsername, []string{testTo}, msg)
	}
}

func (s *NotifyConfigService) buildTestMessage(cfg system.SysNotifyConfig, testTo string) []byte {
	fromName := cfg.EmailFromName
	if fromName == "" {
		fromName = cfg.EmailFromAddr
	}
	subject := fmt.Sprintf("[%s] 测试邮件", fromName)
	from := fmt.Sprintf("%s <%s>", fromName, cfg.EmailFromAddr)

	body := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n这是一封来自 devops-admin 通知系统的测试邮件。\r\n\r\n如果您收到此邮件，说明通知配置正确。",
		from, testTo, subject)
	return []byte(body)
}

func (s *NotifyConfigService) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: strings.Split(addr, ":")[0],
	}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS 连接失败: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, tlsConfig.ServerName)
	if err != nil {
		return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 认证失败: %w", err)
		}
	}
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}
	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("设置收件人失败: %w", err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("准备邮件数据失败: %w", err)
	}
	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("写入邮件数据失败: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("关闭邮件数据失败: %w", err)
	}
	return client.Quit()
}

func (s *NotifyConfigService) sendWithStartTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("TCP 连接失败: %w", err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	defer client.Close()

	if err = client.StartTLS(&tls.Config{ServerName: host}); err != nil {
		return fmt.Errorf("STARTTLS 协商失败: %w", err)
	}

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 认证失败: %w", err)
		}
	}
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}
	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("设置收件人失败: %w", err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("准备邮件数据失败: %w", err)
	}
	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("写入邮件数据失败: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("关闭邮件数据失败: %w", err)
	}
	return client.Quit()
}
