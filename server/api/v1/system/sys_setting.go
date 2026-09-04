package system

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// SettingApi 系统设置聚合接口(对齐前端 /system/setting GET/PUT)
type SettingApi struct{}

// GetSetting
// @Tags      SysSetting
// @Summary   获取系统设置(管理员)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=systemReq.SettingConfig,msg=string}
// @Router    /system/setting [get]
func (s *SettingApi) GetSetting(c *gin.Context) {
	data, err := settingService.Get(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取系统设置失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(data, "获取成功", c)
}

// UpdateSetting
// @Tags      SysSetting
// @Summary   更新系统设置(管理员)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.SettingConfig  true  "系统设置(general/security/ldap/notify/auth 任一可选)"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/setting [put]
func (s *SettingApi) UpdateSetting(c *gin.Context) {
	var req systemReq.SettingConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := settingService.Set(c.Request.Context(), req); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("更新系统设置失败")
		response.FailWithMessage("保存失败", c)
		return
	}
	response.OkWithDetailed(true, "保存成功", c)
}

// GetPublicSetting
// @Tags      SysSetting
// @Summary   获取公开系统设置(登录页,免鉴权)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=systemReq.PublicSetting,msg=string}
// @Router    /system/setting/public [get]
func (s *SettingApi) GetPublicSetting(c *gin.Context) {
	data := settingService.GetPublic(c.Request.Context())
	response.OkWithDetailed(data, "获取成功", c)
}

// TestEmail
// @Tags      SysSetting
// @Summary   发送测试邮件(使用当前表单值,无需先保存)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.TestEmailReq  true  "邮件配置+收件人地址"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/setting/notify/test-email [post]
func (s *SettingApi) TestEmail(c *gin.Context) {
	var req systemReq.TestEmailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	cfg := system.SysNotifyConfig{
		EmailHost:     req.EmailHost,
		EmailPort:     req.EmailPort,
		EmailUsername: req.EmailUsername,
		EmailPassword: req.EmailPassword,
		EmailFromAddr: req.EmailFromAddr,
		EmailFromName: req.EmailFromName,
		EmailSSLMode:  req.EmailSSLMode,
	}
	if err := notifyConfigService.SendTestEmail(cfg, req.TestTo); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("测试邮件发送失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "测试邮件发送成功", c)
}

// TestWecomApp
// @Tags      SysSetting
// @Summary   发送企微应用消息测试(用已保存企微凭证,表单跳转地址未保存也可测)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.TestWecomAppReq  true  "目标用户+跳转base"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/setting/notify/test-wecom-app [post]
func (s *SettingApi) TestWecomApp(c *gin.Context) {
	var req systemReq.TestWecomAppReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := notifySendService.SendTestWecomApp(c.Request.Context(), req.TestUserId, req.RedirectBase); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("企微应用消息测试失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "测试消息发送成功", c)
}

// TestWecomBot
// @Tags      SysSetting
// @Summary   发送企微群机器人测试(按群主键实测已录入群的 webhook)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.TestWecomBotReq  true  "目标群主键"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/setting/notify/test-wecom-bot [post]
func (s *SettingApi) TestWecomBot(c *gin.Context) {
	var req systemReq.TestWecomBotReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := notifySendService.SendTestWecomBot(c.Request.Context(), req.GroupId); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("企微群机器人测试失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "测试消息发送成功", c)
}

// WecomBotGroupList
// @Tags      SysSetting
// @Summary   群机器人群列表
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]system.SysWecomBotGroup,msg=string}
// @Router    /system/setting/notify/wecom-bot-group [get]
func (s *SettingApi) WecomBotGroupList(c *gin.Context) {
	groups, err := wecomBotGroupService.List(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("群机器人群列表查询失败")
		response.FailWithMessage("查询失败", c)
		return
	}
	response.OkWithDetailed(groups, "获取成功", c)
}

// WecomBotGroupCreate
// @Tags      SysSetting
// @Summary   新增群机器人群(群聊名称+webhook)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.WecomBotGroupCreateReq  true  "群聊名称+webhook地址"
// @Success   200   {object}  response.Response{data=system.SysWecomBotGroup,msg=string}
// @Router    /system/setting/notify/wecom-bot-group [post]
func (s *SettingApi) WecomBotGroupCreate(c *gin.Context) {
	var req systemReq.WecomBotGroupCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	group, err := wecomBotGroupService.Create(c.Request.Context(), req.GroupName, req.WebhookUrl)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("群机器人群新增失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(group, "新增成功", c)
}

// WecomBotGroupDelete
// @Tags      SysSetting
// @Summary   删除群机器人群(软删,即时生效)
// @Produce   application/json
// @Param     id   path  int  true  "群主键"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /system/setting/notify/wecom-bot-group/{id} [delete]
func (s *SettingApi) WecomBotGroupDelete(c *gin.Context) {
	var req systemReq.WecomBotGroupIdReq
	if err := c.ShouldBindUri(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := wecomBotGroupService.Delete(c.Request.Context(), req.Id); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("群机器人群删除失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}

// WecomDomainVerify 企业微信可信域名校验：响应 /WW_verify_*.txt 请求。
// 企业微信会请求 http://your-domain/WW_verify_xxxx.txt 验证域名所有权，
// 文件名与内容由管理员在认证配置中填写，系统从 DB 读取并自动响应，无需手动放置文件。
func (s *SettingApi) WecomDomainVerify(c *gin.Context) {
	// 路由 /WW_verify_:name 的 param 按段匹配已含 .txt 后缀，直接拼回完整文件名
	filename := "WW_verify_" + c.Param("name")

	cfg := settingService.CurrentAuth(c.Request.Context())
	// 比对前规范化配置值（trim/前缀后缀补全），兼容存量脏数据；404 落 Warn 便于区分
	// "未配置"与"文件名不一致"（企微后台重新申请校验会轮换文件名），Warn 级不进 sys_error 防公网扫描刷表
	fileName := system.NormalizeWecomDomainFileName(cfg.WecomDomainFileName)
	fileContent := strings.TrimSpace(cfg.WecomDomainFileContent)
	if fileName == "" || fileContent == "" {
		logger.WithCtx(c.Request.Context()).Mod("biz").Field("filename", filename).Warn("企微可信域名校验: 系统设置未配置校验文件, 返回404")
		c.String(http.StatusNotFound, "not found")
		return
	}
	if fileName != filename {
		logger.WithCtx(c.Request.Context()).Mod("biz").Field("filename", filename).Field("configured", fileName).Warn("企微可信域名校验: 请求文件名与配置不一致, 返回404")
		c.String(http.StatusNotFound, "not found")
		return
	}
	c.String(http.StatusOK, fileContent)
}
