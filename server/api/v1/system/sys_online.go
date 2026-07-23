package system

import (
	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// OnlineApi 在线设备管理(对齐前端 /monitor/online 资源,个人中心视角:仅当前用户自己)。
type OnlineApi struct{}

// GetOnlineList
// @Tags      Monitor
// @Summary   获取当前用户在线设备列表
// @Produce   application/json
// @Param     ipaddr    query  string  false  "登录IP(模糊匹配)"
// @Param     pageNum   query  int     true   "页码"
// @Param     pageSize  query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]system.OnlineDevice},msg=string}
// @Router    /monitor/online [get]
func (o *OnlineApi) GetOnlineList(c *gin.Context) {
	var q systemReq.OnlineSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userId := utils.GetUserID(c)
	list, total, err := onlineService.GetOnlineList(c.Request.Context(), userId, q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取在线设备列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows:     list,
		Total:    total,
		PageNum:  q.PageNum,
		PageSize: q.PageSize,
	}, "获取成功", c)
}

// KickOnlineDevice
// @Tags      Monitor
// @Summary   强制下线当前用户的指定在线设备
// @Produce   application/json
// @Param     tokenId  path  string  true  "令牌ID"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /monitor/online/myself/{tokenId} [delete]
func (o *OnlineApi) KickOnlineDevice(c *gin.Context) {
	tokenId := c.Param("tokenId")
	userId := utils.GetUserID(c)
	if err := onlineService.KickSession(c.Request.Context(), userId, tokenId); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "强制下线成功", c)
}
