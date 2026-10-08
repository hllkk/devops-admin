package server

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	serverSvc "github.com/hllkk/devops-admin/server/service/server"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// AgentApi agent 安装流与注册心跳（安装接口挂 PrivateGroup 走 JWT/casbin；
// register/heartbeat 挂 PublicGroup 由 agent token 自鉴权，对齐 Skill Agent 直连先例）。
type AgentApi struct{}

// InstallAgent
// @Tags      ServerAgent
// @Summary   触发 agent 自动安装(异步,纯公钥模式:SSH 信任已在资产录入验证时建立,无需密码)
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
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Field("assetId", id).Error("触发 agent 安装失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{"taskId": taskId}, "安装任务已启动", c)
}

// GetInstallStatus
// @Tags      ServerAgent
// @Summary   轮询 agent 安装任务状态
// @Produce   application/json
// @Param     taskId  path  string  true  "安装任务ID"
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

// AgentRegister
// @Tags      ServerAgent
// @Summary   agent 注册(一次性注册 token 换持久 token;PublicGroup 无 JWT)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  object{token=string,hostname=string,arch=string,os=string,agentVersion=string}  true  "注册信息"
// @Success   200  {object}  response.Response{data=object{agentId=string,token=string,heartbeatInterval=int},msg=string}
// @Router    /server/agent/register [post]
func (a *AgentApi) AgentRegister(c *gin.Context) {
	var req agentRegisterParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	regReq := serverSvc.AgentRegisterRequest{
		Token: req.Token, Hostname: req.Hostname, Arch: req.Arch, Os: req.Os, AgentVersion: req.AgentVersion,
	}
	resp, err := agentRegistryService.Register(c.Request.Context(), regReq)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Error("agent 注册失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(resp, "注册成功", c)
}

// AgentHeartbeat
// @Tags      ServerAgent
// @Summary   agent 心跳(Bearer token 自鉴权;回写 running + 时间戳)
// @Accept    application/json
// @Produce   application/json
// @Param     Authorization  header  string  false  "Bearer <持久token>(或 ?token= 查询参数)"
// @Param     data  body  object{agentVersion=string}  false  "心跳信息(版本可选)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /server/agent/heartbeat [post]
func (a *AgentApi) AgentHeartbeat(c *gin.Context) {
	token := extractAgentToken(c)
	if token == "" {
		response.NoAuth("未提供 agent 认证凭证(Bearer 头或 token 参数)", c)
		return
	}
	var req agentHeartbeatParams
	_ = c.ShouldBindJSON(&req) // 版本字段可选,body 缺失不拒
	hbReq := serverSvc.AgentHeartbeatRequest{AgentVersion: req.AgentVersion}
	if err := agentRegistryService.Heartbeat(c.Request.Context(), token, hbReq); err != nil {
		// token 无效按 401 语义(agent 据此触发重注册)
		response.NoAuth(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "心跳成功", c)
}

// extractAgentToken 双通道取 agent 持久 token（Authorization: Bearer / ?token=）。
func extractAgentToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	return strings.TrimSpace(c.Query("token"))
}

// agentRegisterParams/agentHeartbeatParams 请求结构。
type agentRegisterParams struct {
	Token        string `json:"token" binding:"required" form:"token"`
	Hostname     string `json:"hostname" form:"hostname"`
	Arch         string `json:"arch" form:"arch"`
	Os           string `json:"os" form:"os"`
	AgentVersion string `json:"agentVersion" form:"agentVersion"`
}

type agentHeartbeatParams struct {
	AgentVersion string `json:"agentVersion" form:"agentVersion"`
}

// RestartAgent
// @Tags      ServerAgent
// @Summary   重启目标机 agent(异步,公钥通道)
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
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Field("assetId", id).Error("触发 agent 重启失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{"taskId": taskId}, "重启任务已启动", c)
}

// UninstallAgent
// @Tags      ServerAgent
// @Summary   卸载目标机 agent(异步,公钥通道;平台公钥保留可重装)
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
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Field("assetId", id).Error("触发 agent 卸载失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{"taskId": taskId}, "卸载任务已启动", c)
}

// GetAssetSnapshot
// @Tags      ServerAsset
// @Summary   资产实时快照(agent 心跳上报的轻量指标;agent 未上报时 stale=true)
// @Produce   application/json
// @Param     id  path  int  true  "资产ID"
// @Success   200 {object}  response.Response{data=response.AgentSnapshot,msg=string}
// @Router    /server/asset/{id}/snapshot [get]
func (a *AgentApi) GetAssetSnapshot(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的资产ID", c)
		return
	}
	snap := agentRegistryService.GetSnapshot(c.Request.Context(), id)
	response.OkWithDetailed(snap, "获取成功", c)
}
