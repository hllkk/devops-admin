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

// ProviderApi AI 供应商管理(对齐前端 /gateway/provider/* 资源)
type ProviderApi struct{}

// GetProviderList
// @Tags      GatewayProvider
// @Summary   分页获取供应商列表
// @Produce   application/json
// @Param     name          query  string  false  "供应商名称(模糊)"
// @Param     providerType  query  string  false  "供应商类型(精确)"
// @Param     billingType   query  string  false  "计费类型(精确)"
// @Param     isActive       query  bool    false  "是否启用(精确)"
// @Param     pageNum       query  int     true   "页码"
// @Param     pageSize      query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]gateway.Provider},msg=string}
// @Router    /gateway/provider/list [get]
func (a *ProviderApi) GetProviderList(c *gin.Context) {
	var q gatewayReq.ProviderSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := providerService.GetProviderList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取供应商列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: list, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}

// GetProvider
// @Tags      GatewayProvider
// @Summary   获取供应商详情
// @Produce   application/json
// @Param     id  path  int  true  "供应商ID"
// @Success   200  {object}  response.Response{data=gateway.Provider,msg=string}
// @Router    /gateway/provider/{id} [get]
func (a *ProviderApi) GetProvider(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的供应商ID", c)
		return
	}
	p, err := providerService.GetProvider(c.Request.Context(), id)
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(p, "获取成功", c)
}

// CreateProvider
// @Tags      GatewayProvider
// @Summary   新增供应商
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.ProviderOperateParams  true  "供应商信息"
// @Success   200   {object}  response.Response{data=gateway.Provider,msg=string}
// @Router    /gateway/provider [post]
func (a *ProviderApi) CreateProvider(c *gin.Context) {
	var req gatewayReq.ProviderOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	p, err := providerService.CreateProvider(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(p, "新增成功", c)
}

// UpdateProvider
// @Tags      GatewayProvider
// @Summary   修改供应商
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.ProviderOperateParams  true  "供应商信息(含 providerId)"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /gateway/provider [put]
func (a *ProviderApi) UpdateProvider(c *gin.Context) {
	var req gatewayReq.ProviderOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := providerService.UpdateProvider(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "修改成功", c)
}

// BatchDeleteProvider
// @Tags      GatewayProvider
// @Summary   批量删除供应商
// @Produce   application/json
// @Param     ids  path  string  true  "供应商ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /gateway/provider/{ids} [delete]
func (a *ProviderApi) BatchDeleteProvider(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的供应商ID: "+s, c)
			return
		}
		ids = append(ids, id)
	}
	if err := providerService.DeleteProvider(c.Request.Context(), ids); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}
