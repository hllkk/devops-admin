package gateway

import (
	"github.com/hllkk/devops-admin/server/global"
	"gorm.io/datatypes"
)

// Credential 凭证（认证载体，同步 LiteLLM /credentials 的 credential_name 键）。
// credential_values 为 AES-256-GCM 加密后的 JSON 串落 text 列（规避 AIHelms 明文落库的坑），
// 出网一律解密后仅对敏感 key 掩码（api_base 等非敏感值明文回显）。
// provider_id 纯逻辑关联 gateway_provider（0=未关联），不建外键约束，删除保护在 service 层校验。
type Credential struct {
	global.OPS_AUDIT_MODEL
	CredentialId     int64          `json:"credentialId,string" gorm:"primarykey;comment:凭证ID"`                     // 凭证ID(雪花)
	CredentialName   string         `json:"credentialName" gorm:"size:128;index;comment:凭证名称(LiteLLM键,服务层查重,软删不建唯一索引)"` // 凭证名称(全局唯一,对应 LiteLLM credential_name)
	ProviderId       int64          `json:"providerId,string" gorm:"index;comment:关联供应商ID(0=未关联)"`                // 关联供应商ID(雪花,纯逻辑关联)
	CredentialValues string         `json:"-" gorm:"type:text;comment:凭证键值(AES-256-GCM密文JSON,不序列化出网)"`         // 凭证键值密文(api_key/api_base等)
	CredentialInfo   datatypes.JSON `json:"credentialInfo" gorm:"comment:非敏感元数据(format等,JSONB)" swaggertype:"object"` // 凭证元数据(format:openai/anthropic等)
	LitellmSynced    bool           `json:"litellmSynced" gorm:"default:false;comment:是否已同步LiteLLM"`              // LiteLLM投影同步状态位
	IsActive         bool           `json:"isActive" gorm:"default:true;comment:是否启用"`                      // 是否启用
	Description      string         `json:"description" gorm:"type:text;comment:描述"`                       // 描述
}

// 凭证协议格式(credential_info.format 取值)
const (
	CredentialFormatOpenai    = "openai"    // OpenAI 兼容格式(默认)
	CredentialFormatAnthropic = "anthropic" // Anthropic 原生格式
	CredentialFormatLmstudio  = "lmstudio"  // LM Studio 本地推理(OpenAI 兼容,前缀走差异表 openai+补/v1)
	CredentialFormatOllama    = "ollama"    // Ollama 本地推理(自有 API,前缀 ollama,无需 api_key)
)

func (Credential) TableName() string {
	return "gateway_credential"
}
