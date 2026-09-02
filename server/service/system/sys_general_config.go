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

type GeneralConfigService struct{}

// generalConfigCache 进程内当前生效配置 热读
var generalConfigCache atomic.Value

func setGeneralConfigCache(cfg system.SysGeneralConfig) {
	generalConfigCache.Store(cfg)
}

func getGeneralConfigCache() system.SysGeneralConfig {
	if v := generalConfigCache.Load(); v != nil {
		return v.(system.SysGeneralConfig)
	}
	return system.SysGeneralConfig{}
}

// FillDefaultRoleId 将 DefaultRoleId=0(未配置)的常规配置回填为内置「普通用户」角色(role_key=user)的ID。
// 0 会让企微扫码等自动建号必然失败,视为未配置而非合法取值;管理员显式配置过(非0)不覆盖;
// 角色查不到(初始化顺序异常/角色被删)时保持 0,由建号路径报「默认角色未配置」兜底。
// db 由调用方传入: init 向导传 ctx db,运行期传 global.OPS_DB。
func FillDefaultRoleId(db *gorm.DB, cfg *system.SysGeneralConfig) {
	if cfg.DefaultRoleId != 0 {
		return
	}
	var role system.SysRole
	if err := db.Where("role_key = ?", "user").First(&role).Error; err == nil {
		cfg.DefaultRoleId = role.RoleId
	}
}

// Get 读取单行配置 不存在则按默认创建并返回
func (s *GeneralConfigService) Get(ctx context.Context) (system.SysGeneralConfig, error) {
	var cfg system.SysGeneralConfig
	// 系统尚未初始化(未走 init 向导)或连库失败时 global.OPS_DB 为 nil
	// 此时返回默认配置并带错误: 调用方 Current 据此不写缓存
	// 待数据库就绪后再惰性加载真实行 同时避免对 nil 的 *gorm.DB 解引用导致 panic
	if global.OPS_DB == nil {
		return system.DefaultGeneralConfig(), errors.New("数据库未初始化")
	}
	db := global.OPS_DB.WithContext(ctx)
	err := db.Where("id = ?", 1).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cfg = system.DefaultGeneralConfig()
		cfg.ID = 1
		FillDefaultRoleId(db, &cfg)
		if err = db.Create(&cfg).Error; err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	// 存量库自愈: 初始化早于本逻辑的库 default_role_id=0,读时回填一次落库
	// CAS 条件限定仍为 0,避免与并发的管理员保存互踩;失败仅记日志,返回值已回填不影响本次使用
	FillDefaultRoleId(db, &cfg)
	if cfg.DefaultRoleId != 0 {
		if e := db.Model(&system.SysGeneralConfig{}).
			Where("id = ? AND default_role_id = 0", 1).
			Update("default_role_id", cfg.DefaultRoleId).Error; e != nil {
			logger.WithCtx(ctx).Mod("biz").Error("默认角色回填失败: " + e.Error())
		}
	}
	return cfg, nil
}

// Set 持久化配置 刷新内存缓存(常规配置无副作用,仅保留 OPS_MODEL 基座字段)
func (s *GeneralConfigService) Set(ctx context.Context, cfg system.SysGeneralConfig) error {
	prev, err := s.Get(ctx)
	if err != nil {
		return err
	}
	cfg.OPS_MODEL = prev.OPS_MODEL
	if err = global.OPS_DB.WithContext(ctx).Save(&cfg).Error; err != nil {
		return err
	}
	setGeneralConfigCache(cfg)
	return nil
}

// Current 返回内存缓存当前配置 未加载则惰性 Get
func (s *GeneralConfigService) Current(ctx context.Context) system.SysGeneralConfig {
	if v := generalConfigCache.Load(); v != nil {
		return v.(system.SysGeneralConfig)
	}
	cfg, err := s.Get(ctx)
	if err == nil {
		setGeneralConfigCache(cfg)
	}
	return cfg
}

// LoadAll 启动时加载配置入内存缓存
func (s *GeneralConfigService) LoadAll(ctx context.Context) {
	cfg, err := s.Get(ctx)
	if err != nil {
		logger.WithCtx(ctx).Mod("biz").Error("加载通用配置失败!")
		return
	}
	setGeneralConfigCache(cfg)
}
