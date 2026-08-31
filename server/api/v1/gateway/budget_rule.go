package gateway

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/request"
	"github.com/hllkk/devops-admin/server/model/common/response"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	sysReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/service"
	gatewayService "github.com/hllkk/devops-admin/server/service/gateway"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// BudgetRuleApi 多维预算管控(P3，对齐前端 /gateway/budget/* 资源)
type BudgetRuleApi struct{}

// GetBudgetRuleList
// @Tags      GatewayBudget
// @Summary   分页获取预算规则(含读时聚合已用+预警状态)
// @Produce   application/json
// @Param     scopeType  query  string  false  "维度(dept/user,空=全部)"
// @Param     isActive   query  int     false  "启停(1/0,nil=全部)"
// @Param     pageNum    query  int     true   "页码"
// @Param     pageSize   query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]response.BudgetRuleView},msg=string}
// @Router    /gateway/budget/list [get]
func (a *BudgetRuleApi) GetBudgetRuleList(c *gin.Context) {
	var q gatewayReq.BudgetRuleSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := budgetRuleService.GetBudgetRuleList(c.Request.Context(), &q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取预算规则失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: list, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}

// CreateBudgetRule
// @Tags      GatewayBudget
// @Summary   新增预算规则
// @Produce   application/json
// @Param     data  body  gatewayReq.BudgetRuleOperateParams  true  "规则配置"
// @Success   200   {object}  response.Response{data=response.BudgetRuleView,msg=string}
// @Router    /gateway/budget [post]
func (a *BudgetRuleApi) CreateBudgetRule(c *gin.Context) {
	var req gatewayReq.BudgetRuleOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := budgetRuleService.CreateBudgetRule(c.Request.Context(), &req, utils.GetUserID(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("创建预算规则失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "创建成功", c)
}

// UpdateBudgetRule
// @Tags      GatewayBudget
// @Summary   修改预算规则
// @Produce   application/json
// @Param     data  body  gatewayReq.BudgetRuleOperateParams  true  "规则配置"
// @Success   200   {object}  response.Response{data=response.BudgetRuleView,msg=string}
// @Router    /gateway/budget [put]
func (a *BudgetRuleApi) UpdateBudgetRule(c *gin.Context) {
	var req gatewayReq.BudgetRuleOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := budgetRuleService.UpdateBudgetRule(c.Request.Context(), &req, utils.GetUserID(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("修改预算规则失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "修改成功", c)
}

// DeleteBudgetRules
// @Tags      GatewayBudget
// @Summary   批量删除预算规则
// @Produce   application/json
// @Param     data  body  request.IdsReq  true  "规则ID列表"
// @Success   200   {object}  response.Response{msg=string}
// @Router    /gateway/budget [delete]
func (a *BudgetRuleApi) DeleteBudgetRules(c *gin.Context) {
	var req request.IdsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	ids := req.Int64s()
	if len(ids) == 0 {
		response.FailWithMessage("请选择要删除的规则", c)
		return
	}
	if err := budgetRuleService.DeleteBudgetRules(c.Request.Context(), ids); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("删除预算规则失败")
		response.FailWithMessage("删除失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// CheckBudgetAlerts 手动触发预算预警检查(含软限通知+硬限停用,由 API 层发通知规避 import 环)
// @Tags      GatewayBudget
// @Summary   手动触发预算预警检查
// @Produce   application/json
// @Success   200  {object}  response.Response{data=object,msg=string}
// @Router    /gateway/budget/check [post]
func (a *BudgetRuleApi) CheckBudgetAlerts(c *gin.Context) {
	results, err := budgetRuleService.CheckBudgetAlerts(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("预算预警检查失败")
		response.FailWithMessage("检查失败", c)
		return
	}
	softCnt, hardCnt := a.sendBudgetNotifications(c, results)
	response.OkWithDetailed(map[string]int{"softWarn": softCnt, "hardLimit": hardCnt}, fmt.Sprintf("检查完成: 软限预警 %d 条,硬限超限 %d 条", softCnt, hardCnt), c)
}

// AggregateUsage 手动触发用量聚合(含MCP纳入预算+软限预警+硬限停用)
// @Tags      GatewayBudget
// @Summary   手动触发用量聚合(含MCP纳入预算+软限预警+硬限停用)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=object,msg=string}
// @Router    /gateway/budget/aggregate [post]
func (a *BudgetRuleApi) AggregateUsage(c *gin.Context) {
	result, err := usageAggregateService.AggregateUsage(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("用量聚合失败")
		response.FailWithMessage("聚合失败: "+err.Error(), c)
		return
	}
	alertResults, alertErr := budgetRuleService.CheckBudgetAlerts(c.Request.Context())
	if alertErr != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(alertErr).Warn("预算预警检查失败(聚合已完成)")
	} else {
		softCnt, _ := a.sendBudgetNotifications(c, alertResults)
		result["budgetAlerts"] = softCnt
	}
	response.OkWithDetailed(result, "聚合完成", c)
}

// GetBudgetSummary 预算汇总(按维度:Key/部门/用户)
// @Tags      GatewayBudget
// @Summary   预算汇总(按维度:Key/部门/用户)
// @Produce   application/json
// @Param     scope  query  string  false  "维度(all/self,超管默认all)"
// @Success   200  {object}  response.Response{data=object,msg=string}
// @Router    /gateway/budget/summary [get]
func (a *BudgetRuleApi) GetBudgetSummary(c *gin.Context) {
	scope, userId := resolveScope(c)
	keyItems, err := dashboardService.GetBudget(c.Request.Context(), scope, userId)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取Key级预算失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	var ruleScope string
	if scope == "self" {
		ruleScope = "user"
	}
	isActive := true
	rules, _, err := budgetRuleService.GetBudgetRuleList(c.Request.Context(), &gatewayReq.BudgetRuleSearch{
		ScopeType: ruleScope, IsActive: &isActive,
	})
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Warn("获取预算规则列表失败")
	}
	deptRules, userRules := splitBudgetRules(rules, scope, userId)
	response.OkWithDetailed(map[string]any{
		"keys":  keyItems,
		"depts": deptRules,
		"users": userRules,
	}, "获取成功", c)
}

// sendBudgetNotifications 发送预算通知(规避 service↔gateway import 环,与审批通知同模式)。
func (a *BudgetRuleApi) sendBudgetNotifications(c *gin.Context, results []gatewayService.BudgetAlertResult) (softCnt, hardCnt int) {
	noticeSvc := service.ServiceGroupApp.SystemServiceGroup.NoticeService
	for i := range results {
		r := &results[i]
		var title, content string
		if r.AlertType == "hard_limit" {
			hardCnt++
			title = fmt.Sprintf("AI 预算超限：%s", r.ScopeLabel)
			content = fmt.Sprintf("「%s」预算已超限（已用 ¥%.2f / 限额 ¥%.2f，%.1f%%），已停用该范围内所有活跃 Key。", r.ScopeLabel, r.Used, r.Rule.BudgetLimit, r.Percent)
		} else {
			softCnt++
			title = fmt.Sprintf("AI 预算预警：%s", r.ScopeLabel)
			content = fmt.Sprintf("「%s」预算已用 %.1f%%（¥%.2f / ¥%.2f），达到预警阈值 %d%%。", r.ScopeLabel, r.Percent, r.Used, r.Rule.BudgetLimit, r.Rule.SoftWarnPercent)
		}
		if len(r.TargetIds) > 0 {
			if err := noticeSvc.CreateNotice(c.Request.Context(), sysReq.NoticeOperateParams{
				NoticeTitle: title, NoticeType: "1", NoticeContent: content,
				Status: "0", TargetType: "users", TargetUserIds: r.TargetIds,
			}, 0); err != nil {
				logger.WithCtx(c.Request.Context()).Mod("gateway").Warn(fmt.Sprintf("预算规则 %d 通知发送失败: %v", r.Rule.RuleId, err))
			}
		}
	}
	return
}

func splitBudgetRules(rules []gatewayResp.BudgetRuleView, scope string, userId int64) (deptRules, userRules []gatewayResp.BudgetRuleView) {
	for _, r := range rules {
		if scope == "self" && r.ScopeType == "user" && r.ScopeId != userId {
			continue
		}
		switch r.ScopeType {
		case "dept":
			deptRules = append(deptRules, r)
		case "user":
			userRules = append(userRules, r)
		}
	}
	return
}
