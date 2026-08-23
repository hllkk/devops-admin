package gateway

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// ModelApi 模型管理(对齐前端 /gateway/model/* 资源)
type ModelApi struct{}

// GetModelList
// @Tags      GatewayModel
// @Summary   分页获取模型列表(含部署计数)
// @Produce   application/json
// @Param     name        query  string  false  "展示名(模糊)"
// @Param     modelKey    query  string  false  "路由名(模糊)"
// @Param     category    query  string  false  "类别(精确)"
// @Param     isActive    query  bool    false  "是否启用(精确)"
// @Param     isPublished query  bool    false  "是否发布(精确)"
// @Param     pageNum     query  int     true   "页码"
// @Param     pageSize    query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]response.ModelView},msg=string}
// @Router    /gateway/model/list [get]
func (a *ModelApi) GetModelList(c *gin.Context) {
	var q gatewayReq.ModelSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := modelService.GetModelList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取模型列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: list, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}

// GetActiveModels
// @Tags      GatewayModel
// @Summary   获取对外激活模型列表(active+published，含 anthropic 变体路由名)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]response.ActiveModelView,msg=string}
// @Router    /gateway/model/active [get]
func (a *ModelApi) GetActiveModels(c *gin.Context) {
	list, err := modelService.GetActiveModels(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取激活模型列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// GetModel
// @Tags      GatewayModel
// @Summary   获取模型详情(含部署列表)
// @Produce   application/json
// @Param     id  path  int  true  "模型ID"
// @Success   200  {object}  response.Response{data=response.ModelDetailView,msg=string}
// @Router    /gateway/model/{id} [get]
func (a *ModelApi) GetModel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的模型ID", c)
		return
	}
	detail, err := modelService.GetModel(c.Request.Context(), id)
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(detail, "获取成功", c)
}

// CreateModel
// @Tags      GatewayModel
// @Summary   新增模型
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.ModelOperateParams  true  "模型信息"
// @Success   200   {object}  response.Response{data=response.ModelView,msg=string}
// @Router    /gateway/model [post]
func (a *ModelApi) CreateModel(c *gin.Context) {
	var req gatewayReq.ModelOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := modelService.CreateModel(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "新增成功", c)
}

// UpdateModel
// @Tags      GatewayModel
// @Summary   修改模型(允许改路由名/类别，触发关联部署路由级联重建)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.ModelOperateParams  true  "模型信息(含 modelId)"
// @Success   200   {object}  response.Response{data=response.ModelView,msg=string}
// @Router    /gateway/model [put]
func (a *ModelApi) UpdateModel(c *gin.Context) {
	var req gatewayReq.ModelOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := modelService.UpdateModel(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "修改成功", c)
}

// BatchDeleteModels
// @Tags      GatewayModel
// @Summary   批量删除模型(软删三连：部署先禁用→部署停用→模型软删+清可见性)
// @Produce   application/json
// @Param     ids  path  string  true  "模型ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /gateway/model/{ids} [delete]
func (a *ModelApi) BatchDeleteModels(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的模型ID: "+s, c)
			return
		}
		ids = append(ids, id)
	}
	if err := modelService.DeleteModels(c.Request.Context(), ids); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}

// GetModelPublish
// @Tags      GatewayModel
// @Summary   获取模型发布设置(含指定可见部门)
// @Produce   application/json
// @Param     id  path  int  true  "模型ID"
// @Success   200  {object}  response.Response{data=response.ModelPublishView,msg=string}
// @Router    /gateway/model/publish/{id} [get]
func (a *ModelApi) GetModelPublish(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的模型ID", c)
		return
	}
	view, err := modelService.GetModelPublish(c.Request.Context(), id)
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(view, "获取成功", c)
}

// PublishModel
// @Tags      GatewayModel
// @Summary   更新模型发布设置(可见范围/审批；selected 模式重建部门可见行)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.ModelPublishParams  true  "发布设置"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /gateway/model/publish [put]
func (a *ModelApi) PublishModel(c *gin.Context) {
	var req gatewayReq.ModelPublishParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := modelService.PublishModel(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "发布设置已更新", c)
}
