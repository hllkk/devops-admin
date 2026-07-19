package system

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// DeptApi 部门管理(对齐前端 /system/dept/* 资源)
type DeptApi struct{}

// GetDeptList
// @Tags      SysDept
// @Summary   获取部门列表(树形平表,前端组装树)
// @Produce   application/json
// @Param     deptName  query  string  false  "部门名称(模糊匹配)"
// @Param     status    query  string  false  "部门状态(精确 0正常1停用)"
// @Success   200  {object}  response.Response{data=[]system.SysDepartment,msg=string}
// @Router    /system/dept/list [get]
func (a *DeptApi) GetDeptList(c *gin.Context) {
	var q systemReq.DeptSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, err := departmentService.GetDeptList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取部门列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// GetExcludeDeptList
// @Tags      SysDept
// @Summary   获取排除指定部门及其子部门的列表(选父级用)
// @Produce   application/json
// @Param     deptId  path  int  true  "部门ID"
// @Success   200  {object}  response.Response{data=[]system.SysDepartment,msg=string}
// @Router    /system/dept/list/exclude/{deptId} [get]
func (a *DeptApi) GetExcludeDeptList(c *gin.Context) {
	deptId, _ := strconv.ParseInt(c.Param("deptId"), 10, 64)
	list, err := departmentService.GetExcludeDeptList(c.Request.Context(), deptId)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取排除部门列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// CreateDept
// @Tags      SysDept
// @Summary   新增部门
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.DeptOperateParams  true  "部门信息"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/dept [post]
func (a *DeptApi) CreateDept(c *gin.Context) {
	var req systemReq.DeptOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := departmentService.CreateDept(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "新增成功", c)
}

// UpdateDept
// @Tags      SysDept
// @Summary   修改部门
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.DeptOperateParams  true  "部门信息(含 deptId)"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/dept [put]
func (a *DeptApi) UpdateDept(c *gin.Context) {
	var req systemReq.DeptOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := departmentService.UpdateDept(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "修改成功", c)
}

// BatchDeleteDept
// @Tags      SysDept
// @Summary   批量删除部门
// @Produce   application/json
// @Param     ids  path  string  true  "部门ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /system/dept/{ids} [delete]
func (a *DeptApi) BatchDeleteDept(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的部门ID: "+s, c)
			return
		}
		ids = append(ids, n)
	}
	if err := departmentService.DeleteDept(c.Request.Context(), ids); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}

// GetDeptOption
// @Tags      SysDept
// @Summary   获取部门选择框列表
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]system.SysDepartment,msg=string}
// @Router    /system/dept/optionselect [get]
func (a *DeptApi) GetDeptOption(c *gin.Context) {
	list, err := departmentService.GetDeptOptionList(c.Request.Context())
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}
