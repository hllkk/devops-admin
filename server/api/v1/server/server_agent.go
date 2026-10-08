package server

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// AgentApi 采集端(node_exporter)安装/运维 + Prometheus 查询代理。
// 安装/重启/卸载/状态轮询/快照/趋势挂 PrivateGroup(JWT/casbin,菜单 ApiPrefix
// /server/asset/* 覆盖);HTTP SD 端点挂 PublicGroup(Bearer sd-token 自鉴权,
// 供 Prometheus http_sd_configs 拉取资产清单)。
type AgentApi struct{}

// InstallAgent
// @Tags      ServerAgent
// @Summary   触发采集端(node_exporter)自动安装(异步,纯公钥模式;含防火墙 9100 放行)
// @Produce   application/json
// @Param     id    path  int  true  "资产ID(需已通过 SSH 录入验证)"
// @Success   200   {object}  response.Response{data=object{taskId=string},msg=string}
// @Router    /server/asset/{id}/install-agent [post]
func (a *AgentApi) InstallAgent(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的资产ID", c)
		return
	}
	taskId, err := agentInstallService.StartInstall(c.Request.Context(), id)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Field("assetId", id).Error("触发采集端安装失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{"taskId": taskId}, "安装任务已启动", c)
}

// RestartAgent
// @Tags      ServerAgent
// @Summary   重启目标机采集端(异步,公钥通道)
// @Produce   application/json
// @Param     id  path  int  true  "资产ID"
// @Success   200 {object}  response.Response{data=object{taskId=string},msg=string}
// @Router    /server/asset/{id}/restart-agent [post]
func (a *AgentApi) RestartAgent(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的资产ID", c)
		return
	}
	taskId, err := agentInstallService.StartRestart(c.Request.Context(), id)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Field("assetId", id).Error("触发采集端重启失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{"taskId": taskId}, "重启任务已启动", c)
}

// UninstallAgent
// @Tags      ServerAgent
// @Summary   卸载目标机采集端(异步,公钥通道;平台公钥保留可重装,撤防火墙端口)
// @Produce   application/json
// @Param     id  path  int  true  "资产ID"
// @Success   200 {object}  response.Response{data=object{taskId=string},msg=string}
// @Router    /server/asset/{id}/uninstall-agent [post]
func (a *AgentApi) UninstallAgent(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的资产ID", c)
		return
	}
	taskId, err := agentInstallService.StartUninstall(c.Request.Context(), id)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Field("assetId", id).Error("触发采集端卸载失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{"taskId": taskId}, "卸载任务已启动", c)
}

// GetInstallStatus
// @Tags      ServerAgent
// @Summary   轮询采集端安装/运维任务状态
// @Produce   application/json
// @Param     taskId  path  string  true  "任务ID"
// @Success   200  {object}  response.Response{data=response.AgentInstallStatus,msg=string}
// @Router    /server/asset/install-status/{taskId} [get]
func (a *AgentApi) GetInstallStatus(c *gin.Context) {
	taskId := strings.TrimSpace(c.Param("taskId"))
	if taskId == "" {
		response.FailWithMessage("任务ID不能为空", c)
		return
	}
	status, err := agentInstallService.GetInstallStatus(c.Request.Context(), taskId)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(status, "获取成功", c)
}

// GetAssetSnapshot
// @Tags      ServerAsset
// @Summary   资产实时快照(后端代查 Prometheus instant;未配置/未刮到时 stale=true)
// @Produce   application/json
// @Param     id  path  int  true  "资产ID"
// @Success   200  {object}  response.Response{data=response.AgentSnapshot,msg=string}
// @Router    /server/asset/{id}/snapshot [get]
func (a *AgentApi) GetAssetSnapshot(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的资产ID", c)
		return
	}
	snap, err := promService.GetSnapshot(c.Request.Context(), id)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Field("assetId", id).Error("获取资产快照失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(snap, "获取成功", c)
}

// GetAssetMetricsTrend
// @Tags      ServerAsset
// @Summary   资产指标趋势(后端代查 Prometheus query_range)
// @Produce   application/json
// @Param     id     path   string  true   "资产ID"
// @Param     range  query  string  false  "时间范围(1h/1d/7d/30d,默认1h)"
// @Success   200  {object}  response.Response{data=response.MetricsTrend,msg=string}
// @Router    /server/asset/{id}/metrics [get]
func (a *AgentApi) GetAssetMetricsTrend(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的资产ID", c)
		return
	}
	rng := c.Query("range")
	switch rng {
	case "":
		rng = "1h"
	case "1h", "1d", "7d", "30d":
	default:
		response.FailWithMessage("range 仅支持 1h/1d/7d/30d", c)
		return
	}
	trend, err := promService.GetTrend(c.Request.Context(), id, rng)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Field("assetId", id).Error("获取资产指标趋势失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(trend, "获取成功", c)
}

// GetPrometheusSD
// @Tags      ServerAgent
// @Summary   HTTP SD 端点(Prometheus http_sd_configs 拉资产清单;Bearer sd-token 鉴权)
// @Produce   application/json
// @Param     Authorization  header  string  true  "Bearer <server.prometheus.sd-token>"
// @Success   200  {object}  []object
// @Router    /prometheus/sd [get]
func (a *AgentApi) GetPrometheusSD(c *gin.Context) {
	token := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
	expect := global.OPS_CONFIG.ServerModule.Prometheus.SDToken
	if expect == "" || token == "" || token != expect {
		// 不区分 401 细节:统一按未授权语义(HTTP SD 对非 200 会报错并保持旧 targets)
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	targets := promService.SDTargets(c.Request.Context())
	if targets == nil {
		targets = []map[string]any{}
	}
	c.JSON(200, targets)
}
