package gateway

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// DashboardApi 看板查询(对齐前端 /gateway/dashboard/* 资源)
type DashboardApi struct{}

// parseRange 解析时间范围 query（默认近30天），返回 start/end（业务日截断）。
func parseRange(c *gin.Context) (time.Time, time.Time) {
	layout := "2006-01-02"
	end := time.Now()
	start := end.AddDate(0, 0, -30)
	if s := c.Query("startDate"); s != "" {
		if t, err := time.Parse(layout, s); err == nil {
			start = t
		}
	}
	if e := c.Query("endDate"); e != "" {
		if t, err := time.Parse(layout, e); err == nil {
			end = t.AddDate(0, 0, 1) // 含当天
		}
	}
	return start, end
}

// canViewGlobalData 当前登录者是否可看全平台数据：超管直接放行；其余角色凭 casbin
// 持有「调用日志明细」(/gateway/usage) 权限即放行——能看全平台调用明细的角色，
// 看板汇总/预算/供应商资产不高于其数据敏感级。修复:「系统管理员」等管理角色
// super_admin=false 被强制 scope=self、自身无调用记录时看板整页空白的问题。
func canViewGlobalData(c *gin.Context) bool {
	claims := utils.GetUserInfo(c)
	if claims == nil {
		return false
	}
	if claims.SuperAdmin {
		return true
	}
	e := utils.GetCasbin()
	if e == nil {
		return false
	}
	ok, _ := e.Enforce(strconv.Itoa(int(claims.RoleId)), "/gateway/usage", c.Request.Method)
	return ok
}

// resolveScope 解析 scope：可看全局数据者尊重传入值(默认all)，其余强制 self（数据权限）。
func resolveScope(c *gin.Context) (string, int64) {
	if canViewGlobalData(c) {
		return c.DefaultQuery("scope", "all"), utils.GetUserID(c)
	}
	return "self", utils.GetUserID(c)
}

// GetOverview
// @Tags      GatewayDashboard
// @Summary   看板总览(总成本/请求数/token/预算汇总)
// @Produce   application/json
// @Param     startDate  query  string  false  "开始日期(YYYY-MM-DD,默认近30天)"
// @Param     endDate    query  string  false  "结束日期"
// @Param     scope      query  string  false  "范围(all/self,非超管强制self)"
// @Success   200  {object}  response.Response{data=response.DashboardOverview,msg=string}
// @Router    /gateway/dashboard/overview [get]
func (a *DashboardApi) GetOverview(c *gin.Context) {
	start, end := parseRange(c)
	scope, userId := resolveScope(c)
	ov, err := dashboardService.GetOverview(c.Request.Context(), start, end, scope, userId)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取看板总览失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(ov, "获取成功", c)
}

// GetTrend
// @Tags      GatewayDashboard
// @Summary   成本趋势(按日)
// @Produce   application/json
// @Param     startDate  query  string  false  "开始日期"
// @Param     endDate    query  string  false  "结束日期"
// @Param     scope      query  string  false  "范围(all/self)"
// @Success   200  {object}  response.Response{data=[]response.TrendItem,msg=string}
// @Router    /gateway/dashboard/trend [get]
func (a *DashboardApi) GetTrend(c *gin.Context) {
	start, end := parseRange(c)
	scope, userId := resolveScope(c)
	items, err := dashboardService.GetTrend(c.Request.Context(), start, end, scope, userId)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取成本趋势失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(items, "获取成功", c)
}

// GetTop
// @Tags      GatewayDashboard
// @Summary   Top10排行(按维度 user/model/aiKey,排序键 cost/requests/tokens)
// @Produce   application/json
// @Param     dimension  query  string  false  "维度(user/model/aiKey,默认user)"
// @Param     sort       query  string  false  "排序键(cost/requests/tokens,默认cost)"
// @Param     startDate  query  string  false  "开始日期"
// @Param     endDate    query  string  false  "结束日期"
// @Param     scope      query  string  false  "范围(all/self)"
// @Success   200  {object}  response.Response{data=[]response.TopItem,msg=string}
// @Router    /gateway/dashboard/top [get]
func (a *DashboardApi) GetTop(c *gin.Context) {
	start, end := parseRange(c)
	scope, userId := resolveScope(c)
	dimension := c.DefaultQuery("dimension", "user")
	sortKey := c.DefaultQuery("sort", "cost")
	items, err := dashboardService.GetTop(c.Request.Context(), start, end, dimension, sortKey, scope, userId)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取Top排行失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(items, "获取成功", c)
}

// GetBudget
// @Tags      GatewayDashboard
// @Summary   预算执行率(按Key)
// @Produce   application/json
// @Param     scope  query  string  false  "范围(all/self)"
// @Success   200  {object}  response.Response{data=[]response.BudgetItem,msg=string}
// @Router    /gateway/dashboard/budget [get]
func (a *DashboardApi) GetBudget(c *gin.Context) {
	scope, userId := resolveScope(c)
	items, err := dashboardService.GetBudget(c.Request.Context(), scope, userId)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取预算执行率失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(items, "获取成功", c)
}

// AggregateUsage 手动触发聚合(管理员)：先回流 LLM+MCP 日志再聚合，保证点了就是最新数据。
// 回流失败不阻断聚合(Warn 继续用已有原始日志)；聚合失败才整体失败。
// @Tags      GatewayDashboard
// @Summary   手动触发用量聚合(先回流LLM+MCP日志再滚动重建+budget重算+超限停用闭环)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=object,msg=string}
// @Router    /gateway/dashboard/aggregate [post]
func (a *DashboardApi) AggregateUsage(c *gin.Context) {
	ctx := c.Request.Context()
	// 联动回流：LLM+MCP 各自独立游标管线，失败只 Warn(定时回流仍在，聚合基于已有日志继续)
	synced := 0
	if r, err := usageService.SyncLLMLogs(ctx); err != nil {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Warn("手动聚合前 LLM 回流失败,基于已有日志继续聚合")
	} else {
		synced += r["inserted"]
	}
	if r, err := usageService.SyncMcpLogs(ctx); err != nil {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Warn("手动聚合前 MCP 回流失败,基于已有日志继续聚合")
	} else {
		synced += r["inserted"]
	}

	result, err := usageAggregateService.AggregateUsage(ctx)
	if err != nil {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Error("用量聚合失败")
		response.FailWithMessage("聚合失败: "+err.Error(), c)
		return
	}
	result["synced"] = synced
	response.OkWithDetailed(result, "聚合完成", c)
}
