package gateway

import (
	"bytes"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// ReportApi 效能报告(P3，对齐前端 /gateway/report/* 资源)。
// 挂 AI审计目录菜单(route.ai-audit_report)，user 角色不授：管理员/决策层视角。
// 手动生成不发通知(管理员自己在看)；定时路径的通知由 timer 闭包发送。
type ReportApi struct{}

// GetReportList
// @Tags      GatewayReport
// @Summary   效能报告分页列表(不带内容大字段)
// @Produce   application/json
// @Param     pageNum    query  int    false  "页码"
// @Param     pageSize   query  int    false  "页大小(上限100)"
// @Param     reportType query  string false  "类型筛选(weekly/monthly/custom,空=全部)"
// @Success   200  {object}  response.Response{data=response.PageResult{Rows=[]response.EfficiencyReportView},msg=string}
// @Router    /gateway/report/list [get]
func (a *ReportApi) GetReportList(c *gin.Context) {
	var q gatewayReq.ReportSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	rows, total, err := reportService.GetReportList(c.Request.Context(), &q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取效能报告列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: rows, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}

// GetReport
// @Tags      GatewayReport
// @Summary   效能报告详情(结构化内容+Markdown)
// @Produce   application/json
// @Param     id path string true "报告ID"
// @Success   200  {object}  response.Response{data=response.EfficiencyReportView,msg=string}
// @Router    /gateway/report/{id} [get]
func (a *ReportApi) GetReport(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	view, err := reportService.GetReport(c.Request.Context(), id)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取效能报告详情失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(view, "获取成功", c)
}

// GenerateReport
// @Tags      GatewayReport
// @Summary   手动生成效能报告(weekly/monthly 取上一完整周期,custom 须带起止)
// @Accept    application/json
// @Produce   application/json
// @Param     body body request.ReportGenerateParams true "生成参数"
// @Success   200  {object}  response.Response{data=response.EfficiencyReportView,msg=string}
// @Router    /gateway/report/generate [post]
func (a *ReportApi) GenerateReport(c *gin.Context) {
	var p gatewayReq.ReportGenerateParams
	if err := c.ShouldBindJSON(&p); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	userId := utils.GetUserID(c)
	view, err := reportService.GenerateReport(c.Request.Context(), p.ReportType, p.StartDate, p.EndDate, userId)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("生成效能报告失败")
		response.FailWithMessage("生成失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "生成成功", c)
}

// ExportReport
// @Tags      GatewayReport
// @Summary   导出效能报告 Excel(三 sheet:部门覆盖率/模型分布/用户Top)
// @Produce   application/octet-stream
// @Param     id path string true "报告ID"
// @Success   200  {file} binary
// @Router    /gateway/report/export/{id} [post]
func (a *ReportApi) ExportReport(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	name, data, err := reportService.ExportReport(c.Request.Context(), id)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("导出效能报告失败")
		response.FailWithMessage("导出失败", c)
		return
	}
	writeReportXlsx(c, *name, bytes.NewBuffer(data))
}

// writeReportXlsx 统一输出 xlsx 二进制流(gateway 侧首例,复制自 api/v1/system/sys_export.go
// 的 writeXlsx 同签名同语义；两包各自私有,避免为此提公共包)。
func writeReportXlsx(c *gin.Context, name string, buf *bytes.Buffer) {
	const ct = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	filename := url.QueryEscape(name + ".xlsx")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", ct)
	c.Header("success", "true")
	c.Header("Download-Filename", filename)
	c.Data(200, ct, buf.Bytes())
}
