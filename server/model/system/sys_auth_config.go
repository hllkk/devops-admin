package system

import "github.com/hllkk/devops-admin/server/global"

// SysAuthConfig 认证配置（单行表 id=1，启动加载入内存缓存，保存即热更新；对齐前端 AuthSettingConfig）
//
// 字段分三段（账号功能/第三方登录 OAuth2/企业协作），json tag 对齐前端：
//   - 账号功能：registerEnabled/resetPwdEnabled
//   - 第三方登录：wecom*/wechat*/gitee*/github*/dingtalk* 每组独立 enabled/clientId/clientSecret/callbackUrl
//
// 登录链路：各 provider.enabled=true 时，登录页展示对应第三方登录按钮，点击后跳转 OAuth2 授权。
// PublicSetting 只暴露 enabled 开关，不泄露密钥。
type SysAuthConfig struct {
	global.OPS_MODEL
	// 账号功能
	RegisterEnabled bool `json:"registerEnabled" gorm:"default:false;comment:是否开放注册"`
	ResetPwdEnabled bool `json:"resetPwdEnabled" gorm:"default:false;comment:是否开放找回密码"`
	// 企业微信
	WecomEnabled           bool   `json:"wecomEnabled" gorm:"default:false;comment:企业微信登录开关"`
	WecomCorpId            string `json:"wecomCorpId" gorm:"comment:企业微信 CorpId（企业ID）"`
	WecomAgentId           int    `json:"wecomAgentId" gorm:"default:0;comment:企业微信应用 AgentId"`
	WecomClientId          string `json:"wecomClientId" gorm:"comment:企业微信 ClientId/CorpId"`
	WecomClientSecret      string `json:"wecomClientSecret" gorm:"comment:企业微信 ClientSecret/CorpSecret（应用 Secret）"`
	WecomCallbackUrl       string `json:"wecomCallbackUrl" gorm:"comment:企业微信 OAuth2 回调地址"`
	WecomDomainFileName    string `json:"wecomDomainFileName" gorm:"comment:企业微信可信域名校验文件名(WW_verify_*.txt)"`
	WecomDomainFileContent string `json:"wecomDomainFileContent" gorm:"comment:企业微信可信域名校验文件内容"`
	// 微信开放平台
	WechatEnabled      bool   `json:"wechatEnabled" gorm:"default:false;comment:微信开放平台登录开关"`
	WechatClientId     string `json:"wechatClientId" gorm:"comment:微信开放平台 AppId"`
	WechatClientSecret string `json:"wechatClientSecret" gorm:"comment:微信开放平台 AppSecret"`
	WechatCallbackUrl  string `json:"wechatCallbackUrl" gorm:"comment:微信开放平台 OAuth2 回调地址"`
	// Gitee
	GiteeEnabled      bool   `json:"giteeEnabled" gorm:"default:false;comment:Gitee 登录开关"`
	GiteeClientId     string `json:"giteeClientId" gorm:"comment:Gitee ClientId"`
	GiteeClientSecret string `json:"giteeClientSecret" gorm:"comment:Gitee ClientSecret"`
	GiteeCallbackUrl  string `json:"giteeCallbackUrl" gorm:"comment:Gitee OAuth2 回调地址"`
	// GitHub
	GithubEnabled      bool   `json:"githubEnabled" gorm:"default:false;comment:GitHub 登录开关"`
	GithubClientId     string `json:"githubClientId" gorm:"comment:GitHub ClientId"`
	GithubClientSecret string `json:"githubClientSecret" gorm:"comment:GitHub ClientSecret"`
	GithubCallbackUrl  string `json:"githubCallbackUrl" gorm:"comment:GitHub OAuth2 回调地址"`
	// 钉钉
	DingtalkEnabled      bool   `json:"dingtalkEnabled" gorm:"default:false;comment:钉钉登录开关"`
	DingtalkClientId     string `json:"dingtalkClientId" gorm:"comment:钉钉 AppKey/ClientId"`
	DingtalkClientSecret string `json:"dingtalkClientSecret" gorm:"comment:钉钉 AppSecret/ClientSecret"`
	DingtalkCallbackUrl  string `json:"dingtalkCallbackUrl" gorm:"comment:钉钉 OAuth2 回调地址"`
}

func (SysAuthConfig) TableName() string {
	return "sys_auth_config"
}

// DefaultAuthConfig 返回默认认证配置，调用方负责设 id=1
func DefaultAuthConfig() SysAuthConfig {
	return SysAuthConfig{
		RegisterEnabled: false,
		ResetPwdEnabled: false,
	}
}
