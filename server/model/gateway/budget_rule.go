package gateway

import "github.com/hllkk/devops-admin/server/global"

// BudgetRule 多维预算规则(P3：部门/用户级预算管控，与 Key 级预算并行)。
// scope_type + scope_id 唯一(同维度同对象只有一条规则)；budget_used 读时实时聚合
// (复用成本分析归因 JOIN 口径，不建 used 列避免两套口径漂移)。
// 部门预算范围=直挂部门(部门Key归部门+成员个人Key归主部门，不含子部门)；
// 用户预算范围=用户名下所有 Key。
type BudgetRule struct {
	global.OPS_AUDIT_MODEL
	RuleId          int64   `json:"ruleId,string" gorm:"primarykey;comment:规则ID(雪花)"`                          // 规则ID
	ScopeType       string  `json:"scopeType" gorm:"size:10;uniqueIndex:idx_budget_rule_scope;comment:维度(dept/user)"` // 维度
	ScopeId         int64   `json:"scopeId,string" gorm:"uniqueIndex:idx_budget_rule_scope;comment:对象ID(部门ID/用户ID)"` // 对象ID
	BudgetLimit     float64 `json:"budgetLimit" gorm:"type:numeric(12,4);comment:预算上限(¥,0=不限)"`                // 预算上限
	BudgetHardLimit bool    `json:"budgetHardLimit" gorm:"default:false;comment:硬限(超支停用scope内活跃Key)"`         // 硬限
	BudgetDuration  string  `json:"budgetDuration" gorm:"size:10;default:30d;comment:预算周期(1d/7d/30d)"`          // 预算周期
	SoftWarnPercent int     `json:"softWarnPercent" gorm:"default:80;comment:软限预警阈值(百分比,超此通知)"`            // 软限阈值
	IsActive        bool    `json:"isActive" gorm:"default:true;comment:是否启用"`                                 // 是否启用
}

func (BudgetRule) TableName() string {
	return "gateway_budget_rule"
}

// BudgetScopeType 预算维度
const (
	BudgetScopeDept = "dept" // 部门(直挂口径)
	BudgetScopeUser = "user" // 用户
)

// BudgetAlert 预算预警去重(P3：同维度同对象同周期只告警一次，防重复通知)。
// period_key 格式 YYYY-MM(月周期)或 YYYY-MM-DD(日/周周期，按 budget_duration 切)。
type BudgetAlert struct {
	global.OPS_MODEL
	AlertId   int64  `json:"alertId,string" gorm:"primarykey;comment:告警ID(雪花)"`                     // 告警ID
	RuleId    int64  `json:"ruleId,string" gorm:"uniqueIndex:idx_budget_alert;comment:关联规则ID"`       // 关联规则
	PeriodKey string `json:"periodKey" gorm:"size:20;uniqueIndex:idx_budget_alert;comment:周期键(去重)"` // 周期键
	AlertType string `json:"alertType" gorm:"size:10;comment:告警类型(soft_warn/hard_limit)"`          // 告警类型
}

func (BudgetAlert) TableName() string {
	return "gateway_budget_alert"
}

// BudgetAlertType 告警类型
const (
	BudgetAlertSoftWarn  = "soft_warn"  // 软限预警
	BudgetAlertHardLimit = "hard_limit" // 硬限超限
)
