package system

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/model/system"
)

// SettingApi 系统设置 API：参数提取、调用 Service、统一响应。
type SettingApi struct{}

// GetPublicSystemSettings 获取公开系统设置（登录页使用，无需登录）
// @Tags 系统设置
// @Summary 获取公开系统设置
// @Produce json
// @Success 200 {object} response.Response{data=system.PublicSystemSettings}
// @Router /system/setting/public [get]
func (s *SettingApi) GetPublicSystemSettings(c *gin.Context) {
	settings, err := settingService.GetPublicSystemSettings()
	if err != nil {
		global.OPS_LOG.Error("获取公开系统设置失败", zap.Error(err))
		response.FailWithMessage("获取系统设置失败", c)
		return
	}
	response.OkWithDetailed(settings, "获取系统设置成功", c)
}

// GetSystemSettings 获取系统设置（需管理员）
// @Tags 系统设置
// @Summary 获取系统设置
// @Produce json
// @Success 200 {object} response.Response{data=system.SystemSettings}
// @Router /system/setting [get]
func (s *SettingApi) GetSystemSettings(c *gin.Context) {
	settings, err := settingService.GetSystemSettings()
	if err != nil {
		global.OPS_LOG.Error("获取系统设置失败", zap.Error(err))
		response.FailWithMessage("获取系统设置失败", c)
		return
	}
	response.OkWithDetailed(settings, "获取系统设置成功", c)
}

// UpdateSystemSettings 更新系统设置（需管理员）
// @Tags 系统设置
// @Summary 更新系统设置
// @Produce json
// @Param data body system.SystemSettings true "系统设置"
// @Success 200 {object} response.Response
// @Router /system/setting [put]
func (s *SettingApi) UpdateSystemSettings(c *gin.Context) {
	var settings system.SystemSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		response.FailWithMessage("参数校验失败: "+err.Error(), c)
		return
	}
	if err := settingService.UpdateSystemSettings(settings); err != nil {
		global.OPS_LOG.Error("更新系统设置失败", zap.Error(err))
		response.FailWithMessage("更新系统设置失败", c)
		return
	}
	response.OkWithMessage("更新系统设置成功", c)
}
