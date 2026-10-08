package response

import (
	"github.com/hllkk/devops-admin/server/model/server"
)

// CredentialView 凭据出网视图：模型本体(credential_values 密文列 json:"-" 不出网) +
// 解密后掩码的 credentialValues(仅敏感键掩码，username 等非敏感值明文)。
type CredentialView struct {
	server.Credential
	CredentialValues map[string]string `json:"credentialValues"` // 凭据键值(敏感值已掩码)
}

// CredentialOptions 凭据下拉选项(资产表单选择 SSH 凭据用；type 过滤，仅启用中)。
type CredentialOptions struct {
	CredentialId   int64  `json:"credentialId,string" example:"1"`   // 凭据ID
	CredentialName string `json:"credentialName" example:"root-key"` // 凭据名称
	CredentialType string `json:"credentialType" example:"ssh"`      // 凭据类型
}
