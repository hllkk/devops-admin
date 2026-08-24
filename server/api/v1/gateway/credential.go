package gateway

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
	"github.com/hllkk/devops-admin/server/utils/request"
)

// CredentialApi 凭证管理(对齐前端 /gateway/credential/* 资源)
type CredentialApi struct{}

// GetCredentialList
// @Tags      GatewayCredential
// @Summary   分页获取凭证列表
// @Produce   application/json
// @Param     credentialName  query  string  false  "凭证名称(模糊)"
// @Param     providerId      query  int     false  "关联供应商ID(0=不限)"
// @Param     isActive        query  bool    false  "是否启用(精确)"
// @Param     litellmSynced   query  bool    false  "是否已同步LiteLLM(精确)"
// @Param     pageNum         query  int     true   "页码"
// @Param     pageSize        query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]response.CredentialView},msg=string}
// @Router    /gateway/credential/list [get]
func (a *CredentialApi) GetCredentialList(c *gin.Context) {
	var q gatewayReq.CredentialSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	request.NormalizeEmptyBoolQuery(c, &q)
	list, total, err := credentialService.GetCredentialList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取凭证列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: list, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}

// GetCredential
// @Tags      GatewayCredential
// @Summary   获取凭证详情
// @Produce   application/json
// @Param     id  path  int  true  "凭证ID"
// @Success   200  {object}  response.Response{data=response.CredentialView,msg=string}
// @Router    /gateway/credential/{id} [get]
func (a *CredentialApi) GetCredential(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的凭证ID", c)
		return
	}
	view, err := credentialService.GetCredential(c.Request.Context(), id)
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(view, "获取成功", c)
}

// CreateCredential
// @Tags      GatewayCredential
// @Summary   新增凭证(事务内同步 LiteLLM，推送失败整体回滚)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.CredentialOperateParams  true  "凭证信息"
// @Success   200   {object}  response.Response{data=response.CredentialView,msg=string}
// @Router    /gateway/credential [post]
func (a *CredentialApi) CreateCredential(c *gin.Context) {
	var req gatewayReq.CredentialOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := credentialService.CreateCredential(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "新增成功", c)
}

// UpdateCredential
// @Tags      GatewayCredential
// @Summary   修改凭证(不允许改凭证名；键值合并语义：敏感值掩码回传=未修改；仅投影变化才重推 LiteLLM)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.CredentialOperateParams  true  "凭证信息(含 credentialId)"
// @Success   200   {object}  response.Response{data=response.CredentialView,msg=string}
// @Router    /gateway/credential [put]
func (a *CredentialApi) UpdateCredential(c *gin.Context) {
	var req gatewayReq.CredentialOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := credentialService.UpdateCredential(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "修改成功", c)
}

// BatchDeleteCredential
// @Tags      GatewayCredential
// @Summary   批量删除凭证(先删 LiteLLM 投影，失败则本地不动)
// @Produce   application/json
// @Param     ids  path  string  true  "凭证ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /gateway/credential/{ids} [delete]
func (a *CredentialApi) BatchDeleteCredential(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的凭证ID: "+s, c)
			return
		}
		ids = append(ids, id)
	}
	if err := credentialService.DeleteCredential(c.Request.Context(), ids); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}

// GetProviderFields
// @Tags      GatewayCredential
// @Summary   获取各供应商凭证表单字段定义(透传 LiteLLM /public/providers/fields，供前端动态渲染)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]object,msg=string}
// @Router    /gateway/credential/provider-fields [get]
func (a *CredentialApi) GetProviderFields(c *gin.Context) {
	fields, err := credentialService.GetProviderFields(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取供应商凭证字段失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(fields, "获取成功", c)
}

// ResyncCredentials
// @Tags      GatewayCredential
// @Summary   手动重同步全部凭证到 LiteLLM(投影比对，漂移兜底)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=response.ResyncResult,msg=string}
// @Router    /gateway/credential/resync [post]
func (a *CredentialApi) ResyncCredentials(c *gin.Context) {
	result, err := credentialService.ResyncCredentials(c.Request.Context())
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "重同步完成", c)
}
