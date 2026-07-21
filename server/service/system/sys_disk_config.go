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

type DiskConfigService struct{}

// diskConfigCache 进程内当前生效配置 热读
var diskConfigCache atomic.Value

func setDiskConfigCache(cfg system.SysDiskConfig) {
	diskConfigCache.Store(cfg)
}

func getDiskConfigCache() system.SysDiskConfig {
	if v := diskConfigCache.Load(); v != nil {
		return v.(system.SysDiskConfig)
	}
	return system.SysDiskConfig{}
}

// Get 读取单行配置,不存在则按默认创建并返回
func (s *DiskConfigService) Get(ctx context.Context) (system.SysDiskConfig, error) {
	var cfg system.SysDiskConfig
	if global.OPS_DB == nil {
		return system.DefaultDiskConfig(), errors.New("数据库未初始化")
	}
	err := global.OPS_DB.WithContext(ctx).Where("id = ?", 1).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cfg = system.DefaultDiskConfig()
		cfg.ID = 1
		if err = global.OPS_DB.WithContext(ctx).Create(&cfg).Error; err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	return cfg, err
}

// Set 持久化配置,刷新内存缓存
func (s *DiskConfigService) Set(ctx context.Context, cfg system.SysDiskConfig) error {
	prev, err := s.Get(ctx)
	if err != nil {
		return err
	}
	cfg.OPS_MODEL = prev.OPS_MODEL
	if err = global.OPS_DB.WithContext(ctx).Save(&cfg).Error; err != nil {
		return err
	}
	setDiskConfigCache(cfg)
	return nil
}

// Current 返回内存缓存当前配置,未加载则惰性 Get
func (s *DiskConfigService) Current(ctx context.Context) system.SysDiskConfig {
	if v := diskConfigCache.Load(); v != nil {
		return v.(system.SysDiskConfig)
	}
	cfg, err := s.Get(ctx)
	if err == nil {
		setDiskConfigCache(cfg)
	}
	return cfg
}

// LoadAll 启动时加载配置入内存缓存
func (s *DiskConfigService) LoadAll(ctx context.Context) {
	cfg, err := s.Get(ctx)
	if err != nil {
		logger.WithCtx(ctx).Mod("biz").Error("加载网盘配置失败!")
		return
	}
	setDiskConfigCache(cfg)
}
