package system

import (
	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
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
// @Param     data  body  systemReq.SettingConfig  true  "系统设置(general/security 任一可选)"
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
