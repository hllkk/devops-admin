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

// KeyScenarioApi 使用场景字典(挂在 /gateway/ai-key/scenario/* 子资源，随密钥管理菜单 api_prefix 走 casbin)。
type KeyScenarioApi struct{}

// GetKeyScenarioList
// @Tags      GatewayAiKey
// @Summary   分页获取使用场景列表
// @Produce   application/json
// @Param     name      query  string  false  "名称(模糊)"
// @Param     isActive  query  bool    false  "是否启用(精确)"
// @Param     pageNum   query  int     true   "页码"
// @Param     pageSize  query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]gateway.KeyScenario},msg=string}
// @Router    /gateway/ai-key/scenario/list [get]
func (a *KeyScenarioApi) GetKeyScenarioList(c *gin.Context) {
	var q gatewayReq.KeyScenarioSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	request.NormalizeEmptyBoolQuery(c, &q)
	list, total, err := keyScenarioService.GetKeyScenarioList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取场景列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: list, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}

// GetAllScenarios
// @Tags      GatewayAiKey
// @Summary   获取启用中的场景全量(建 Key 表单下拉)
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]gateway.KeyScenario,msg=string}
// @Router    /gateway/ai-key/scenario/all [get]
func (a *KeyScenarioApi) GetAllScenarios(c *gin.Context) {
	list, err := keyScenarioService.GetAllScenarios(c.Request.Context())
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取场景全量失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// CreateKeyScenario
// @Tags      GatewayAiKey
// @Summary   新增使用场景(name 未软删行内唯一)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.KeyScenarioOperateParams  true  "场景信息"
// @Success   200   {object}  response.Response{data=gateway.KeyScenario,msg=string}
// @Router    /gateway/ai-key/scenario [post]
func (a *KeyScenarioApi) CreateKeyScenario(c *gin.Context) {
	var req gatewayReq.KeyScenarioOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	sc, err := keyScenarioService.CreateKeyScenario(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(sc, "新增成功", c)
}

// UpdateKeyScenario
// @Tags      GatewayAiKey
// @Summary   修改使用场景(改名查重；停用后新建 Key 不可选)
// @Accept    application/json
// @Produce   application/json
// @Param     data  body  gatewayReq.KeyScenarioOperateParams  true  "场景信息(含 scenarioId)"
// @Success   200   {object}  response.Response{data=gateway.KeyScenario,msg=string}
// @Router    /gateway/ai-key/scenario [put]
func (a *KeyScenarioApi) UpdateKeyScenario(c *gin.Context) {
	var req gatewayReq.KeyScenarioOperateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	sc, err := keyScenarioService.UpdateKeyScenario(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(sc, "修改成功", c)
}

// BatchDeleteKeyScenarios
// @Tags      GatewayAiKey
// @Summary   批量删除使用场景(被密钥引用时拒删)
// @Produce   application/json
// @Param     ids  path  string  true  "场景ID列表(逗号分隔)"
// @Success   200  {object}  response.Response{data=bool,msg=string}
// @Router    /gateway/ai-key/scenario/{ids} [delete]
func (a *KeyScenarioApi) BatchDeleteKeyScenarios(c *gin.Context) {
	ids := make([]int64, 0, 4)
	for s := range strings.SplitSeq(c.Param("ids"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			response.FailWithMessage("无效的场景ID: "+s, c)
			return
		}
		ids = append(ids, id)
	}
	if err := keyScenarioService.DeleteKeyScenario(c.Request.Context(), ids); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(true, "删除成功", c)
}
