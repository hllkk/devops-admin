package system

import "github.com/hllkk/devops-admin/server/global"

// SysGeneralConfig 通用配置(单行表 固定 id=1,系统设置 general 段;对齐前端 GeneralSettingConfig)
// 启动加载入内存缓存,保存即热更新。无审计字段(配置表),嵌入 OPS_MODEL 复用 id 单行。
// 验证码配置统一落在 SysSecurityConfig.Captcha* 段,本表不含 verifyCode* 字段。
type SysGeneralConfig struct {
	global.OPS_MODEL
	SystemName                string `json:"systemName" gorm:"default:DevOps Admin;comment:系统名称"`
	SystemDescription         string `json:"systemDescription" gorm:"default:DevOps 管理平台;comment:系统描述"`
	LogoUrl                   string `json:"logoUrl" gorm:"comment:Logo地址"`
	FaviconUrl                string `json:"faviconUrl" gorm:"comment:Favicon地址"`
	LoginLogRetentionDays     int    `json:"loginLogRetentionDays" gorm:"default:30;comment:登录日志保留天数"`
	OperationLogRetentionDays int    `json:"operationLogRetentionDays" gorm:"default:30;comment:操作日志保留天数"`
	DefaultPassword           string `json:"defaultPassword" gorm:"default:User@1234;comment:导入/重置用户的默认密码(建议满足密码复杂度)"` // 导入新建/复活用户的初始密码;首登强制改密
	DefaultRoleId             int64  `json:"defaultRoleId,string" gorm:"default:0;comment:默认角色ID(企微扫码等自动建号用,初始化指向普通用户,0视为未配置则建号失败)"`
}

func (SysGeneralConfig) TableName() string {
	return "sys_general_config"
}

// DefaultGeneralConfig 默认通用配置(调用方负责设 id=1)
func DefaultGeneralConfig() SysGeneralConfig {
	return SysGeneralConfig{
		SystemName:                "DevOps Admin",
		SystemDescription:         "DevOps 管理平台",
		LoginLogRetentionDays:     30,
		OperationLogRetentionDays: 30,
		DefaultPassword:           "User@1234",
	}
}
