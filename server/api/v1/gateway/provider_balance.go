package gateway

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// ProviderBalanceApi 套餐余量旁路（供应商面板 + 看板汇总卡，只读口径，不进成本链路）。
type ProviderBalanceApi struct{}

// parseProviderId 路径参数 → 供应商ID。
func parseProviderId(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.FailWithMessage("无效的供应商ID", c)
		return 0, false
	}
	return id, true
}

// GetProviderBalances
// @Tags      GatewayProvider
// @Summary   供应商套餐余量明细(坐席/共享包,厂商侧快照)
// @Produce   application/json
// @Param     id  path  int  true  "供应商ID"
// @Success   200  {object}  response.Response{data=response.ProviderBalanceDetail,msg=string}
// @Router    /gateway/provider/{id}/balance [get]
func (a *ProviderBalanceApi) GetProviderBalances(c *gin.Context) {
	id, ok := parseProviderId(c)
	if !ok {
		return
	}
	items, summary, err := providerBalanceService.GetProviderBalances(c.Request.Context(), id)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取供应商余量失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(gatewayResp.ProviderBalanceDetail{Summary: summary, Items: items}, "获取成功", c)
}

// GetBalanceConfig
// @Tags      GatewayProvider
// @Summary   读余量采集配置(AK/SK 掩码回显)
// @Produce   application/json
// @Param     id  path  int  true  "供应商ID"
// @Success   200  {object}  response.Response{data=gateway.BalanceSyncConfig,msg=string}
// @Router    /gateway/provider/{id}/balance-config [get]
func (a *ProviderBalanceApi) GetBalanceConfig(c *gin.Context) {
	id, ok := parseProviderId(c)
	if !ok {
		return
	}
	cfg, err := providerBalanceService.GetBalanceConfigView(c.Request.Context(), id)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取余量采集配置失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(cfg, "获取成功", c)
}

// SaveBalanceConfig
// @Tags      GatewayProvider
// @Summary   保存余量采集配置(阿里云 AK/SK,掩码占位保留旧明文)
// @Accept    application/json
// @Produce   application/json
// @Param     id    path  int                           true  "供应商ID"
// @Param     data  body  gateway.BalanceSyncConfig     true  "采集配置"
// @Success   200   {object}  response.Response{msg=string}
// @Router    /gateway/provider/{id}/balance-config [put]
func (a *ProviderBalanceApi) SaveBalanceConfig(c *gin.Context) {
	id, ok := parseProviderId(c)
	if !ok {
		return
	}
	var cfg gateway.BalanceSyncConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := providerBalanceService.SaveBalanceConfig(c.Request.Context(), id, cfg); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("保存余量采集配置失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("保存成功", c)
}

// SyncProviderBalance
// @Tags      GatewayProvider
// @Summary   同步供应商套餐余量(拉百炼坐席+共享包)
// @Produce   application/json
// @Param     id    path  int   true  "供应商ID"
// @Param     auto  query bool  false "自动模式(进入面板静默触发:未配置凭证/距上次同步过近/失败均不报错,返回当前快照汇总)"
// @Success   200  {object}  response.Response{data=response.ProviderBalanceSummary,msg=string}
// @Router    /gateway/provider/{id}/balance-sync [post]
func (a *ProviderBalanceApi) SyncProviderBalance(c *gin.Context) {
	id, ok := parseProviderId(c)
	if !ok {
		return
	}
	summary, err := providerBalanceService.SyncProviderBalance(c.Request.Context(), id, c.Query("auto") == "true")
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("同步供应商余量失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(summary, "同步成功", c)
}

// GetBalanceSummary
// @Tags      GatewayDashboard
// @Summary   跨供应商套餐余量汇总(看板汇总卡;非超管返回空,厂商侧口径与标价成本互不并)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]response.ProviderBalanceSummary,msg=string}
// @Router    /gateway/dashboard/balance-summary [get]
func (a *ProviderBalanceApi) GetBalanceSummary(c *gin.Context) {
	// 供应商资产属管理视角：无全局数据视角者(强制 self scope)不下发(口径同 resolveScope)
	if !canViewGlobalData(c) {
		response.OkWithDetailed([]gatewayResp.ProviderBalanceSummary{}, "获取成功", c)
		return
	}
	list, err := providerBalanceService.GetBalanceSummaryAll(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取余量汇总失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}
