package gateway

import (
	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// RouterSettingsApi 全局路由策略管理(对齐前端 /gateway/router/settings)。
type RouterSettingsApi struct{}

// GetRouterSettings
// @Tags      GatewayRouterSettings
// @Summary   获取全局路由策略
// @Produce   application/json
// @Success   200  {object}  response.Response{data=response.RouterSettingsView,msg=string}
// @Router    /gateway/router/settings [get]
func (a *RouterSettingsApi) GetRouterSettings(c *gin.Context) {
	view, err := routerSettingsService.Get(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取路由策略失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(view, "获取成功", c)
}

// UpdateRouterSettings
// @Tags      GatewayRouterSettings
// @Summary   更新全局路由策略(落库 + 同步 LiteLLM 热更新)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.RouterSettingsUpdate  true  "路由策略配置"
// @Success   200   {object}  response.Response{data=response.RouterSettingsView,msg=string}
// @Router    /gateway/router/settings [put]
func (a *RouterSettingsApi) UpdateRouterSettings(c *gin.Context) {
	var req gatewayReq.RouterSettingsUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := routerSettingsService.Update(c.Request.Context(), req)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("更新路由策略失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "更新成功", c)
}
