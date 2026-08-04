package system

import (
	"context"

	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
)

type SettingService struct{}

// Get 聚合读取 {general, security, ldap, notify, auth}:分别从各单行配置表读取,拼装为前端 Api.System.Setting
func (s *SettingService) Get(ctx context.Context) (systemReq.SettingConfig, error) {
	general, err := (&GeneralConfigService{}).Get(ctx)
	if err != nil {
		return systemReq.SettingConfig{}, err
	}
	security, err := (&SecurityConfigService{}).Get(ctx)
	if err != nil {
		return systemReq.SettingConfig{}, err
	}
	ldap, err := (&LdapConfigService{}).Get(ctx)
	if err != nil {
		return systemReq.SettingConfig{}, err
	}
	notify, err := (&NotifyConfigService{}).Get(ctx)
	if err != nil {
		return systemReq.SettingConfig{}, err
	}
	auth, err := (&AuthConfigService{}).Get(ctx)
	if err != nil {
		return systemReq.SettingConfig{}, err
	}
	return systemReq.SettingConfig{General: &general, Security: &security, Ldap: &ldap, Notify: &notify, Auth: &auth}, nil
}

// Set 聚合保存:按段落非空分发到对应配置表,各自刷内存缓存。
// 注意:非跨表事务,段落成功而另一段失败会部分生效(配置类数据可接受,前端重试即可)
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
	if req.Ldap != nil {
		if err := (&LdapConfigService{}).Set(ctx, *req.Ldap); err != nil {
			return err
		}
	}
	if req.Notify != nil {
		if err := (&NotifyConfigService{}).Set(ctx, *req.Notify); err != nil {
			return err
		}
	}
	if req.Auth != nil {
		if err := (&AuthConfigService{}).Set(ctx, *req.Auth); err != nil {
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
	auth := (&AuthConfigService{}).Current(ctx)
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
		RegisterEnabled:   auth.RegisterEnabled,
		ResetPwdEnabled:   auth.ResetPwdEnabled,
		WecomEnabled:      auth.WecomEnabled,
		WechatEnabled:     auth.WechatEnabled,
		GiteeEnabled:      auth.GiteeEnabled,
		GithubEnabled:     auth.GithubEnabled,
	}
}

// CurrentAuth 返回当前认证配置(从内存缓存读取，用于热路径如可信域名校验)
func (s *SettingService) CurrentAuth(ctx context.Context) system.SysAuthConfig {
	return (&AuthConfigService{}).Current(ctx)
}
