package response

import (
	"github.com/hllkk/devops-admin/server/model/gateway"
)

// CredentialView 凭证出网视图：模型本体(credential_values 密文列 json:"-" 不出网) +
// 解密后掩码的 credentialValues(仅敏感 key 掩码，api_base 等非敏感值明文)。
type CredentialView struct {
	gateway.Credential
	CredentialValues map[string]any `json:"credentialValues"` // 凭证键值(敏感值已掩码,如 sk-ab****cdef)
}

// ResyncResult 手动重同步 LiteLLM 凭证投影的结果汇总。
type ResyncResult struct {
	Total   int      `json:"total"`   // 参与比对的凭证总数
	Pushed  int      `json:"pushed"`  // 实际推送(新建或更新)数
	Skipped int      `json:"skipped"` // 投影一致跳过数
	Failed  []string `json:"failed"`  // 失败凭证名列表(解密/推送失败，不中断整体)
}
