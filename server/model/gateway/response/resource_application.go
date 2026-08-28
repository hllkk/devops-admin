package response

import (
	"github.com/hllkk/devops-admin/server/model/gateway"
)

// ApplicationView 申请视图(我的申请/审批列表共用):主体 + 申请人/资源/审批人名称回填。
type ApplicationView struct {
	gateway.ResourceApplication
	UserName     string `json:"userName"`     // 申请人昵称
	ResourceName string `json:"resourceName"` // 资源展示名(model=模型名)
	ResourceKey  string `json:"resourceKey"`  // 资源路由名(model=model_key,其余类型空)
	ReviewerName string `json:"reviewerName"` // 审批人昵称(未审批空)
}

// ApplicationReviewResult 单条审批结果(warnings=授权/LiteLLM 同步警告,主流程成功)。
// 主 Key 不存在时 scope 圈空集、授权由自愈差集源在后建主 Key 时补上,不视为失败。
type ApplicationReviewResult struct {
	Warnings []string `json:"warnings"` // 非致命警告(LiteLLM 推送失败等,由每日 ResyncAiKeys 兜底)
}

// BatchReviewFailure 批量审批单条失败项。
type BatchReviewFailure struct {
	ApplicationId int64  `json:"applicationId,string"` // 申请ID
	Reason        string `json:"reason"`               // 失败原因
}

// BatchReviewResult 批量审批结果(成功 ID 列表 + 失败明细)。
type BatchReviewResult struct {
	Success []int64              `json:"success"` // 成功的申请ID
	Failed  []BatchReviewFailure `json:"failed"`  // 失败明细
}

// ReviewNotification 审批结果通知所需信息(service → api 层)。通知在 api 层发送:
// service/system 已反向 import service/gateway(用户级联),service 层调用会成环。
type ReviewNotification struct {
	UserId       int64  `json:"userId"`       // 申请人
	ResourceName string `json:"resourceName"` // 资源展示名
	Approved     bool   `json:"approved"`     // 是否通过
	ReviewNotes  string `json:"reviewNotes"`  // 审批意见
}
