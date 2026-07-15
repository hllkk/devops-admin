package system

import (
	"encoding/json"
	"errors"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SettingService 系统设置服务：读取聚合配置、整体保存、派生公开配置。
// 纯业务层：不依赖 gin.Context，不引入跨包回调与缓存副作用。
type SettingService struct{}

// ErrDBNotInitialized 数据库未初始化。
var ErrDBNotInitialized = errors.New("项目未初始化")

// GetSystemSettings 获取系统设置：全表扫描 sys_setting，按 name 反序列化各分类到聚合 DTO。
func (s *SettingService) GetSystemSettings() (settings system.SystemSettings, err error) {
	if global.OPS_DB == nil {
		return settings, ErrDBNotInitialized
	}
	var rows []system.SysSetting
	if err = global.OPS_DB.Find(&rows).Error; err != nil {
		return
	}
	for _, row := range rows {
		switch row.Name {
		case "general":
			var v system.GeneralSettings
			if e := json.Unmarshal([]byte(row.Value), &v); e == nil {
				settings.General = &v
			}
		case "security":
			var v system.SecuritySettings
			if e := json.Unmarshal([]byte(row.Value), &v); e == nil {
				settings.Security = &v
			}
		case "authentication":
			var v system.AuthenticationSettings
			if e := json.Unmarshal([]byte(row.Value), &v); e == nil {
				settings.Authentication = &v
			}
		case "ldap":
			var v system.LdapSettings
			if e := json.Unmarshal([]byte(row.Value), &v); e == nil {
				settings.Ldap = &v
			}
		case "notify":
			var v system.NotifySettings
			if e := json.Unmarshal([]byte(row.Value), &v); e == nil {
				settings.Notify = &v
			}
		case "disk":
			var v system.DiskSettings
			if e := json.Unmarshal([]byte(row.Value), &v); e == nil {
				settings.Disk = &v
			}
		}
	}
	return
}

// GetPublicSystemSettings 获取公开系统设置：从完整配置派生登录页所需字段（脱敏，不含密钥）。
func (s *SettingService) GetPublicSystemSettings() (system.PublicSystemSettings, error) {
	settings, err := s.GetSystemSettings()
	if err != nil {
		return system.PublicSystemSettings{}, err
	}
	public := system.PublicSystemSettings{}
	if settings.General != nil {
		g := settings.General
		public.SystemName = g.SystemName
		public.SystemDescription = g.SystemDescription
		public.LogoUrl = g.LogoUrl
		public.FaviconUrl = g.FaviconUrl
		public.EnableVerifyCode = g.EnableVerifyCode
		public.VerifyCodeType = g.VerifyCodeType
		public.VerifyCodeLen = g.VerifyCodeLen
		public.VerifyCodeExp = g.VerifyCodeExp
		public.VerifyCodeTokenExp = g.VerifyCodeTokenExp
		public.VerifyInaccuracy = g.VerifyInaccuracy
	}
	if settings.Authentication != nil {
		if settings.Authentication.Wecom != nil && settings.Authentication.Wecom.EnableWecom {
			public.EnableWecom = true
		}
		if settings.Authentication.Wechat != nil && settings.Authentication.Wechat.EnableWechat {
			public.EnableWechat = true
		}
		if settings.Authentication.Gitee != nil && settings.Authentication.Gitee.EnableGitee {
			public.EnableGitee = true
		}
		if settings.Authentication.Github != nil && settings.Authentication.Github.EnableGithub {
			public.EnableGithub = true
		}
	}
	return public, nil
}

// UpdateSystemSettings 更新系统设置：事务内按分类 upsert（name 冲突时更新 value 与 update_time），nil 分类跳过。
func (s *SettingService) UpdateSystemSettings(settings system.SystemSettings) error {
	if global.OPS_DB == nil {
		return ErrDBNotInitialized
	}
	return global.OPS_DB.Transaction(func(tx *gorm.DB) error {
		save := func(name string, data interface{}) error {
			if data == nil {
				return nil
			}
			jsonBytes, err := json.Marshal(data)
			if err != nil {
				return err
			}
			return tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoUpdates: clause.AssignmentColumns([]string{"value", "update_time"}),
			}).Create(&system.SysSetting{
				Name:  name,
				Value: string(jsonBytes),
			}).Error
		}
		if err := save("general", settings.General); err != nil {
			return err
		}
		if err := save("security", settings.Security); err != nil {
			return err
		}
		if err := save("authentication", settings.Authentication); err != nil {
			return err
		}
		if err := save("ldap", settings.Ldap); err != nil {
			return err
		}
		if err := save("notify", settings.Notify); err != nil {
			return err
		}
		if err := save("disk", settings.Disk); err != nil {
			return err
		}
		return nil
	})
}
