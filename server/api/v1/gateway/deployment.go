package gateway

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// DeploymentApi 模型部署管理(对齐前端 /gateway/model/deployment/* 资源)
type DeploymentApi struct{}

// GetDeploymentList
// @Tags      GatewayModelDeployment
// @Summary   分页获取部署列表(带路由名/凭证上下文)
// @Produce   application/json
// @Param     modelId      query  int     false  "关联模型ID(0=不限)"
// @Param     credentialId query  int     false  "关联凭证ID(0=不限)"
// @Param     keyword      query  string  false  "部署名(模糊)"
// @Param     isActive     query  bool    false  "是否启用(精确)"
// @Param     pageNum      query  int     true   "页码"
// @Param     pageSize     query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]response.DeploymentView},msg=string}
// @Router    /gateway/model/deployment/list [get]
func (a *DeploymentApi) GetDeploymentList(c *gin.Context) {
	var q gatewayReq.DeploymentSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := deploymentService.GetDeploymentList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取部署列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: list, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}

// CreateDeployment
// @Tags      GatewayModelDeployment
// @Summary   新增部署(事务内同步 LiteLLM，推送失败整体回滚；前缀化路由+凭证引用)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.DeploymentOperateParams  true  "部署信息"
// @Success   200   {object}  response.Response{data=response.DeploymentView,msg=string}
// @Router    /gateway/model/deployment [post]
func (a *DeploymentApi) CreateDeployment(c *gin.Context) {
	var req gatewayReq.DeploymentOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := deploymentService.CreateDeployment(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "新增成功", c)
}

// UpdateDeployment
// @Tags      GatewayModelDeployment
// @Summary   修改部署(敏感参数掩码回传还原；重建投影并推送，改名+active 双写)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.DeploymentOperateParams  true  "部署信息(含 deploymentId)"
// @Success   200   {object}  response.Response{data=response.DeploymentView,msg=string}
// @Router    /gateway/model/deployment [put]
func (a *DeploymentApi) UpdateDeployment(c *gin.Context) {
	var req gatewayReq.DeploymentOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := deploymentService.UpdateDeployment(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "修改成功", c)
}

// BatchDeleteDeployments
// @Tags      GatewayModelDeployment
// @Summary   批量删除部署(先在 LiteLLM 侧禁用留痕，失败则本地不动)
// @Produce   application/json
// @Param     ids  path  string  true  "部署ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /gateway/model/deployment/{ids} [delete]
func (a *DeploymentApi) BatchDeleteDeployments(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的部署ID: "+s, c)
			return
		}
		ids = append(ids, id)
	}
	if err := deploymentService.DeleteDeployments(c.Request.Context(), ids); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}

// TestDeployment
// @Tags      GatewayModelDeployment
// @Summary   部署连通性测试(管理员视角，经 LiteLLM 数据面按类别分流，错误脱敏)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.DeploymentTestParams  true  "部署ID"
// @Success   200   {object}  response.Response{data=response.DeploymentTestResult,msg=string}
// @Router    /gateway/model/deployment/test [post]
func (a *DeploymentApi) TestDeployment(c *gin.Context) {
	var req gatewayReq.DeploymentTestParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	result, err := deploymentService.TestDeployment(c.Request.Context(), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "测试完成", c)
}
