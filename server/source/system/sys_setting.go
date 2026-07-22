package system

import (
	"context"

	"github.com/hllkk/devops-admin/server/global"
	sysModel "github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// initSetting 系统设置聚合初始化器: 在 /initdb 流程中建并填充 6 张单行配置表。
//
// 背景: 此前 general/security/ldap/disk/notify/auth 这 6 张配置表既不在任何 initializer 的
// MigrateTable 里, 也不在 extraTables 中, 导致走 /init 向导初始化后这些表不会被创建,
// 系统设置页 GET /system/setting 报 "relation \"xxx\" does not exist"。建表此前只存在于重启路径
// initialize.RegisterTables, 形成"初始化后必须重启一次"的缺陷。本初始化器把建表与 id=1 默认数据
// 统一纳入 /initdb, 与 service 层 SettingService 聚合语义对齐: 初始化完成即数据齐全, 前端无需再提交默认值。
//
// order 取 InitOrderSystem + 100, 远离 sys_casbin..sys_notice(11..18)单链, 避免撞号(对齐 gva config 区间)。
const initOrderSetting = system.InitOrderSystem + 100

type initSetting struct{}

// auto run
func init() {
	system.RegisterInit(initOrderSetting, &initSetting{})
}

func (i *initSetting) InitializerName() string {
	return "sys_setting"
}

func (i *initSetting) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(
		&sysModel.SysGeneralConfig{},
		&sysModel.SysSecurityConfig{},
		&sysModel.SysLdapConfig{},
		&sysModel.SysDiskConfig{},
		&sysModel.SysNotifyConfig{},
		&sysModel.SysAuthConfig{},
		&sysModel.SysNotice{},
		&sysModel.SysNoticeRecord{},
	)
}

func (i *initSetting) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	// 6 张表同批迁移, 以 general 作代表探表是否存在。
	return db.Migrator().HasTable(&sysModel.SysGeneralConfig{})
}

// InitializeData 为 6 张配置表填充默认行(均固定 id=1)。
// 每张表先探 id=1 是否存在, 缺失才插入, 保证二次 /initdb 幂等(不单依赖 DataInserted 的整体探针)。
func (i *initSetting) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}

	general := sysModel.DefaultGeneralConfig()
	general.ID = 1
	if err := ensureConfigRow(db, &sysModel.SysGeneralConfig{}, &general); err != nil {
		return ctx, errors.Wrap(err, "sys_general_config 默认配置初始化失败")
	}

	security := sysModel.DefaultSecurityConfig(global.OPS_CONFIG.Captcha)
	security.ID = 1
	if err := ensureConfigRow(db, &sysModel.SysSecurityConfig{}, &security); err != nil {
		return ctx, errors.Wrap(err, "sys_security_config 默认配置初始化失败")
	}

	ldap := sysModel.DefaultLdapConfig()
	ldap.ID = 1
	if err := ensureConfigRow(db, &sysModel.SysLdapConfig{}, &ldap); err != nil {
		return ctx, errors.Wrap(err, "sys_ldap_config 默认配置初始化失败")
	}

	disk := sysModel.DefaultDiskConfig()
	disk.ID = 1
	if err := ensureConfigRow(db, &sysModel.SysDiskConfig{}, &disk); err != nil {
		return ctx, errors.Wrap(err, "sys_disk_config 默认配置初始化失败")
	}

	notify := sysModel.DefaultNotifyConfig()
	notify.ID = 1
	if err := ensureConfigRow(db, &sysModel.SysNotifyConfig{}, &notify); err != nil {
		return ctx, errors.Wrap(err, "sys_notify_config 默认配置初始化失败")
	}

	auth := sysModel.DefaultAuthConfig()
	auth.ID = 1
	if err := ensureConfigRow(db, &sysModel.SysAuthConfig{}, &auth); err != nil {
		return ctx, errors.Wrap(err, "sys_auth_config 默认配置初始化失败")
	}

	return ctx, nil
}

func (i *initSetting) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	// 6 表同批插入, 以 general 作聚合代表: general 的 id=1 存在即视为已初始化。
	if errors.Is(db.Where("id = ?", 1).First(&sysModel.SysGeneralConfig{}).Error, gorm.ErrRecordNotFound) {
		return false
	}
	return true
}

// ensureConfigRow 探测配置表 id=1 行是否存在, 缺失则插入 def(def 调用方已设 ID=1)。幂等。
func ensureConfigRow(db *gorm.DB, probe, def any) error {
	if err := db.First(probe, 1).Error; err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return db.Create(def).Error
}
