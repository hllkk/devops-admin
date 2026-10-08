package server

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	serverReq "github.com/hllkk/devops-admin/server/model/server/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
	"github.com/hllkk/devops-admin/server/utils/request"
)

// AssetApi 服务器模块统一资产管理(对齐前端 /server/asset/* 资源)
type AssetApi struct{}

// GetAssetList
// @Tags      ServerAsset
// @Summary   分页获取资产列表
// @Produce   application/json
// @Param     assetName     query  string  false  "资产名称(模糊)"
// @Param     manageIp      query  string  false  "管理IP(模糊)"
// @Param     assetType     query  string  false  "资产类型(精确:physical/vm/docker_host/db_instance/net_device)"
// @Param     env           query  string  false  "环境标签(精确)"
// @Param     monitorStatus query  string  false  "监控在线状态(精确:online/offline/unknown)"
// @Param     isActive      query  bool    false  "是否启用(精确)"
// @Param     pageNum       query  int     true   "页码"
// @Param     pageSize      query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]server.Asset},msg=string}
// @Router    /server/asset/list [get]
func (a *AssetApi) GetAssetList(c *gin.Context) {
	var q serverReq.AssetSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	request.NormalizeEmptyBoolQuery(c, &q)
	list, total, err := assetService.GetAssetList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Error("获取资产列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: list, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}

// GetAsset
// @Tags      ServerAsset
// @Summary   获取资产详情
// @Produce   application/json
// @Param     id  path  int  true  "资产ID"
// @Success   200  {object}  response.Response{data=server.Asset,msg=string}
// @Router    /server/asset/{id} [get]
func (a *AssetApi) GetAsset(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的资产ID", c)
		return
	}
	asset, err := assetService.GetAsset(c.Request.Context(), id)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Error("获取资产详情失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(asset, "获取成功", c)
}

// CreateAsset
// @Tags      ServerAsset
// @Summary   新增资产
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  serverReq.AssetOperateParams  true  "资产信息"
// @Success   200   {object}  response.Response{data=server.Asset,msg=string}
// @Router    /server/asset [post]
func (a *AssetApi) CreateAsset(c *gin.Context) {
	var req serverReq.AssetOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	asset, err := assetService.CreateAsset(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Error("新增资产失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(asset, "新增成功", c)
}

// UpdateAsset
// @Tags      ServerAsset
// @Summary   修改资产
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  serverReq.AssetOperateParams  true  "资产信息(含 assetId)"
// @Success   200   {object}  response.Response{data=bool,msg=string}
// @Router    /server/asset [put]
func (a *AssetApi) UpdateAsset(c *gin.Context) {
	var req serverReq.AssetOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := assetService.UpdateAsset(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Error("修改资产失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "修改成功", c)
}

// BatchDeleteAsset
// @Tags      ServerAsset
// @Summary   批量删除资产
// @Produce   application/json
// @Param     ids  path  string  true  "资产ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /server/asset/{ids} [delete]
func (a *AssetApi) BatchDeleteAsset(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的资产ID: "+s, c)
			return
		}
		ids = append(ids, id)
	}
	if err := assetService.DeleteAsset(c.Request.Context(), ids); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Error("删除资产失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}

// GetAssetOverview
// @Tags      ServerAsset
// @Summary   服务器概览统计(各类型数量/监控状态/Agent状态/环境分布)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=response.AssetOverview,msg=string}
// @Router    /server/asset/overview [get]
func (a *AssetApi) GetAssetOverview(c *gin.Context) {
	overview, err := assetService.GetAssetOverview(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Error("获取服务器概览统计失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(overview, "获取成功", c)
}
