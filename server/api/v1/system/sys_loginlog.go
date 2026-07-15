package system

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hllkk/devops-admin/server/model/common/response"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
)

// LoginLogApi 登录日志接口，对接前端 /log/loginlog/* 契约。
type LoginLogApi struct{}

// GetLoginLogList 分页查询登录日志
// @Tags     Log
// @Summary  登录日志列表
// @Produce  application/json
// @Param    data  query  systemReq.LoginLogSearch  true  "分页与过滤参数"
// @Success  200   {object}  response.Response{data=object}
// @Router   /log/loginlog/list [get]
func (b *LoginLogApi) GetLoginLogList(c *gin.Context) {
	var search systemReq.LoginLogSearch
	if err := c.ShouldBindQuery(&search); err != nil {
		response.FailWithMessage("参数校验不通过", c)
		return
	}
	list, total, err := loginLogService.GetLoginLogList(search)
	if err != nil {
		response.FailWithMessage("查询失败", c)
		return
	}
	response.OkWithDetailed(gin.H{
		"pageNum":  search.PageNum,
		"pageSize": search.PageSize,
		"total":    total,
		"rows":     list,
	}, "查询成功", c)
}

// HandleLoginLogDelete 处理 DELETE /log/loginlog/:action：action="clean" 清空，否则为逗号分隔 infoId 批量删除。
// gin 不允许同层共存静态段 clean 与参数段 :ids，故合并到一个参数路由。
// @Tags     Log
// @Summary  删除/清空登录日志
// @Produce  application/json
// @Router   /log/loginlog/{action} [delete]
func (b *LoginLogApi) HandleLoginLogDelete(c *gin.Context) {
	action := c.Param("action")
	if action == "clean" {
		if err := loginLogService.CleanLoginLog(); err != nil {
			response.FailWithMessage("清空失败", c)
			return
		}
		response.OkWithMessage("清空成功", c)
		return
	}
	parts := strings.Split(action, ",")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		if id, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		response.FailWithMessage("未指定有效ID", c)
		return
	}
	if err := loginLogService.DeleteLoginLog(ids); err != nil {
		response.FailWithMessage("删除失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// UnlockLoginLog 解锁用户：清除该账号登录失败计数（取消强制验证码触发）。
// @Tags     Log
// @Summary  解锁登录
// @Produce  application/json
// @Router   /log/loginlog/unlock/{username} [get]
func (b *LoginLogApi) UnlockLoginLog(c *gin.Context) {
	captchaService.UnlockUser(c.Param("username"))
	response.OkWithMessage("解锁成功", c)
}
