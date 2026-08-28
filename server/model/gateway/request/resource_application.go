package request

import (
	"github.com/hllkk/devops-admin/server/model/common"
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// ApplicationCreateParams 用户提交资源申请(用户侧,body)。
// reason 前端按钮约束非空,后端不强制(对齐 AIHelms 宽容口径)。
type ApplicationCreateParams struct {
	ResourceType string `json:"resourceType" binding:"required"`     // 资源类型(model/mcp/skill)
	ResourceId   int64  `json:"resourceId,string" binding:"required"` // 资源ID
	Reason       string `json:"reason"`                               // 申请理由
}

// ApplicationSearch 申请列表查询(GET query,用户侧 my 与管理端 list 共用;
// my 模式 userId 强制取当前登录人,查询参数里的 userId 忽略)。
type ApplicationSearch struct {
	commonReq.PageInfo
	Status       string `json:"status" form:"status"`             // 状态(精确,空=不限)
	ResourceType string `json:"resourceType" form:"resourceType"` // 资源类型(精确,空=不限)
	UserId       int64  `json:"userId,string" form:"userId"`      // 申请人(0=不限,仅管理端生效)
}

// ApplicationReviewParams 单条审批(通过/驳回,body)。
type ApplicationReviewParams struct {
	ApplicationId int64  `json:"applicationId,string" binding:"required"` // 申请ID
	ReviewNotes   string `json:"reviewNotes"`                             // 审批意见(可空)
}

// ApplicationBatchReviewParams 批量审批(body;前端 ID 列表为字符串,须用 Int64StringSlice)。
type ApplicationBatchReviewParams struct {
	ApplicationIds common.Int64StringSlice `json:"applicationIds" binding:"required"` // 申请ID列表
	ReviewNotes    string                     `json:"reviewNotes"`                       // 审批意见(批量统一)
}
