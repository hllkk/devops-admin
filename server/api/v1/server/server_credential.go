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

// CredentialApi 服务器模块采集凭据管理(对齐前端 /server/credential/* 资源)
type CredentialApi struct{}

// GetCredentialList
// @Tags      ServerCredential
// @Summary   分页获取凭据列表
// @Produce   application/json
// @Param     credentialName  query  string  false  "凭据名称(模糊)"
// @Param     credentialType  query  string  false  "凭据类型(精确:ssh/bmc/snmp/db)"
// @Param     isActive        query  bool    false  "是否启用(精确)"
// @Param     pageNum         query  int     true   "页码"
// @Param     pageSize        query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]response.CredentialView},msg=string}
// @Router    /server/credential/list [get]
func (a *CredentialApi) GetCredentialList(c *gin.Context) {
	var q serverReq.CredentialSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	request.NormalizeEmptyBoolQuery(c, &q)
	list, total, err := credentialService.GetCredentialList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Error("获取凭据列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: list, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}

// GetCredential
// @Tags      ServerCredential
// @Summary   获取凭据详情
// @Produce   application/json
// @Param     id  path  int  true  "凭据ID"
// @Success   200  {object}  response.Response{data=response.CredentialView,msg=string}
// @Router    /server/credential/{id} [get]
func (a *CredentialApi) GetCredential(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的凭据ID", c)
		return
	}
	view, err := credentialService.GetCredential(c.Request.Context(), id)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Error("获取凭据详情失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(view, "获取成功", c)
}

// GetCredentialOptions
// @Tags      ServerCredential
// @Summary   获取凭据下拉选项(仅启用中,type 过滤,资产表单用)
// @Produce   application/json
// @Param     type  query  string  false  "凭据类型过滤(ssh/bmc/snmp/db,空=全部)"
// @Success   200  {object}  response.Response{data=[]response.CredentialOptions,msg=string}
// @Router    /server/credential/options [get]
func (a *CredentialApi) GetCredentialOptions(c *gin.Context) {
	opts, err := credentialService.GetCredentialOptions(c.Request.Context(), c.Query("type"))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Error("获取凭据选项失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(opts, "获取成功", c)
}

// CreateCredential
// @Tags      ServerCredential
// @Summary   新增凭据
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  serverReq.CredentialOperateParams  true  "凭据信息"
// @Success   200   {object}  response.Response{data=response.CredentialView,msg=string}
// @Router    /server/credential [post]
func (a *CredentialApi) CreateCredential(c *gin.Context) {
	var req serverReq.CredentialOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := credentialService.CreateCredential(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Error("新增凭据失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "新增成功", c)
}

// UpdateCredential
// @Tags      ServerCredential
// @Summary   修改凭据
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  serverReq.CredentialOperateParams  true  "凭据信息(含 credentialId)"
// @Success   200   {object}  response.Response{data=response.CredentialView,msg=string}
// @Router    /server/credential [put]
func (a *CredentialApi) UpdateCredential(c *gin.Context) {
	var req serverReq.CredentialOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := credentialService.UpdateCredential(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Error("修改凭据失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "修改成功", c)
}

// BatchDeleteCredential
// @Tags      ServerCredential
// @Summary   批量删除凭据(被资产关联时拒删)
// @Produce   application/json
// @Param     ids  path  string  true  "凭据ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /server/credential/{ids} [delete]
func (a *CredentialApi) BatchDeleteCredential(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的凭据ID: "+s, c)
			return
		}
		ids = append(ids, id)
	}
	if err := credentialService.DeleteCredential(c.Request.Context(), ids); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("server").Err(err).Error("删除凭据失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}
