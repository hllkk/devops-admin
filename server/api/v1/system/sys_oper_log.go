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

// OperLogApi 操作日志管理(对齐前端 /log/operlog/* 资源)
type OperLogApi struct{}

// GetOperLogList
// @Tags      SysOperLog
// @Summary   分页获取操作日志列表
// @Produce   application/json
// @Param     title                  query  string  false  "系统模块(模糊匹配)"
// @Param     businessType           query  string  false  "操作类型(0~9)"
// @Param     operName               query  string  false  "操作人员(模糊匹配)"
// @Param     operIp                 query  string  false  "操作IP(模糊匹配)"
// @Param     status                 query  string  false  "操作状态(0正常 1异常)"
// @Param     params[beginTime]      query  string  false  "开始时间(yyyy-MM-dd HH:mm:ss)"
// @Param     params[endTime]        query  string  false  "结束时间(yyyy-MM-dd HH:mm:ss)"
// @Param     pageNum                query  int     true   "页码"
// @Param     pageSize               query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]system.SysOperLog},msg=string}
// @Router    /log/operlog/list [get]
func (o *OperLogApi) GetOperLogList(c *gin.Context) {
	var q systemReq.OperLogSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	// 前端 qs.stringify 将 params:{beginTime,endTime} 序列化为 params[beginTime]/params[endTime](bracket),
	// gin struct binding 不支持 bracket 嵌套, 这里显式从 query 取后赋给 BeginTime/EndTime。
	q.BeginTime = c.Query("params[beginTime]")
	q.EndTime = c.Query("params[endTime]")
	list, total, err := operLogService.GetOperLogList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取操作日志列表失败")
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

// BatchDeleteOperLog
// @Tags      SysOperLog
// @Summary   批量删除操作日志
// @Produce   application/json
// @Param     ids  path  string  true  "操作日志ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /log/operlog/{ids} [delete]
func (o *OperLogApi) BatchDeleteOperLog(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的操作日志ID: "+s, c)
			return
		}
		ids = append(ids, n)
	}
	if err := operLogService.DeleteOperLog(c.Request.Context(), ids); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}

// CleanOperLog
// @Tags      SysOperLog
// @Summary   清空操作日志
// @Produce   application/json
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /log/operlog/clean [delete]
func (o *OperLogApi) CleanOperLog(c *gin.Context) {
	if err := operLogService.CleanOperLog(c.Request.Context()); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "清空成功", c)
}

// ExportOperLog
// @Tags      SysOperLog
// @Summary   导出操作日志(Excel)
// @Router    /log/operlog/export [post]
func (o *OperLogApi) ExportOperLog(c *gin.Context) {
	var q systemReq.OperLogSearch
	if err := c.ShouldBind(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	// 与 GetOperLogList 一致:bracket 形式 params[beginTime/endTime] 经表单体传输,由显式取值赋给 BeginTime/EndTime
	q.BeginTime = c.PostForm("params[beginTime]")
	q.EndTime = c.PostForm("params[endTime]")
	list, err := operLogService.ExportOperLogList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("导出操作日志失败")
		response.FailWithMessage("导出失败", c)
		return
	}
	buf, err := excel.Export(list, operLogHeaders, "操作日志")
	if err != nil {
		response.FailWithMessage("导出失败", c)
		return
	}
	writeXlsx(c, "操作日志", buf)
}
