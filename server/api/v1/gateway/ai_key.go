package gateway

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
	"github.com/hllkk/devops-admin/server/utils/request"
)

// AiKeyApi AI 密钥管理(对齐前端 /gateway/ai-key/* 与 identity/* 资源)
type AiKeyApi struct{}

// GetMyIdentity
// @Tags      GatewayAiKey
// @Summary   获取我的 AI 身份(管理员创建制,未开通 opened=false + 主 Key 明文 + 场景 Key 列表 + 可用模型)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=response.MyIdentityView,msg=string}
// @Router    /gateway/ai-key/identity/my [get]
func (a *AiKeyApi) GetMyIdentity(c *gin.Context) {
	view, err := aiKeyService.GetMyIdentity(c.Request.Context(), utils.GetUserID(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取 AI 身份失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(view, "获取成功", c)
}

// GetAvailableModels
// @Tags      GatewayAiKey
// @Summary   获取可授权模型列表(发布+激活，含 anthropic 变体标注)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]response.AvailableModelView,msg=string}
// @Router    /gateway/ai-key/identity/available-models [get]
func (a *AiKeyApi) GetAvailableModels(c *gin.Context) {
	list, err := aiKeyService.GetAvailableModels(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取可用模型失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// GetAiKeyList
// @Tags      GatewayAiKey
// @Summary   分页获取密钥列表(管理员视角，不返回 KeyValue)
// @Produce   application/json
// @Param     keyType    query  string  false  "密钥类型(精确)"
// @Param     ownerType  query  string  false  "归属类型(精确)"
// @Param     ownerId   query  int     false  "归属ID(0=不限)"
// @Param     name      query  string  false  "名称(模糊)"
// @Param     isActive  query  bool    false  "是否启用(精确)"
// @Param     pageNum   query  int     true   "页码"
// @Param     pageSize  query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]response.AiKeyView},msg=string}
// @Router    /gateway/ai-key/list [get]
func (a *AiKeyApi) GetAiKeyList(c *gin.Context) {
	var q gatewayReq.AiKeySearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	request.NormalizeEmptyBoolQuery(c, &q)
	list, total, err := aiKeyService.GetAiKeyList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取密钥列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: list, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}

// GetAiKey
// @Tags      GatewayAiKey
// @Summary   获取密钥详情(管理员视角，不返回 KeyValue)
// @Produce   application/json
// @Param     id  path  int  true  "密钥ID"
// @Success   200  {object}  response.Response{data=response.AiKeyView,msg=string}
// @Router    /gateway/ai-key/{id} [get]
func (a *AiKeyApi) GetAiKey(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的密钥ID", c)
		return
	}
	view, err := aiKeyService.GetAiKey(c.Request.Context(), id)
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(view, "获取成功", c)
}

// RevealAiKey
// @Tags      GatewayAiKey
// @Summary   查看密钥完整明文(仅管理员/超管；解密 key_value，操作日志自动审计)
// @Produce   application/json
// @Param     id  path  int  true  "密钥ID"
// @Success   200  {object}  response.Response{data=response.AiKeyRevealView,msg=string}
// @Router    /gateway/ai-key/value/{id} [get]
func (a *AiKeyApi) RevealAiKey(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的密钥ID", c)
		return
	}
	view, err := aiKeyService.RevealAiKeyValue(c.Request.Context(), id, utils.GetUserInfo(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("查看密钥明文失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "获取成功", c)
}

// CreateSceneKey
// @Tags      GatewayAiKey
// @Summary   创建密钥(场景 Key 或管理员手动建部门主 Key)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.AiKeyOperateParams  true  "密钥信息"
// @Success   200   {object}  response.Response{data=response.AiKeyView,msg=string}
// @Router    /gateway/ai-key [post]
func (a *AiKeyApi) CreateSceneKey(c *gin.Context) {
	var req gatewayReq.AiKeyOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := aiKeyService.CreateSceneKey(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "新增成功", c)
}

// UpdateAiKey
// @Tags      GatewayAiKey
// @Summary   修改密钥(授权/预算/限流/启停；类型不可改。name/预算周期/限流模式空串与 tpm/rpm/预算额度 nil=不改，expires_at nil=改回永不过期)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.AiKeyOperateParams  true  "密钥信息(含 aiKeyId)"
// @Success   200   {object}  response.Response{data=response.AiKeyView,msg=string}
// @Router    /gateway/ai-key [put]
func (a *AiKeyApi) UpdateAiKey(c *gin.Context) {
	var req gatewayReq.AiKeyOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := aiKeyService.UpdateAiKey(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "修改成功", c)
}

// BatchDeleteAiKeys
// @Tags      GatewayAiKey
// @Summary   批量删除密钥(先删 LiteLLM，失败则本地不动)
// @Produce   application/json
// @Param     ids  path  string  true  "密钥ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /gateway/ai-key/{ids} [delete]
func (a *AiKeyApi) BatchDeleteAiKeys(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的密钥ID: "+s, c)
			return
		}
		ids = append(ids, id)
	}
	if err := aiKeyService.DeleteAiKey(c.Request.Context(), ids); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}

// RotateAiKey
// @Tags      GatewayAiKey
// @Summary   轮换密钥(原地换 Key 值保归因；旧 Key 立即失效，新明文仅 owner 经 identity/my 可查)
// @Produce   application/json
// @Param     id  path  int  true  "密钥ID"
// @Success   200  {object}  response.Response{data=response.AiKeyView,msg=string}
// @Router    /gateway/ai-key/rotate/{id} [post]
func (a *AiKeyApi) RotateAiKey(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithMessage("无效的密钥ID", c)
		return
	}
	view, err := aiKeyService.RotateAiKey(c.Request.Context(), id, utils.GetUserID(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("轮换密钥失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "轮换成功，新 Key 请用户在首页查看", c)
}

// BatchCreateMainKeys
// @Tags      GatewayAiKey
// @Summary   批量开通个人主 Key(按部门/按用户；已有跳过，部分失败不中断)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.AiKeyBatchCreateParams  true  "目标(deptId 优先,userIds 补充)"
// @Success   200   {object}  response.Response{data=response.BatchCreateMainKeysResult,msg=string}
// @Router    /gateway/ai-key/batch [post]
func (a *AiKeyApi) BatchCreateMainKeys(c *gin.Context) {
	var req gatewayReq.AiKeyBatchCreateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	result, err := aiKeyService.BatchCreateMainKeys(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("批量开通主 Key 失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	// 部分失败走成功响应+data 标记(前端按 failed 渲染)，避免 axios 自动弹错误造成双提示
	response.OkWithDetailed(result, fmt.Sprintf("开通 %d/%d(跳过 %d，失败 %d)",
		result.Created, result.Total, result.Skipped, len(result.Failed)), c)
}

// ResyncAiKeys
// @Tags      GatewayAiKey
// @Summary   手动重推全部密钥投影到 LiteLLM(改名级联/授权对齐同步失败的漂移兜底)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=response.ResyncResult,msg=string}
// @Router    /gateway/ai-key/resync [post]
func (a *AiKeyApi) ResyncAiKeys(c *gin.Context) {
	result, err := aiKeyService.ResyncAllKeys(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("密钥重同步失败")
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "重同步完成", c)
}
