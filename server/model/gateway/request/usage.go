package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// UsageLogSearch 用量日志分页查询(管理员视角，query 传输)。
type UsageLogSearch struct {
	commonReq.PageInfo
	UserId       int64  `json:"userId,string" form:"userId"`       // 归因用户(0=不限)
	AiKeyId      int64  `json:"aiKeyId,string" form:"aiKeyId"`     // 归因密钥(0=不限)
	DeploymentId int64  `json:"deploymentId,string" form:"deploymentId"` // 归因部署(0=不限)
	Model        string `json:"model" form:"model"`               // 模型名(模糊)
	Provider     string `json:"provider" form:"provider"`          // 供应商(精确)
	StartTime    string `json:"startTime" form:"startTime"`        // 开始时间起(ISO,可选)
	EndTime      string `json:"endTime" form:"endTime"`            // 结束时间止(ISO,可选)
}
