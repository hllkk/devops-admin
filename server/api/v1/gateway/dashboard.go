package gateway

import (
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

// resolveScope 解析 scope：非超管强制 self（数据权限），超管尊重传入值。
func resolveScope(c *gin.Context) (string, int64) {
	claims := utils.GetUserInfo(c)
	if claims != nil && claims.SuperAdmin {
		scope := c.DefaultQuery("scope", "all")
		return scope, utils.GetUserID(c)
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
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(items, "获取成功", c)
}

// GetTop
// @Tags      GatewayDashboard
// @Summary   成本Top10(按维度 user/model/aiKey)
// @Produce   application/json
// @Param     dimension  query  string  false  "维度(user/model/aiKey,默认user)"
// @Param     startDate  query  string  false  "开始日期"
// @Param     endDate    query  string  false  "结束日期"
// @Param     scope      query  string  false  "范围(all/self)"
// @Success   200  {object}  response.Response{data=[]response.TopItem,msg=string}
// @Router    /gateway/dashboard/top [get]
func (a *DashboardApi) GetTop(c *gin.Context) {
	start, end := parseRange(c)
	scope, userId := resolveScope(c)
	dimension := c.DefaultQuery("dimension", "user")
	items, err := dashboardService.GetTop(c.Request.Context(), start, end, dimension, scope, userId)
	if err != nil {
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
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(items, "获取成功", c)
}

// AggregateUsage 手动触发聚合(管理员)
// @Tags      GatewayDashboard
// @Summary   手动触发用量聚合(滚动重建+budget重算+超限停用闭环)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=object,msg=string}
// @Router    /gateway/dashboard/aggregate [post]
func (a *DashboardApi) AggregateUsage(c *gin.Context) {
	result, err := usageAggregateService.AggregateUsage(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("用量聚合失败")
		response.FailWithMessage("聚合失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "聚合完成", c)
}
