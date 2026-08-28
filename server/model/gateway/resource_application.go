package gateway

import (
	"time"

	"github.com/hllkk/devops-admin/server/global"
)

// 资源申请状态(一次性单向:pending → approved/rejected,无撤回)
const (
	ApplicationStatusPending  = "pending"  // 待审批
	ApplicationStatusApproved = "approved" // 已批准
	ApplicationStatusRejected = "rejected" // 已驳回
)

// 资源申请类型。P2 先实现 model;mcp/skill 实体未建,表结构与枚举预留(公共底座,
// 后续切片只加类型分支不动表)。
const (
	ApplicationResourceModel = "model"
	ApplicationResourceMcp   = "mcp"
	ApplicationResourceSkill = "skill"
)

// ResourceApplication 资源申请审批(模型"领用前需审批"档的消费闭环)。
// (user_id, resource_type, resource_id) 复合唯一:pending 重复提交拒、approved 拒(已拥有)、
// rejected 再申请=复用原行重置为 pending(审批字段清空)——行永不软删,规避软删行占唯一
// 索引挡重新申请(gateway_model_visibility 投影表同款考量)。
type ResourceApplication struct {
	global.OPS_AUDIT_MODEL
	ApplicationId int64      `json:"applicationId,string" gorm:"primarykey;comment:申请ID(雪花)"`                                          // 申请ID
	UserId        int64      `json:"userId,string" gorm:"uniqueIndex:idx_gateway_application_resource;comment:申请人(sys_users.id)"`        // 申请人
	ResourceType  string     `json:"resourceType" gorm:"size:20;uniqueIndex:idx_gateway_application_resource;comment:资源类型(model/mcp/skill)"` // 资源类型
	ResourceId    int64      `json:"resourceId,string" gorm:"uniqueIndex:idx_gateway_application_resource;comment:资源ID(多态,无FK)"`         // 资源ID
	Reason        string     `json:"reason" gorm:"type:text;comment:申请理由"`                                                               // 申请理由
	Status        string     `json:"status" gorm:"size:20;index;default:pending;comment:状态(pending/approved/rejected)"`                  // 状态
	ReviewedBy    int64      `json:"reviewedBy,string" gorm:"default:0;comment:审批人(0=未审批)"`                                              // 审批人
	ReviewedAt    *time.Time `json:"reviewedAt" gorm:"comment:审批时间(nil=未审批)"`                                                            // 审批时间
	ReviewNotes   string     `json:"reviewNotes" gorm:"type:text;comment:审批意见"`                                                          // 审批意见
}

func (ResourceApplication) TableName() string {
	return "gateway_resource_application"
}
