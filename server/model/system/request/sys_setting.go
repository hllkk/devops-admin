package request

import "github.com/hllkk/devops-admin/server/model/system"

// SettingConfig 系统设置聚合配置(GET/PUT /system/setting 请求与响应体,对齐前端 Api.System.Setting)
// general 段落表 sys_general_config(系统信息/账户默认/日志清理);security 段落表 sys_security_config(六段安全策略)
// general 与 security 任一可选:前端聚合页一次性提交两段,后端分发到两张配置表
type SettingConfig struct {
	General  *system.SysGeneralConfig  `json:"general,omitempty"`
	Security *system.SysSecurityConfig `json:"security,omitempty"`
}
