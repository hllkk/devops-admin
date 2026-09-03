package gateway

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// AdoptionApi 覆盖率/采用度(P3，对齐前端 /gateway/adoption/* 资源)。
// 挂 AI审计目录菜单(route.ai-audit_adoption)，user 角色不授：决策层/管理员视角，
// 普通用户由 casbin 菜单授权挡住(规避 AIHelms 读端点零权限的坑)。
type AdoptionApi struct{}

// GetAdoptionOverview
// @Tags      GatewayAdoption
// @Summary   覆盖率总览(KPI 含环比+DAU 按日趋势)
// @Produce   application/json
// @Param     startDate    query  string  false  "开始业务日(YYYY-MM-DD,缺省本月首日)"
// @Param     endDate      query  string  false  "结束业务日(缺省今天)"
// @Param     departmentId query  string  false  "部门筛选(含子树,0=不限)"
// @Param     userId       query  string  false  "用户筛选(0=不限)"
// @Param     aiKeyId      query  string  false  "密钥筛选(0=不限)"
// @Param     model        query  string  false  "模型名(精确)"
// @Param     provider     query  string  false  "供应商(精确)"
// @Success   200  {object}  response.Response{data=response.AdoptionOverview,msg=string}
// @Router    /gateway/adoption/overview [get]
func (a *AdoptionApi) GetAdoptionOverview(c *gin.Context) {
	var f gatewayReq.AdoptionSearch
	if err := c.ShouldBindQuery(&f); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	ov, err := adoptionService.GetAdoptionOverview(c.Request.Context(), &f)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取覆盖率总览失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(ov, "获取成功", c)
}

// GetAdoptionDepartments
// @Tags      GatewayAdoption
// @Summary   部门覆盖率明细(全部部门含零调用,激活/成员/消耗)
// @Produce   application/json
// @Param     startDate    query  string  false  "开始业务日"
// @Param     endDate      query  string  false  "结束业务日"
// @Param     departmentId query  string  false  "部门筛选(含子树)"
// @Success   200  {object}  response.Response{data=[]response.AdoptionDeptRow,msg=string}
// @Router    /gateway/adoption/departments [get]
func (a *AdoptionApi) GetAdoptionDepartments(c *gin.Context) {
	var f gatewayReq.AdoptionSearch
	if err := c.ShouldBindQuery(&f); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	rows, err := adoptionService.GetAdoptionDepartments(c.Request.Context(), &f)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取部门覆盖率失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(rows, "获取成功", c)
}

// GetAdoptionDeptUsers
// @Tags      GatewayAdoption
// @Summary   部门成员明细下钻(含未激活成员,兼未使用人员清单)
// @Produce   application/json
// @Param     id         path   string  true  "部门ID"
// @Param     startDate  query  string  false  "开始业务日"
// @Param     endDate    query  string  false  "结束业务日"
// @Success   200  {object}  response.Response{data=[]response.AdoptionUserRow,msg=string}
// @Router    /gateway/adoption/departments/{id}/users [get]
func (a *AdoptionApi) GetAdoptionDeptUsers(c *gin.Context) {
	deptId, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var f gatewayReq.AdoptionSearch
	if err := c.ShouldBindQuery(&f); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	rows, err := adoptionService.GetAdoptionDeptUsers(c.Request.Context(), deptId, &f)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取部门成员覆盖率失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(rows, "获取成功", c)
}

// GetAdoptionModels
// @Tags      GatewayAdoption
// @Summary   模型分布(LLM 维,调用/成本占比)
// @Produce   application/json
// @Param     startDate    query  string  false  "开始业务日"
// @Param     endDate      query  string  false  "结束业务日"
// @Param     departmentId query  string  false  "部门筛选(含子树)"
// @Success   200  {object}  response.Response{data=[]response.AdoptionModelRow,msg=string}
// @Router    /gateway/adoption/models [get]
func (a *AdoptionApi) GetAdoptionModels(c *gin.Context) {
	var f gatewayReq.AdoptionSearch
	if err := c.ShouldBindQuery(&f); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	rows, err := adoptionService.GetAdoptionModels(c.Request.Context(), &f)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取模型分布失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(rows, "获取成功", c)
}
