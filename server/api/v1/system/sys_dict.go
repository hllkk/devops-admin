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

// DictApi 字典管理(对齐前端 /system/dict/type/* 与 /system/dict/data/* 资源)
type DictApi struct{}

// GetDictTypeList
// @Tags      SysDictType
// @Summary   分页获取字典类型列表
// @Produce   application/json
// @Param     dictName  query  string  false  "字典名称(模糊匹配)"
// @Param     dictType  query  string  false  "字典类型(模糊匹配)"
// @Param     pageNum   query  int     true   "页码"
// @Param     pageSize  query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]system.SysDictType},msg=string}
// @Router    /system/dict/type/list [get]
func (d *DictApi) GetDictTypeList(c *gin.Context) {
	var q systemReq.DictTypeSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := dictTypeService.GetDictTypeList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取字典类型列表失败")
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

// CreateDictType
// @Tags      SysDictType
// @Summary   新增字典类型
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.DictTypeOperateParams  true  "字典类型信息"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/dict/type [post]
func (d *DictApi) CreateDictType(c *gin.Context) {
	var req systemReq.DictTypeOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := dictTypeService.CreateDictType(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "新增成功", c)
}

// UpdateDictType
// @Tags      SysDictType
// @Summary   修改字典类型
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.DictTypeOperateParams  true  "字典类型信息(含 dictId)"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/dict/type [put]
func (d *DictApi) UpdateDictType(c *gin.Context) {
	var req systemReq.DictTypeOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := dictTypeService.UpdateDictType(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "修改成功", c)
}

// BatchDeleteDictType
// @Tags      SysDictType
// @Summary   批量删除字典类型
// @Produce   application/json
// @Param     ids  path  string  true  "字典类型ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /system/dict/type/{ids} [delete]
func (d *DictApi) BatchDeleteDictType(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的字典类型ID: "+s, c)
			return
		}
		ids = append(ids, n)
	}
	if err := dictTypeService.DeleteDictType(c.Request.Context(), ids); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}

// GetDictTypeOption
// @Tags      SysDictType
// @Summary   获取字典类型选择框列表
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]system.SysDictType,msg=string}
// @Router    /system/dict/type/optionselect [get]
func (d *DictApi) GetDictTypeOption(c *gin.Context) {
	list, err := dictTypeService.GetDictTypeOptionList(c.Request.Context())
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// GetDictDataList
// @Tags      SysDictData
// @Summary   分页获取字典数据列表
// @Produce   application/json
// @Param     dictLabel  query  string  false  "字典标签(模糊匹配)"
// @Param     dictType   query  string  false  "字典类型(精确匹配)"
// @Param     pageNum    query  int     true   "页码"
// @Param     pageSize   query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]system.SysDictData},msg=string}
// @Router    /system/dict/data/list [get]
func (d *DictApi) GetDictDataList(c *gin.Context) {
	var q systemReq.DictDataSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := dictDataService.GetDictDataList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取字典数据列表失败")
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

// GetDictDataByType
// @Tags      SysDictData
// @Summary   按字典类型查字典数据
// @Produce   application/json
// @Param     dictType  path  string  true  "字典类型"
// @Success   200  {object}  response.Response{data=[]system.SysDictData,msg=string}
// @Router    /system/dict/data/type/{dictType} [get]
func (d *DictApi) GetDictDataByType(c *gin.Context) {
	list, err := dictDataService.GetDictDataByType(c.Request.Context(), c.Param("dictType"))
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// CreateDictData
// @Tags      SysDictData
// @Summary   新增字典数据
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.DictDataOperateParams  true  "字典数据信息"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/dict/data [post]
func (d *DictApi) CreateDictData(c *gin.Context) {
	var req systemReq.DictDataOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := dictDataService.CreateDictData(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "新增成功", c)
}

// UpdateDictData
// @Tags      SysDictData
// @Summary   修改字典数据
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  systemReq.DictDataOperateParams  true  "字典数据信息(含 dictCode)"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /system/dict/data [put]
func (d *DictApi) UpdateDictData(c *gin.Context) {
	var req systemReq.DictDataOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := dictDataService.UpdateDictData(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "修改成功", c)
}

// BatchDeleteDictData
// @Tags      SysDictData
// @Summary   批量删除字典数据
// @Produce   application/json
// @Param     dictCodes  path  string  true  "字典编码列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /system/dict/data/{dictCodes} [delete]
func (d *DictApi) BatchDeleteDictData(c *gin.Context) {
	codes := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("dictCodes"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的字典编码: "+s, c)
			return
		}
		codes = append(codes, n)
	}
	if err := dictDataService.DeleteDictData(c.Request.Context(), codes); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}
