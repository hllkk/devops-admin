package response

import "github.com/hllkk/devops-admin/server/model/gateway"

// LlmLogView 用量日志出网视图：原始日志 + 归因实体可读名回填(用户/密钥/部署)。
// metadata 原始元数据不出网(仅排障接口按需透出)。
type LlmLogView struct {
	gateway.LlmLog
	UserName       string `json:"userName"`       // 归因用户昵称(未归因为空)
	AiKeyName      string `json:"aiKeyName"`      // 归因密钥名(未归因为空)
	DeploymentName string `json:"deploymentName"` // 归因部署名(未归因为空)
}
