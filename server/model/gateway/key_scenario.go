package gateway

import (
	"github.com/hllkk/devops-admin/server/global"
)

// KeyScenario 使用场景(场景 Key 的分类字典，对齐 AIHelms key_scenarios)。
// 极简实体：名称+描述+启停；场景 Key 经 AiKey.ScenarioId 逻辑关联(不建外键)。
// 同名约束按"未软删行"口径(部分唯一索引 idx_keyscenario_name，软删后可重建同名；
// 停用行仍占名——停用场景在列表可见可恢复，同名二义比占名更糟)。
// 区别于 P4 规划的"业务场景模板"(BusinessScenario 级别，带资源配置包)，本实体只是字典。
type KeyScenario struct {
	global.OPS_AUDIT_MODEL
	ScenarioId  int64  `json:"scenarioId,string" gorm:"primarykey;comment:场景ID(雪花)"` // 场景ID(雪花)
	Name        string `json:"name" gorm:"size:64;comment:场景名称"`                    // 场景名称(未软删行唯一)
	Description string `json:"description" gorm:"type:text;comment:描述"`                // 描述
	IsActive    bool   `json:"isActive" gorm:"default:true;comment:是否启用(停用后新建Key不可选)"` // 是否启用
}

func (KeyScenario) TableName() string {
	return "gateway_key_scenario"
}
