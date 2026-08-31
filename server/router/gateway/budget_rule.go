package gateway

import "github.com/gin-gonic/gin"

// BudgetRuleRouter 多维预算管控路由(对齐前端 /gateway/budget/* 资源)
type BudgetRuleRouter struct{}

// InitBudgetRuleRouter 挂在 PrivateGroup，鉴权/操作日志由该组全局中间件统一处理。
func (r *BudgetRuleRouter) InitBudgetRuleRouter(Router *gin.RouterGroup) {
	g := Router.Group("gateway/budget")
	{
		g.GET("list", budgetRuleApi.GetBudgetRuleList)           // 预算规则列表
		g.POST("", budgetRuleApi.CreateBudgetRule)               // 新增
		g.PUT("", budgetRuleApi.UpdateBudgetRule)                // 修改
		g.DELETE("", budgetRuleApi.DeleteBudgetRules)            // 删除
		g.GET("summary", budgetRuleApi.GetBudgetSummary)         // 三维度预算汇总
		g.POST("check", budgetRuleApi.CheckBudgetAlerts)         // 手动触发预警检查
		g.POST("aggregate", budgetRuleApi.AggregateUsage)        // 手动触发聚合(含预算检查)
	}
}
