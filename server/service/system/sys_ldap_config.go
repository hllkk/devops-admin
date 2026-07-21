package system

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils/logger"
	"gorm.io/gorm"
)

type LdapConfigService struct{}

// ldapConfigCache 进程内当前生效配置 热读
var ldapConfigCache atomic.Value

func setLdapConfigCache(cfg system.SysLdapConfig) {
	ldapConfigCache.Store(cfg)
}

func getLdapConfigCache() system.SysLdapConfig {
	if v := ldapConfigCache.Load(); v != nil {
		return v.(system.SysLdapConfig)
	}
	return system.SysLdapConfig{}
}

// Get 读取单行配置,不存在则按默认创建并返回
func (s *LdapConfigService) Get(ctx context.Context) (system.SysLdapConfig, error) {
	var cfg system.SysLdapConfig
	if global.OPS_DB == nil {
		return system.DefaultLdapConfig(), errors.New("数据库未初始化")
	}
	err := global.OPS_DB.WithContext(ctx).Where("id = ?", 1).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cfg = system.DefaultLdapConfig()
		cfg.ID = 1
		if err = global.OPS_DB.WithContext(ctx).Create(&cfg).Error; err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	return cfg, err
}

// Set 持久化配置,刷新内存缓存
func (s *LdapConfigService) Set(ctx context.Context, cfg system.SysLdapConfig) error {
	prev, err := s.Get(ctx)
	if err != nil {
		return err
	}
	cfg.OPS_MODEL = prev.OPS_MODEL
	if err = global.OPS_DB.WithContext(ctx).Save(&cfg).Error; err != nil {
		return err
	}
	setLdapConfigCache(cfg)
	return nil
}

// Current 返回内存缓存当前配置,未加载则惰性 Get
func (s *LdapConfigService) Current(ctx context.Context) system.SysLdapConfig {
	if v := ldapConfigCache.Load(); v != nil {
		return v.(system.SysLdapConfig)
	}
	cfg, err := s.Get(ctx)
	if err == nil {
		setLdapConfigCache(cfg)
	}
	return cfg
}

// LoadAll 启动时加载配置入内存缓存
func (s *LdapConfigService) LoadAll(ctx context.Context) {
	cfg, err := s.Get(ctx)
	if err != nil {
		logger.WithCtx(ctx).Mod("biz").Error("加载 LDAP 配置失败!")
		return
	}
	setLdapConfigCache(cfg)
}
