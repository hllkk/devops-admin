package system

import (
	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// UpgradeApi 在线升级接口(实际执行者为 updater sidecar,本层只做版本展示/检查/转发/状态代理)
type UpgradeApi struct{}

// GetVersion
// @Tags      SysUpgrade
// @Summary   版本信息(「关于」弹窗;登录即可)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=response.VersionInfo,msg=string}
// @Router    /system/upgrade/version [get]
func (u *UpgradeApi) GetVersion(c *gin.Context) {
	response.OkWithDetailed(upgradeService.GetVersion(), "获取成功", c)
}

// CheckUpdate
// @Tags      SysUpgrade
// @Summary   检查更新(拉取发布服务器 manifest 与当前版本比对)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=response.UpgradeCheckResult,msg=string}
// @Router    /system/upgrade/check [get]
func (u *UpgradeApi) CheckUpdate(c *gin.Context) {
	data, err := upgradeService.CheckUpdate(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("检查更新失败")
		response.FailWithMessage("检查更新失败", c)
		return
	}
	response.OkWithDetailed(data, "获取成功", c)
}

// StartUpgrade
// @Tags      SysUpgrade
// @Summary   触发在线升级(转发 updater 执行,进度轮询 /system/upgrade/status;走 casbin 菜单授权)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=response.UpgradeStartResult,msg=string}
// @Router    /system/upgrade/start [post]
func (u *UpgradeApi) StartUpgrade(c *gin.Context) {
	data, err := upgradeService.StartUpgrade(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("触发在线升级失败")
		response.FailWithMessage("触发升级失败", c)
		return
	}
	response.OkWithDetailed(data, "获取成功", c)
}

// GetUpgradeStatus
// @Tags      SysUpgrade
// @Summary   获取升级状态(代理 updater 状态机:downloading/verifying/unpacking/installing/success/failed/unreachable)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=response.UpgradeStateInfo,msg=string}
// @Router    /system/upgrade/status [get]
func (u *UpgradeApi) GetUpgradeStatus(c *gin.Context) {
	response.OkWithDetailed(upgradeService.GetStatus(c.Request.Context()), "获取成功", c)
}
