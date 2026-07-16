package system

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hllkk/devops-admin/server/model/system"
	sysSvc "github.com/hllkk/devops-admin/server/service/system"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const initOrderSysSetting = sysSvc.InitOrderSystem + 4

type initSysSetting struct{}

func init() { sysSvc.RegisterInit(initOrderSysSetting, &initSysSetting{}) }

// MigrateTable 创建 sys_setting 表（建表职责归属于本 initializer，不再隐式依赖全量兜底）。
func (i *initSysSetting) MigrateTable(ctx context.Context) (context.Context, error) {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return ctx, err
	}
	return ctx, db.AutoMigrate(&system.SysSetting{})
}

func (i *initSysSetting) TableCreated(ctx context.Context) bool {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return false
	}
	return db.Migrator().HasTable(&system.SysSetting{})
}

func (i *initSysSetting) InitializerName() string { return "sys_setting" }

// InitializeData 写入默认 general / security 配置（按 name upsert，已存在则不覆盖）。
func (i *initSysSetting) InitializeData(ctx context.Context) (context.Context, error) {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return ctx, err
	}

	general := system.GeneralSettings{
		SystemName:                "devops-admin",
		SystemDescription:         "企业运维管理平台",
		EnableVerifyCode:          false,
		VerifyCodeType:            "click",
		VerifyCodeLen:             4,
		VerifyCodeExp:             5,
		VerifyCodeTokenExp:        5,
		VerifyInaccuracy:          40,
		LoginLogRetentionDays:     90,
		OperationLogRetentionDays: 90,
	}
	security := system.SecuritySettings{
		PasswordMinLength:        8,
		PasswordRequireUppercase: false,
		PasswordRequireLowercase: true,
		PasswordRequireDigit:     true,
		PasswordRequireSpecial:   true,
		LoginFailLockCount:       5,
		LoginFailLockTime:        30,
		IpValidationEnabled:      false,
		IpValidationMode:         "blacklist",
	}

	if err := saveDefaultSetting(db, "general", general); err != nil {
		return ctx, fmt.Errorf("%s 数据初始化失败: %w", i.InitializerName(), err)
	}
	if err := saveDefaultSetting(db, "security", security); err != nil {
		return ctx, fmt.Errorf("%s 数据初始化失败: %w", i.InitializerName(), err)
	}
	return ctx, nil
}

func (i *initSysSetting) DataInserted(ctx context.Context) bool {
	db, err := sysSvc.DBFromCtx(ctx)
	if err != nil {
		return false
	}
	var count int64
	db.Model(&system.SysSetting{}).Count(&count)
	return count > 0
}

// saveDefaultSetting 以 name 为唯一键 upsert；冲突时仅更新 value 与 update_time。
func saveDefaultSetting(db *gorm.DB, name string, data interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化 %s 配置失败: %w", name, err)
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "update_time"}),
	}).Create(&system.SysSetting{
		Name:  name,
		Value: string(jsonBytes),
	}).Error
}
