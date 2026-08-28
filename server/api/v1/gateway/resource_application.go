package gateway

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	"github.com/hllkk/devops-admin/server/model/system/request"
	systemService "github.com/hllkk/devops-admin/server/service"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// ResourceApplicationApi 资源申请审批(对齐前端 /gateway/application/* 资源)。
// apply/my 为用户侧(入 casbin 登录白名单,数据范围由 JWT 锁定);list/approve/reject/batch-*
// 为管理端(走菜单 ApiPrefix → casbin)。
type ResourceApplicationApi struct{}

// CreateApplication
// @Tags      GatewayApplication
// @Summary   提交资源申请(用户侧,暂仅模型;需审批+对申请人可见才可申请)
// @Produce   application/json
// @Param     data  body  gatewayReq.ApplicationCreateParams  true  "资源类型/资源ID/申请理由"
// @Success   200   {object}  response.Response{data=response.ApplicationView,msg=string}
// @Router    /gateway/application/apply [post]
func (a *ResourceApplicationApi) CreateApplication(c *gin.Context) {
	var req gatewayReq.ApplicationCreateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	view, err := applicationService.Create(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Warn("提交申请失败: " + err.Error())
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(view, "申请已提交", c)
}

// GetMyApplications
// @Tags      GatewayApplication
// @Summary   我的申请列表(用户侧,强制本人;分页+状态/类型筛选)
// @Produce   application/json
// @Param     status        query  string  false  "状态(精确)"
// @Param     resourceType  query  string  false  "资源类型(精确)"
// @Param     pageNum       query  int     true   "页码"
// @Param     pageSize      query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]response.ApplicationView},msg=string}
// @Router    /gateway/application/my [get]
func (a *ResourceApplicationApi) GetMyApplications(c *gin.Context) {
	var q gatewayReq.ApplicationSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := applicationService.GetMyList(c.Request.Context(), utils.GetUserID(c), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取我的申请失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: list, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}

// GetApplicationList
// @Tags      GatewayApplication
// @Summary   分页获取申请列表(管理端审批列表,按状态/类型/申请人筛选)
// @Produce   application/json
// @Param     status        query  string  false  "状态(精确,空=全部)"
// @Param     resourceType  query  string  false  "资源类型(精确,空=全部)"
// @Param     userId        query  int     false  "申请人(0=不限)"
// @Param     pageNum       query  int     true   "页码"
// @Param     pageSize      query  int     true   "每页大小"
// @Success   200  {object}  response.Response{data=response.PageResult{rows=[]response.ApplicationView},msg=string}
// @Router    /gateway/application/list [get]
func (a *ResourceApplicationApi) GetApplicationList(c *gin.Context) {
	var q gatewayReq.ApplicationSearch
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := applicationService.GetApplicationList(c.Request.Context(), q)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Err(err).Error("获取申请列表失败")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		Rows: list, Total: total, PageNum: q.PageNum, PageSize: q.PageSize,
	}, "获取成功", c)
}

// ApproveApplication
// @Tags      GatewayApplication
// @Summary   审批通过(授权模型到申请人个人主 Key;批量用 batch-approve)
// @Produce   application/json
// @Param     data  body  gatewayReq.ApplicationReviewParams  true  "申请ID/审批意见"
// @Success   200   {object}  response.Response{data=response.ApplicationReviewResult,msg=string}
// @Router    /gateway/application/approve [put]
func (a *ResourceApplicationApi) ApproveApplication(c *gin.Context) {
	var req gatewayReq.ApplicationReviewParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	result, notice, err := applicationService.Approve(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Warn("审批通过失败: " + err.Error())
		response.FailWithMessage(err.Error(), c)
		return
	}
	notifyApplicationReviewed(c, notice)
	msg := "已通过并授权到申请人主 Key"
	if len(result.Warnings) > 0 {
		msg = fmt.Sprintf("已通过(有 %d 条同步警告,将由每日密钥重同步兜底)", len(result.Warnings))
	}
	response.OkWithDetailed(result, msg, c)
}

// RejectApplication
// @Tags      GatewayApplication
// @Summary   审批驳回(仅留痕,无授权动作)
// @Produce   application/json
// @Param     data  body  gatewayReq.ApplicationReviewParams  true  "申请ID/审批意见"
// @Success   200   {object}  response.Response{data=response.ApplicationReviewResult,msg=string}
// @Router    /gateway/application/reject [put]
func (a *ResourceApplicationApi) RejectApplication(c *gin.Context) {
	var req gatewayReq.ApplicationReviewParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	_, notice, err := applicationService.Reject(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Warn("审批驳回失败: " + err.Error())
		response.FailWithMessage(err.Error(), c)
		return
	}
	notifyApplicationReviewed(c, notice)
	response.OkWithMessage("已驳回", c)
}

// BatchApproveApplications
// @Tags      GatewayApplication
// @Summary   批量审批通过(逐条独立事务,单条失败不中断;成功者逐条定向通知)
// @Produce   application/json
// @Param     data  body  gatewayReq.ApplicationBatchReviewParams  true  "申请ID列表/审批意见"
// @Success   200   {object}  response.Response{data=response.BatchReviewResult,msg=string}
// @Router    /gateway/application/batch-approve [put]
func (a *ResourceApplicationApi) BatchApproveApplications(c *gin.Context) {
	batchReview(c, true)
}

// BatchRejectApplications
// @Tags      GatewayApplication
// @Summary   批量审批驳回(逐条独立事务,单条失败不中断;成功者逐条定向通知)
// @Produce   application/json
// @Param     data  body  gatewayReq.ApplicationBatchReviewParams  true  "申请ID列表/审批意见"
// @Success   200   {object}  response.Response{data=response.BatchReviewResult,msg=string}
// @Router    /gateway/application/batch-reject [put]
func (a *ResourceApplicationApi) BatchRejectApplications(c *gin.Context) {
	batchReview(c, false)
}

// batchReview 批量审批共用 handler(通过/驳回仅差一个布尔,接口保持两个语义明确的 path)。
func batchReview(c *gin.Context, approve bool) {
	var req gatewayReq.ApplicationBatchReviewParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	result, notices, err := applicationService.BatchReview(c.Request.Context(), req, utils.GetUserID(c), approve)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Warn("批量审批失败: " + err.Error())
		response.FailWithMessage(err.Error(), c)
		return
	}
	for _, n := range notices {
		notifyApplicationReviewed(c, n)
	}
	response.OkWithDetailed(result, fmt.Sprintf("成功 %d 条,失败 %d 条", len(result.Success), len(result.Failed)), c)
}

// notifyApplicationReviewed 审批结果定向通知(复用 SysNotice+SSE)。放 api 层:
// service/system 已反向 import service/gateway(用户级联),service 层调用会成环。
// 通知失败只告警,不影响审批结果。
func notifyApplicationReviewed(c *gin.Context, n gatewayResp.ReviewNotification) {
	var title, content string
	if n.Approved {
		title = "AI 资源申请已通过"
		content = fmt.Sprintf("您申请的模型「%s」已通过审批,已授权到您的个人主 Key。", n.ResourceName)
	} else {
		title = "AI 资源申请未通过"
		content = fmt.Sprintf("您申请的模型「%s」未通过审批。", n.ResourceName)
	}
	if n.ReviewNotes != "" {
		content += "审批意见:" + n.ReviewNotes
	}
	err := systemService.ServiceGroupApp.SystemServiceGroup.NoticeService.CreateNotice(
		c.Request.Context(),
		request.NoticeOperateParams{
			NoticeTitle:   title,
			NoticeType:    "1", // 通知(定向投递)
			NoticeContent: content,
			Status:        "0",
			TargetType:    "users",
			TargetUserIds: []int64{n.UserId},
		},
		utils.GetUserID(c),
	)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("gateway").Warn(fmt.Sprintf("审批结果通知发送失败(user %d): %v", n.UserId, err))
	}
}
