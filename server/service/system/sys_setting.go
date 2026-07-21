package system

import (
	"context"

	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
)

type SettingService struct{}

// Get 聚合读取 {general, security}:分别从两张单行配置表读取,拼装为前端 Api.System.Setting
func (s *SettingService) Get(ctx context.Context) (systemReq.SettingConfig, error) {
	general, err := (&GeneralConfigService{}).Get(ctx)
	if err != nil {
		return systemReq.SettingConfig{}, err
	}
	security, err := (&SecurityConfigService{}).Get(ctx)
	if err != nil {
		return systemReq.SettingConfig{}, err
	}
	return systemReq.SettingConfig{General: &general, Security: &security}, nil
}

// Set 聚合保存:按段落非空分发到对应配置表,各自刷内存缓存。
// 注意:非跨表事务,general 成功而 security 失败会部分生效(配置类数据可接受,前端重试即可)
func (s *SettingService) Set(ctx context.Context, req systemReq.SettingConfig) error {
	if req.General != nil {
		if err := (&GeneralConfigService{}).Set(ctx, *req.General); err != nil {
			return err
		}
	}
	if req.Security != nil {
		if err := (&SecurityConfigService{}).Set(ctx, *req.Security); err != nil {
			return err
		}
	}
	return nil
}

// GetPublic 脱敏公开配置(登录页,免鉴权):常规配置的系统信息 + 安全配置的验证码段。
// 读 Current 内存缓存(惰性加载,DB 未就绪时返回默认配置),不返回 error,保证登录页始终可用。
func (s *SettingService) GetPublic(ctx context.Context) systemReq.PublicSetting {
	general := (&GeneralConfigService{}).Current(ctx)
	security := (&SecurityConfigService{}).Current(ctx)
	return systemReq.PublicSetting{
		SystemName:        general.SystemName,
		SystemDescription: general.SystemDescription,
		LogoUrl:           general.LogoUrl,
		FaviconUrl:        general.FaviconUrl,
		CaptchaEnabled:    security.CaptchaEnabled,
		CaptchaType:       security.CaptchaType,
		CaptchaOpen:       security.CaptchaOpen,
		KeyLong:           security.KeyLong,
		ImgWidth:          security.ImgWidth,
		ImgHeight:         security.ImgHeight,
	}
}
