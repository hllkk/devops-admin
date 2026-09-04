package system

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	"gorm.io/gorm"
)

type AuthConfigService struct{}

// authConfigCache 进程内当前生效配置 热读
var authConfigCache atomic.Value

func setAuthConfigCache(cfg system.SysAuthConfig) {
	authConfigCache.Store(cfg)
}

func getAuthConfigCache() system.SysAuthConfig {
	if v := authConfigCache.Load(); v != nil {
		return v.(system.SysAuthConfig)
	}
	return system.SysAuthConfig{}
}

// Get 读取单行配置 不存在则按默认创建并返回
func (s *AuthConfigService) Get(ctx context.Context) (system.SysAuthConfig, error) {
	var cfg system.SysAuthConfig
	if global.OPS_DB == nil {
		return system.DefaultAuthConfig(), errors.New("数据库未初始化")
	}
	err := global.OPS_DB.WithContext(ctx).Where("id = ?", 1).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cfg = system.DefaultAuthConfig()
		cfg.ID = 1
		if err = global.OPS_DB.WithContext(ctx).Create(&cfg).Error; err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	return cfg, err
}

// Set 持久化配置 刷新内存缓存（仅保留 OPS_MODEL 基座字段）
func (s *AuthConfigService) Set(ctx context.Context, cfg system.SysAuthConfig) error {
	prev, err := s.Get(ctx)
	if err != nil {
		return err
	}
	cfg.OPS_MODEL = prev.OPS_MODEL
	// 企微可信域名校验两字段规范化：文件名 trim+仅中段自动补全、内容 trim，
	// 防粘贴带空格/丢前缀后缀导致公网校验恒 404
	cfg.WecomDomainFileName = system.NormalizeWecomDomainFileName(cfg.WecomDomainFileName)
	cfg.WecomDomainFileContent = strings.TrimSpace(cfg.WecomDomainFileContent)
	if err = global.OPS_DB.WithContext(ctx).Save(&cfg).Error; err != nil {
		return err
	}
	setAuthConfigCache(cfg)
	return nil
}

// Current 返回内存缓存当前配置 未加载则惰性 Get
func (s *AuthConfigService) Current(ctx context.Context) system.SysAuthConfig {
	if v := authConfigCache.Load(); v != nil {
		return v.(system.SysAuthConfig)
	}
	cfg, err := s.Get(ctx)
	if err == nil {
		setAuthConfigCache(cfg)
	}
	return cfg
}

// LoadAll 启动时加载配置入内存缓存
func (s *AuthConfigService) LoadAll(ctx context.Context) {
	cfg, err := s.Get(ctx)
	if err != nil {
		// 加载失败不阻塞启动，后续 Current() 会惰性重试
		return
	}
	setAuthConfigCache(cfg)
}
