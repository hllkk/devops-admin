package system

import "github.com/hllkk/devops-admin/server/global"

// SysLdapConfig LDAP 配置(单行表 id=1,启动加载入内存缓存,保存即热更新;对齐前端 LdapSettingConfig)
//
// 字段分三段(连接配置/属性映射/用户策略),json tag 对齐前端:
//   - 连接配置:host/port/useSSL/bindDN/bindPass/baseDN/filter
//   - 属性映射:attrUsername/attrNickname/attrEmail
//   - 用户策略:autoCreate
//
// 登录链路:LDAP 开关 enabled=true 时,本地用户不存在或密码不匹配则走 LDAP 绑定校验。
type SysLdapConfig struct {
	global.OPS_MODEL
	// 连接配置
	Enabled bool   `json:"enabled" gorm:"default:false;comment:是否启用LDAP"`
	Host    string `json:"host" gorm:"default:localhost;comment:LDAP服务器地址"`
	Port    int    `json:"port" gorm:"default:389;comment:LDAP端口"`
	UseSSL  bool   `json:"useSSL" gorm:"default:false;comment:是否LDAPS"`
	BindDN  string `json:"bindDN" gorm:"comment:管理员绑定DN"`
	BindPass string `json:"bindPass" gorm:"comment:管理员绑定密码"`
	BaseDN  string `json:"baseDN" gorm:"comment:用户搜索BaseDN"`
	Filter  string `json:"filter" gorm:"comment:用户过滤器 %%s 认证时替换为登录用户名"`
	// 属性映射
	AttrUsername string `json:"attrUsername" gorm:"default:uid;comment:用户名属性"`
	AttrNickname string `json:"attrNickname" gorm:"default:cn;comment:昵称属性"`
	AttrEmail    string `json:"attrEmail" gorm:"default:mail;comment:邮箱属性"`
	// 用户策略
	AutoCreate bool `json:"autoCreate" gorm:"default:false;comment:LDAP认证通过后自动创建本地用户"`
}

func (SysLdapConfig) TableName() string {
	return "sys_ldap_config"
}

// DefaultLdapConfig 返回默认 LDAP 配置,调用方负责设 id=1
func DefaultLdapConfig() SysLdapConfig {
	return SysLdapConfig{
		Enabled:      false,
		Host:         "localhost",
		Port:         389,
		UseSSL:       false,
		BindDN:       "",
		BindPass:     "",
		BaseDN:       "",
		Filter:       "(uid=%s)",
		AttrUsername: "uid",
		AttrNickname: "cn",
		AttrEmail:    "mail",
		AutoCreate:   false,
	}
}
