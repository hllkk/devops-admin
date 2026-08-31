package gateway

import (
	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// UsageApi 用量统计与回流(对齐前端 /gateway/usage/* 资源)
type UsageApi struct{}

// SyncLLMLogs
// @Tags      GatewayUsage
// @Summary   手动触发 LiteLLM 用量回流(归因+成本重算，复合游标增量幂等)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=object,msg=string}
// @Router    /gateway/usage/sync [post]
func (a *UsageApi) SyncLLMLogs(c *gin.Context) {
	result, err := usageService.SyncLLMLogs(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("用量回流失败")
		response.FailWithMessage("回流失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "回流完成", c)
}

// ReconcileLLMLogs
// @Tags      GatewayUsage
// @Summary   手动触发对账回灌 LiteLLM 漏单(近30天 NOT EXISTS 兜底)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=object,msg=string}
// @Router    /gateway/usage/reconcile [post]
func (a *UsageApi) ReconcileLLMLogs(c *gin.Context) {
	result, err := usageService.ReconcileLLMLogs(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("用量对账失败")
		response.FailWithMessage("对账失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "对账完成", c)
}

// GetUsageLogList
// @Tags      GatewayUsage
// @Summary   分页获取用量日志(管理员视角，按用户/密钥/部署/模型/时间过滤)
// @Produce   application/json
// @Param     userId        query  int     false  "归因用户(0=不限)"
// @Param     aiKeyId       query  int     false  "归因密钥(0=不限)"
// @Param     deploymentId  query  int     false  "归因部署(0=不限)"
// @Param     model         query  string  false  "模型名(模糊)"
// @Param     provider      query  string  false  "供应商(精确)"
// @Param     startTime     query  string  false  "开始时间起(ISO8601)"
// @Param     endTime       query  string  false  "结束时间止(ISO8601)"
// @Param     pageNum       query  int     true   "页码"
// @Param     pageSize      query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]gateway.LlmLog},msg=string}
// @Router    /gateway/usage/list [get]
func (a *UsageApi) GetUsageLogList(c *gin.Context) {
	var q gatewayReq.UsageLogSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := usageService.GetUsageLogList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取用量日志失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: list, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}

// SyncMcpLogs
// @Tags      GatewayUsage
// @Summary   手动触发 MCP 调用回流(工具归因+per_call 成本,独立游标)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=object,msg=string}
// @Router    /gateway/usage/mcp/sync [post]
func (a *UsageApi) SyncMcpLogs(c *gin.Context) {
	result, err := usageService.SyncMcpLogs(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("MCP回流失败")
		response.FailWithMessage("回流失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "回流完成", c)
}

// ReconcileMcpLogs
// @Tags      GatewayUsage
// @Summary   手动触发 MCP 漏单对账回灌(近30天兜底)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=object,msg=string}
// @Router    /gateway/usage/mcp/reconcile [post]
func (a *UsageApi) ReconcileMcpLogs(c *gin.Context) {
	result, err := usageService.ReconcileMcpLogs(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("MCP对账失败")
		response.FailWithMessage("对账失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "对账完成", c)
}

// GetMcpLogList
// @Tags      GatewayUsage
// @Summary   分页获取 MCP 调用日志(管理员视角，按用户/密钥/服务器/工具/状态/时间过滤)
// @Produce   application/json
// @Param     userId       query  int     false  "归因用户(0=不限)"
// @Param     aiKeyId      query  int     false  "归因密钥(0=不限)"
// @Param     mcpServerId  query  int     false  "归因MCP服务器(0=不限)"
// @Param     toolName     query  string  false  "工具名(模糊)"
// @Param     status       query  string  false  "状态(success/error,空=全部)"
// @Param     startTime    query  string  false  "开始时间起(ISO8601)"
// @Param     endTime      query  string  false  "结束时间止(ISO8601)"
// @Param     pageNum      query  int     true   "页码"
// @Param     pageSize     query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]response.McpLogView},msg=string}
// @Router    /gateway/usage/mcp/list [get]
func (a *UsageApi) GetMcpLogList(c *gin.Context) {
	var q gatewayReq.McpLogSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := usageService.GetMcpLogList(c.Request.Context(), &q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取MCP调用日志失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: list, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}
