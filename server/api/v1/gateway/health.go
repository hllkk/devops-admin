package gateway

import (
	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// HealthApi 健康检查(P3，对齐前端 /gateway/health/* 资源)。
// 挂 AI审计目录菜单(route.ai-audit_health)，user 角色不授：管理员/运维视角。
type HealthApi struct{}

// GetHealthSummary
// @Tags      GatewayHealth
// @Summary   健康检查汇总(四卡:MCP上游/模型部署/基础组件/数据回流新鲜度+三块明细)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=response.HealthSummary,msg=string}
// @Router    /gateway/health/summary [get]
func (a *HealthApi) GetHealthSummary(c *gin.Context) {
	sum, err := healthService.GetHealthSummary(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取健康汇总失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(sum, "获取成功", c)
}

// HealthCheckDeployments
// @Tags      GatewayHealth
// @Summary   手动巡检全部模型部署(按路由组探测落库,单机模式跳过)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=number,msg=string}
// @Router    /gateway/health/check-deployments [post]
func (a *HealthApi) HealthCheckDeployments(c *gin.Context) {
	checked, err := healthService.HealthCheckAllDeployments(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("模型部署健康巡检失败")
		response.FailWithMessage("巡检失败", c)
		return
	}
	response.OkWithDetailed(checked, "巡检完成", c)
}
