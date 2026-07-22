package system

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils/excel"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// LoginLogApi 登录日志管理(对齐前端 /log/loginlog/* 资源)
type LoginLogApi struct{}

// GetLoginLogList
// @Tags      SysLoginLog
// @Summary   分页获取登录日志列表
// @Produce   application/json
// @Param     userName  query  string  false  "用户账号(模糊匹配)"
// @Param     ipaddr    query  string  false  "登录IP(模糊匹配)"
// @Param     status    query  string  false  "登录状态(0成功 1失败)"
// @Param     pageNum   query  int     true   "页码"
// @Param     pageSize  query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]system.SysLoginLog},msg=string}
// @Router    /log/loginlog/list [get]
func (l *LoginLogApi) GetLoginLogList(c *gin.Context) {
	var q systemReq.LoginLogSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := loginLogService.GetLoginLogList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取登录日志列表失败")
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

// BatchDeleteLoginLog
// @Tags      SysLoginLog
// @Summary   批量删除登录日志
// @Produce   application/json
// @Param     ids  path  string  true  "登录日志ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /log/loginlog/{ids} [delete]
func (l *LoginLogApi) BatchDeleteLoginLog(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的登录日志ID: "+s, c)
			return
		}
		ids = append(ids, n)
	}
	if err := loginLogService.DeleteLoginLog(c.Request.Context(), ids); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}

// CleanLoginLog
// @Tags      SysLoginLog
// @Summary   清空登录日志
// @Produce   application/json
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /log/loginlog/clean [delete]
func (l *LoginLogApi) CleanLoginLog(c *gin.Context) {
	if err := loginLogService.CleanLoginLog(c.Request.Context()); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "清空成功", c)
}

// UnlockLoginLog
// @Tags      SysLoginLog
// @Summary   解锁账号(清除登录失败计数与锁)
// @Produce   application/json
// @Param     username  path  string  true  "用户账号"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /log/loginlog/unlock/{username} [get]
func (l *LoginLogApi) UnlockLoginLog(c *gin.Context) {
	if err := loginLogService.UnlockLoginLog(c.Request.Context(), c.Param("username")); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "解锁成功", c)
}

// ExportLoginLog
// @Tags      SysLoginLog
// @Summary   导出登录日志(Excel)
// @Router    /log/loginlog/export [post]
func (l *LoginLogApi) ExportLoginLog(c *gin.Context) {
	var q systemReq.LoginLogSearch
	if err := c.ShouldBind(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, err := loginLogService.ExportLoginLogList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("导出登录日志失败")
		response.FailWithMessage("导出失败", c)
		return
	}
	buf, err := excel.Export(list, loginLogHeaders, "登录日志")
	if err != nil {
		response.FailWithMessage("导出失败", c)
		return
	}
	writeXlsx(c, "登录日志", buf)
}
