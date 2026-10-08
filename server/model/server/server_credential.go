package server

import (
	"github.com/hllkk/devops-admin/server/global"
)

// Credential 服务器模块采集凭据(SSH/BMC/SNMP/数据库连接账号)。
// credential_values 落库 AES-256-GCM 加密(密钥 server.credential-key)，
// 出网仅敏感键掩码(见 service/server/credential_payload.go)。
// SSH 密码仅用于一次性部署公钥/安装 agent，不在此存储(用户决策)。
type Credential struct {
	global.OPS_AUDIT_MODEL
	CredentialId     int64  `json:"credentialId,string" gorm:"primarykey;comment:凭据ID(雪花)"`            // 凭据ID
	CredentialName   string `json:"credentialName" gorm:"index;comment:凭据名称"`                          // 凭据名称(未软删行内唯一,服务层查重)
	CredentialType   string `json:"credentialType" gorm:"size:20;index;comment:凭据类型(ssh/bmc/snmp/db)"` // 凭据类型
	CredentialValues string `json:"-" gorm:"type:text;comment:凭据键值(AES-256-GCM密文JSON,不序列化出网)"`         // 凭据键值(密文)
	Description      string `json:"description" gorm:"type:text;comment:描述"`                           // 描述
	IsActive         bool   `json:"isActive" gorm:"default:true;comment:是否启用"`                         // 是否启用
}

// 凭据类型(与前端常量严格对齐)
const (
	CredentialTypeSSH  = "ssh"  // SSH(公钥认证用户名,agent 安装/SSH 采集)
	CredentialTypeBMC  = "bmc"  // BMC(IPMI/Redfish 管理账号)
	CredentialTypeSNMP = "snmp" // SNMP(v2c community / v3 账号)
	CredentialTypeDB   = "db"   // 数据库连接账号
)

func (Credential) TableName() string {
	return "server_credential"
}
