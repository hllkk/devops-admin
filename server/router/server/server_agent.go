package server

import "github.com/gin-gonic/gin"

// AgentRouter 采集端(node_exporter)安装/运维 + Prometheus 查询代理路由。
// 私有组挂 PrivateGroup(JWT/casbin,菜单 ApiPrefix /server/asset/* 覆盖);
// HTTP SD 端点挂 PublicGroup(Bearer sd-token 自鉴权,供 Prometheus http_sd_configs)。
type AgentRouter struct{}

// InitAgentRouter 挂载双组路由。
func (a *AgentRouter) InitAgentRouter(Router *gin.RouterGroup, PublicRouter *gin.RouterGroup) {
	r := Router.Group("server/asset")
	{
		r.POST(":id/install-agent", agentApi.InstallAgent)         // 触发采集端安装(异步,纯公钥模式)
		r.POST(":id/restart-agent", agentApi.RestartAgent)         // 重启采集端(异步,公钥通道)
		r.POST(":id/uninstall-agent", agentApi.UninstallAgent)     // 卸载采集端(异步,公钥通道)
		r.GET(":id/snapshot", agentApi.GetAssetSnapshot)           // 资产实时快照(代查 Prometheus)
		r.GET(":id/metrics", agentApi.GetAssetMetricsTrend)        // 资产指标趋势(代查 Prometheus)
		r.GET("install-status/:taskId", agentApi.GetInstallStatus) // 运维任务状态轮询(装/重/卸共用;静态段在前)
	}
	// HTTP SD:Prometheus http_sd_configs 拉资产清单(Bearer sd-token 自鉴权)
	PublicRouter.GET("prometheus/sd", agentApi.GetPrometheusSD)
}
