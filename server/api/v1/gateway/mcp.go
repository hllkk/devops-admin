package gateway

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hllkk/devops-admin/server/model/common/response"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
	"github.com/hllkk/devops-admin/server/utils/request"
)

// MCPApi MCP 服务器管理(对齐前端 /gateway/mcp/* 资源)
type MCPApi struct{}

// GetMCPServerList
// @Tags      GatewayMCP
// @Summary   分页获取MCP服务器列表
// @Produce   application/json
// @Param     name          query  string  false  "展示名/路由名(模糊)"
// @Param     category      query  string  false  "分类(精确)"
// @Param     isActive      query  bool    false  "是否启用(精确)"
// @Param     isPublished   query  bool    false  "是否发布(精确)"
// @Param     healthStatus  query  string  false  "健康状态(精确)"
// @Param     pageNum       query  int     true   "页码"
// @Param     pageSize      query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]response.MCPServerView},msg=string}
// @Router    /gateway/mcp/list [get]
func (a *MCPApi) GetMCPServerList(c *gin.Context) {
	var q gatewayReq.MCPServerSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	request.NormalizeEmptyBoolQuery(c, &q)
	list, total, err := mcpService.GetMCPServerList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取MCP服务器列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: list, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}

// GetMCPCategories
// @Tags      GatewayMCP
// @Summary   MCP分类去重列表(管理端下拉受控数据源)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]string,msg=string}
// @Router    /gateway/mcp/categories [get]
func (a *MCPApi) GetMCPCategories(c *gin.Context) {
	list, err := mcpService.GetMCPCategories(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取MCP分类列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// GetMCPServer
// @Tags      GatewayMCP
// @Summary   获取MCP服务器详情(含工具列表)
// @Produce   application/json
// @Param     id  path  int  true  "服务器ID"
// @Success   200  {object}  response.Response{data=response.MCPServerDetail,msg=string}
// @Router    /gateway/mcp/{id} [get]
func (a *MCPApi) GetMCPServer(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的服务器ID", c)
		return
	}
	detail, err := mcpService.GetMCPServer(c.Request.Context(), id)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取MCP服务器详情失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(detail, "获取成功", c)
}

// CreateMCPServer
// @Tags      GatewayMCP
// @Summary   注册MCP服务器
// @Produce   application/json
// @Param     data  body  gatewayReq.MCPServerOperateParams  true  "MCP服务器参数"
// @Success   200  {object}  response.Response{data=response.MCPServerView,msg=string}
// @Router    /gateway/mcp [post]
func (a *MCPApi) CreateMCPServer(c *gin.Context) {
	var req gatewayReq.MCPServerOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := mcpService.CreateMCPServer(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("创建MCP服务器失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "创建成功", c)
}

// UpdateMCPServer
// @Tags      GatewayMCP
// @Summary   修改MCP服务器
// @Produce   application/json
// @Param     data  body  gatewayReq.MCPServerOperateParams  true  "MCP服务器参数"
// @Success   200  {object}  response.Response{data=response.MCPServerView,msg=string}
// @Router    /gateway/mcp [put]
func (a *MCPApi) UpdateMCPServer(c *gin.Context) {
	var req gatewayReq.MCPServerOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := mcpService.UpdateMCPServer(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("修改MCP服务器失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "修改成功", c)
}

// DeleteMCPServers
// @Tags      GatewayMCP
// @Summary   批量删除MCP服务器
// @Produce   application/json
// @Param     ids  path  string  true  "服务器ID(逗号分隔)"
// @Success   200  {object}  response.Response{msg=string}
// @Router    /gateway/mcp/{ids} [delete]
func (a *MCPApi) DeleteMCPServers(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的服务器ID: "+s, c)
			return
		}
		ids = append(ids, id)
	}
	if err := mcpService.DeleteMCPServers(c.Request.Context(), ids); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("删除MCP服务器失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// GetMCPPublish
// @Tags      GatewayMCP
// @Summary   获取MCP发布设置(含可见部门/用户回显)
// @Produce   application/json
// @Param     id  path  int  true  "服务器ID"
// @Success   200  {object}  response.Response{data=response.MCPPublishView,msg=string}
// @Router    /gateway/mcp/publish/{id} [get]
func (a *MCPApi) GetMCPPublish(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的服务器ID", c)
		return
	}
	view, err := mcpService.GetMCPPublish(c.Request.Context(), id)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取MCP发布设置失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "获取成功", c)
}

// PublishMCPServer
// @Tags      GatewayMCP
// @Summary   更新MCP发布设置(三档可见性+需审批)
// @Produce   application/json
// @Param     data  body  gatewayReq.MCPPublishParams  true  "发布设置"
// @Success   200  {object}  response.Response{msg=string}
// @Router    /gateway/mcp/publish [put]
func (a *MCPApi) PublishMCPServer(c *gin.Context) {
	var req gatewayReq.MCPPublishParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := mcpService.PublishMCPServer(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("更新MCP发布设置失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("发布设置已更新", c)
}

// RefreshMCPTools
// @Tags      GatewayMCP
// @Summary   刷新MCP工具列表(远端全量重建)
// @Produce   application/json
// @Param     id  path  int  true  "服务器ID"
// @Success   200  {object}  response.Response{data=[]response.MCPToolView,msg=string}
// @Router    /gateway/mcp/{id}/refresh-tools [post]
func (a *MCPApi) RefreshMCPTools(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的服务器ID", c)
		return
	}
	tools, err := mcpService.RefreshMCPTools(c.Request.Context(), id)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("刷新MCP工具列表失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(tools, "工具列表刷新成功", c)
}

// UpdateMCPToolBilling
// @Tags      GatewayMCP
// @Summary   更新MCP工具计费(空=继承服务器)
// @Produce   application/json
// @Param     toolId  path  int  true  "工具ID"
// @Param     data    body  gatewayReq.MCPToolBillingParams  true  "计费配置"
// @Success   200  {object}  response.Response{data=response.MCPToolView,msg=string}
// @Router    /gateway/mcp/tool/{toolId}/billing [put]
func (a *MCPApi) UpdateMCPToolBilling(c *gin.Context) {
	toolId, err := strconv.ParseInt(c.Param("toolId"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的工具ID", c)
		return
	}
	var req gatewayReq.MCPToolBillingParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := mcpService.UpdateMCPToolBilling(c.Request.Context(), toolId, req.BillingType, req.ExternalCostPerCall)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("更新MCP工具计费失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "工具计费已更新", c)
}

// HealthCheckMCPServer
// @Tags      GatewayMCP
// @Summary   MCP服务器健康检查
// @Produce   application/json
// @Param     id  path  int  true  "服务器ID"
// @Success   200  {object}  response.Response{data=response.MCPServerView,msg=string}
// @Router    /gateway/mcp/{id}/health-check [post]
func (a *MCPApi) HealthCheckMCPServer(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的服务器ID", c)
		return
	}
	view, err := mcpService.HealthCheckMCPServer(c.Request.Context(), id)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("MCP服务器健康检查失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "健康检查完成", c)
}

// GetAvailableMcps
// @Tags      GatewayMCP
// @Summary   可授权MCP列表(管理端,全部启用,授权下拉用)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]response.AvailableMcpView,msg=string}
// @Router    /gateway/mcp/available [get]
func (a *MCPApi) GetAvailableMcps(c *gin.Context) {
	list, err := mcpService.GetAvailableMcps(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取可授权MCP列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// GetActiveMcps
// @Tags      GatewayMCP
// @Summary   用户侧可见MCP列表(广场,按发布可见性过滤;casbin 登录白名单)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]response.AvailableMcpView,msg=string}
// @Router    /gateway/mcp/active [get]
func (a *MCPApi) GetActiveMcps(c *gin.Context) {
	list, err := mcpService.GetActiveMcps(c.Request.Context(), utils.GetUserID(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取用户侧MCP列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// GetMCPConnectConfig
// @Tags      GatewayMCP
// @Summary   获取MCP接入配置(用户侧,主Key明文作鉴权头;casbin 登录白名单)
// @Produce   application/json
// @Param     id  path  int  true  "服务器ID"
// @Success   200  {object}  response.Response{data=response.MCPConnectConfigView,msg=string}
// @Router    /gateway/mcp/connect-config/{id} [get]
func (a *MCPApi) GetMCPConnectConfig(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的服务器ID", c)
		return
	}
	view, err := mcpService.GetMCPConnectConfig(c.Request.Context(), id, utils.GetUserID(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取MCP接入配置失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "获取成功", c)
}
