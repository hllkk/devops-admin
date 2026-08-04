package system

import (
	"net/http"

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

// WecomDomainVerify 企业微信可信域名校验：响应 /WW_verify_*.txt 请求。
// 企业微信会请求 http://your-domain/WW_verify_xxxx.txt 验证域名所有权，
// 文件名与内容由管理员在认证配置中填写，系统从 DB 读取并自动响应，无需手动放置文件。
func (s *SettingApi) WecomDomainVerify(c *gin.Context) {
	name := c.Param("name")
	filename := "WW_verify_" + name + ".txt"

	cfg := settingService.CurrentAuth(c.Request.Context())
	if cfg.WecomDomainFileName == "" || cfg.WecomDomainFileContent == "" {
		c.String(http.StatusNotFound, "not found")
		return
	}
	if cfg.WecomDomainFileName != filename {
		c.String(http.StatusNotFound, "not found")
		return
	}
	c.String(http.StatusOK, cfg.WecomDomainFileContent)
}
