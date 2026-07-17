package system

import "github.com/hllkk/devops-admin/server/global"

// SysGeneralConfig 通用配置(单行表 固定 id=1,系统设置 general 段;对齐前端 GeneralSettingConfig)
// 启动加载入内存缓存,保存即热更新。无审计字段(配置表),嵌入 OPS_MODEL 复用 id 单行。
type SysGeneralConfig struct {
	global.OPS_MODEL
	SystemName                string `json:"systemName" gorm:"comment:系统名称"`                       // 系统名称
	SystemDescription         string `json:"systemDescription" gorm:"comment:系统描述"`                 // 系统描述
	LogoUrl                   string `json:"logoUrl" gorm:"comment:Logo地址"`                        // Logo地址
	FaviconUrl                string `json:"faviconUrl" gorm:"comment:Favicon地址"`                   // Favicon地址
	UserDefaultPassword       string `json:"userDefaultPassword" gorm:"comment:用户默认密码"`            // 用户默认密码
	UserDefaultRole           string `json:"userDefaultRole" gorm:"comment:用户默认角色"`               // 用户默认角色
	EnableVerifyCode          bool   `json:"enableVerifyCode" gorm:"default:true;comment:是否开启验证码"` // 是否开启验证码
	VerifyCodeType            string `json:"verifyCodeType" gorm:"default:image;comment:验证码类型"`    // 验证码类型
	VerifyCodeLen             int    `json:"verifyCodeLen" gorm:"default:4;comment:验证码长度"`         // 验证码长度
	VerifyCodeExp             int    `json:"verifyCodeExp" gorm:"default:300;comment:验证码过期(秒)"`   // 验证码过期时间(秒)
	VerifyCodeTokenExp        int    `json:"verifyCodeTokenExp" gorm:"default:600;comment:验证码token过期(秒)"` // 验证码 token 过期(秒)
	VerifyInaccuracy          int    `json:"verifyInaccuracy" gorm:"default:0;comment:验证码容错数"`     // 验证码容错数
	LoginLogRetentionDays     int    `json:"loginLogRetentionDays" gorm:"default:30;comment:登录日志保留天数"`  // 登录日志保留天数
	OperationLogRetentionDays int    `json:"operationLogRetentionDays" gorm:"default:30;comment:操作日志保留天数"` // 操作日志保留天数
}

func (SysGeneralConfig) TableName() string {
	return "sys_general_config"
}

// DefaultGeneralConfig 默认通用配置(调用方负责设 id=1)
func DefaultGeneralConfig() SysGeneralConfig {
	return SysGeneralConfig{
		SystemName:                "DevOps Admin",
		SystemDescription:         "DevOps 管理平台",
		EnableVerifyCode:          true,
		VerifyCodeType:            "image",
		VerifyCodeLen:             4,
		VerifyCodeExp:             300,
		VerifyCodeTokenExp:        600,
		VerifyInaccuracy:          0,
		LoginLogRetentionDays:     30,
		OperationLogRetentionDays: 30,
	}
}
