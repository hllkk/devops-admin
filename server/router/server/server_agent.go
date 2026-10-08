package server

import "github.com/gin-gonic/gin"

// AgentRouter agent 安装流与注册心跳路由。
// 安装/状态挂 PrivateGroup(JWT/casbin,菜单 ApiPrefix /server/asset/* 覆盖);
// register/heartbeat 挂 PublicGroup(agent 无登录态,注册 token/持久 token 自鉴权,
// 对齐 Skill Agent 直连双组拆分先例)。
type AgentRouter struct{}

// InitAgentRouter 挂载双组路由。
func (a *AgentRouter) InitAgentRouter(Router *gin.RouterGroup, PublicRouter *gin.RouterGroup) {
	r := Router.Group("server/asset")
	{
		r.POST(":id/install-agent", agentApi.InstallAgent)         // 触发安装(异步,纯公钥模式)
		r.POST(":id/restart-agent", agentApi.RestartAgent)         // 重启 agent(异步,公钥通道)
		r.POST(":id/uninstall-agent", agentApi.UninstallAgent)     // 卸载 agent(异步,公钥通道)
		r.GET(":id/snapshot", agentApi.GetAssetSnapshot)           // 资产实时快照(agent 心跳指标)
		r.GET("install-status/:taskId", agentApi.GetInstallStatus) // 运维任务状态轮询(装/重/卸共用;静态段在前)
	}
	pub := PublicRouter.Group("server/agent")
	{
		pub.POST("register", agentApi.AgentRegister)   // agent 注册(一次性 token 换持久 token)
		pub.POST("heartbeat", agentApi.AgentHeartbeat) // agent 心跳(Bearer 自鉴权)
	}
}
