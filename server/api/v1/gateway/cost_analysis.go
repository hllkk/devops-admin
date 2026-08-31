package gateway

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// CostAnalysisApi 成本分析(P3，对齐前端 /gateway/cost/* 资源)。
// 挂 AI审计目录菜单(route.ai-audit_cost)，user 角色不授：管理员/决策层视角，
// 普通用户由 casbin 菜单授权挡住(规避 AIHelms 读端点零权限的坑)。
type CostAnalysisApi struct{}

// GetCostOverview
// @Tags      GatewayCost
// @Summary   成本分析总览(KPI含环比+按日趋势)
// @Produce   application/json
// @Param     startDate    query  string  false  "开始业务日(YYYY-MM-DD,缺省本月首日)"
// @Param     endDate      query  string  false  "结束业务日(缺省今天)"
// @Param     departmentId query  string  false  "部门筛选(含子树,0=不限)"
// @Param     userId       query  string  false  "用户筛选(0=不限)"
// @Param     aiKeyId      query  string  false  "密钥筛选(0=不限)"
// @Param     model        query  string  false  "模型名(精确)"
// @Param     provider     query  string  false  "供应商(精确)"
// @Success   200  {object}  response.Response{data=response.CostOverview,msg=string}
// @Router    /gateway/cost/overview [get]
func (a *CostAnalysisApi) GetCostOverview(c *gin.Context) {
	var f gatewayReq.CostAnalysisSearch
	if err := c.ShouldBindQuery(&f); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	ov, err := costAnalysisService.GetCostOverview(c.Request.Context(), &f)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取成本总览失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(ov, "获取成功", c)
}

// GetCostDetail
// @Tags      GatewayCost
// @Summary   成本多维明细(六维聚合,服务端分页)
// @Produce   application/json
// @Param     dimension    query  string  false  "维度(department/user/model/aiKey/provider/date,默认department)"
// @Param     sort         query  string  false  "排序键(internal/external/requests/tokens,默认internal)"
// @Param     pageNum      query  int     false  "页码"
// @Param     pageSize     query  int     false  "页大小(上限100)"
// @Param     startDate    query  string  false  "开始业务日"
// @Param     endDate      query  string  false  "结束业务日"
// @Param     departmentId query  string  false  "部门筛选(含子树)"
// @Param     userId       query  string  false  "用户筛选"
// @Param     aiKeyId      query  string  false  "密钥筛选"
// @Param     model        query  string  false  "模型名(精确)"
// @Param     provider     query  string  false  "供应商(精确)"
// @Success   200  {object}  response.Response{data=response.PageResult{Rows=[]response.CostDetailRow},msg=string}
// @Router    /gateway/cost/detail [get]
func (a *CostAnalysisApi) GetCostDetail(c *gin.Context) {
	var f gatewayReq.CostAnalysisSearch
	if err := c.ShouldBindQuery(&f); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	rows, total, err := costAnalysisService.GetCostDetail(c.Request.Context(), &f)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取成本明细失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: rows, Total: total, PageNum: f.PageNum, PageSize: f.PageSize,
	}, "获取成功", c)
}

// GetCostScopeUsers
// @Tags      GatewayCost
// @Summary   部门下钻成员成本(直挂口径,部门行=子和)
// @Produce   application/json
// @Param     deptId     query  string  true   "部门ID"
// @Param     startDate  query  string  false  "开始业务日"
// @Param     endDate    query  string  false  "结束业务日"
// @Param     userId     query  string  false  "用户筛选"
// @Param     aiKeyId    query  string  false  "密钥筛选"
// @Param     model      query  string  false  "模型名(精确)"
// @Param     provider   query  string  false  "供应商(精确)"
// @Success   200  {object}  response.Response{data=[]response.CostScopeUserRow,msg=string}
// @Router    /gateway/cost/detail/scope-users [get]
func (a *CostAnalysisApi) GetCostScopeUsers(c *gin.Context) {
	deptId, _ := strconv.ParseInt(c.Query("deptId"), 10, 64)
	var f gatewayReq.CostAnalysisSearch
	if err := c.ShouldBindQuery(&f); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	rows, err := costAnalysisService.GetCostScopeUsers(c.Request.Context(), deptId, &f)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取部门成员成本失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(rows, "获取成功", c)
}

// GetCostMcpTools
// @Tags      GatewayCost
// @Summary   MCP 维工具子表(指定 server 按工具聚合)
// @Produce   application/json
// @Param     serverId   query  string  true   "MCP服务器ID"
// @Param     startDate  query  string  false  "开始业务日"
// @Param     endDate    query  string  false  "结束业务日"
// @Param     departmentId query  string  false  "部门筛选(含子树)"
// @Param     userId     query  string  false  "用户筛选"
// @Param     aiKeyId    query  string  false  "密钥筛选"
// @Success   200  {object}  response.Response{data=[]response.CostDetailRow,msg=string}
// @Router    /gateway/cost/detail/mcp-tools [get]
func (a *CostAnalysisApi) GetCostMcpTools(c *gin.Context) {
	serverId, _ := strconv.ParseInt(c.Query("serverId"), 10, 64)
	var f gatewayReq.CostAnalysisSearch
	if err := c.ShouldBindQuery(&f); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	rows, err := costAnalysisService.GetCostMcpTools(c.Request.Context(), serverId, &f)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取MCP工具成本失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(rows, "获取成功", c)
}
