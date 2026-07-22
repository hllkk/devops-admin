package system

import (
	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// SysErrorApi 错误日志管理(对齐前端 /log/sysError/* 资源)
type SysErrorApi struct{}

// CreateSysError 创建错误日志(前端上报,无鉴权)
// @Tags      SysError
// @Summary   创建错误日志
// @Produce   application/json
// @Param     data  body  system.SysError  true  "错误日志"
// @Success   200  {object}  response.Response{msg=string}
// @Router    /log/sysError/createSysError [post]
func (sysErrorApi *SysErrorApi) CreateSysError(c *gin.Context) {
	ctx := c.Request.Context()
	var sysError system.SysError
	if err := c.ShouldBindJSON(&sysError); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := sysErrorService.CreateSysError(ctx, &sysError); err != nil {
		logger.WithCtx(ctx).Mod("biz").Err(err).Error("创建错误日志失败")
		response.FailWithMessage("创建失败", c)
		return
	}
	response.OkWithMessage("创建成功", c)
}

// DeleteSysError 删除错误日志
// @Tags      SysError
// @Summary   删除错误日志
// @Produce   application/json
// @Param     ID  query  string  true  "错误日志ID"
// @Success   200  {object}  response.Response{msg=string}
// @Router    /log/sysError/deleteSysError [delete]
func (sysErrorApi *SysErrorApi) DeleteSysError(c *gin.Context) {
	ctx := c.Request.Context()
	ID := c.Query("ID")
	if err := sysErrorService.DeleteSysError(ctx, ID); err != nil {
		response.FailWithMessage("删除失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// DeleteSysErrorByIds 批量删除错误日志(POST body 避免 query array 序列化问题)
// @Tags      SysError
// @Summary   批量删除错误日志
// @Accept     application/json
// @Produce   application/json
// @Param     ids  body  []string  true  "错误日志ID列表"
// @Success   200  {object}  response.Response{msg=string}
// @Router    /log/sysError/deleteSysErrorByIds [post]
func (sysErrorApi *SysErrorApi) DeleteSysErrorByIds(c *gin.Context) {
	ctx := c.Request.Context()
	var IDs []string
	if err := c.ShouldBindJSON(&IDs); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := sysErrorService.DeleteSysErrorByIds(ctx, IDs); err != nil {
		response.FailWithMessage("批量删除失败", c)
		return
	}
	response.OkWithMessage("批量删除成功", c)
}

// UpdateSysError 更新错误日志
// @Tags      SysError
// @Summary   更新错误日志
// @Produce   application/json
// @Param     data  body  system.SysError  true  "错误日志"
// @Success   200  {object}  response.Response{msg=string}
// @Router    /log/sysError/updateSysError [put]
func (sysErrorApi *SysErrorApi) UpdateSysError(c *gin.Context) {
	ctx := c.Request.Context()
	var sysError system.SysError
	if err := c.ShouldBindJSON(&sysError); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := sysErrorService.UpdateSysError(ctx, sysError); err != nil {
		response.FailWithMessage("更新失败", c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// FindSysError 根据ID查询错误日志
// @Tags      SysError
// @Summary   根据ID查询错误日志
// @Produce   application/json
// @Param     ID  query  string  true  "错误日志ID"
// @Success   200  {object}  response.Response{data=system.SysError,msg=string}
// @Router    /log/sysError/findSysError [get]
func (sysErrorApi *SysErrorApi) FindSysError(c *gin.Context) {
	ctx := c.Request.Context()
	ID := c.Query("ID")
	res, err := sysErrorService.GetSysError(ctx, ID)
	if err != nil {
		response.FailWithMessage("查询失败", c)
		return
	}
	response.OkWithData(res, c)
}

// GetSysErrorList 分页获取错误日志列表
// @Tags      SysError
// @Summary   分页获取错误日志列表
// @Produce   application/json
// @Param     pageNum   query  int     true   "页码"
// @Param     pageSize  query  int     true   "每页大小"
// @Param     form      query  string  false  "错误来源"
// @Param     info      query  string  false  "错误内容(模糊)"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]system.SysError},msg=string}
// @Router    /log/sysError/getSysErrorList [get]
func (sysErrorApi *SysErrorApi) GetSysErrorList(c *gin.Context) {
	ctx := c.Request.Context()
	var q systemReq.SysErrorSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := sysErrorService.GetSysErrorInfoList(ctx, q)
	if err != nil {
		logger.WithCtx(ctx).Mod("biz").Err(err).Error("获取错误日志列表失败")
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

// GetSysErrorSolution 触发错误日志的异步AI处理
// @Tags      SysError
// @Summary   根据ID触发AI处理:标记为处理中,异步生成解决方案后改为处理完成
// @Produce   application/json
// @Param     id  query  string  true  "错误日志ID"
// @Success   200  {object}  response.Response{msg=string}
// @Router    /log/sysError/getSysErrorSolution [get]
func (sysErrorApi *SysErrorApi) GetSysErrorSolution(c *gin.Context) {
	ctx := c.Request.Context()
	ID := c.Query("id")
	if ID == "" {
		response.FailWithMessage("缺少参数: id", c)
		return
	}
	if err := sysErrorService.GetSysErrorSolution(ctx, ID, utils.GetUserID(c)); err != nil {
		response.FailWithMessage("处理触发失败", c)
		return
	}
	response.OkWithMessage("已提交至AI处理", c)
}
