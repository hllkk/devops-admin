package gateway

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// ResourceApplicationService 资源申请审批(P2·AI 市场)。
// 消费模型发布"领用前需审批"档：用户对需审批模型提交申请,管理员批准后把 model_key
// 授权进申请人个人主 Key(复用 syncModelToMainKeys,与发布自动授权同管线)。
// 规避 AIHelms resource_applications 四坑：①主 Key 不存在静默 skip 仍标记 approved——
// 此处 scope 圈空集不报错,授权由自愈差集源(approvedApplicationModelKeys)在后建主 Key
// 时补上;②申请只查存在性不校验可见性——此处经 visibleModelScope 全量校验;③防重无
// DB 约束——此处复合唯一索引 + 条件更新双保险;④审批无通知——api 层接 SysNotice 定向通知。
type ResourceApplicationService struct{}

// Create 提交申请(用户侧)：校验模型对申请人可见且需审批 → 唯一索引防重
// (pending 拒/approved 拒/rejected 复用原行重置)。免审批模型不收申请(本来就自动授权)。
func (s *ResourceApplicationService) Create(ctx context.Context, req gatewayReq.ApplicationCreateParams, userId int64) (gatewayResp.ApplicationView, error) {
	if req.ResourceType != gateway.ApplicationResourceModel {
		return gatewayResp.ApplicationView{}, errors.New("暂仅支持模型申请(MCP/技能市场即将上线)")
	}
	db := global.OPS_DB.WithContext(ctx)
	// 模型校验:存在 + 启用 + 已发布 + 有路由名 + 对申请人可见(三档可见性过滤)
	q := visibleModelScope(
		db.Model(&gateway.Model{}).
			Where("is_active = ? AND is_published = ? AND model_key <> ''", true, true),
		userId, userDeptIdOf(db, userId),
	)
	var m gateway.Model
	if err := q.Where("model_id = ?", req.ResourceId).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return gatewayResp.ApplicationView{}, errors.New("模型不存在或对您不可见")
		}
		return gatewayResp.ApplicationView{}, err
	}
	if !m.RequiresApproval {
		return gatewayResp.ApplicationView{}, errors.New("该模型无需申请,可直接使用")
	}

	// 防重:复合唯一索引兜底并发,应用层按既有行状态分流
	var exist gateway.ResourceApplication
	err := db.Where("user_id = ? AND resource_type = ? AND resource_id = ?",
		userId, req.ResourceType, req.ResourceId).First(&exist).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		app := gateway.ResourceApplication{
			UserId:       userId,
			ResourceType: req.ResourceType,
			ResourceId:   req.ResourceId,
			Reason:       req.Reason,
			Status:       gateway.ApplicationStatusPending,
		}
		app.CreateBy = userId
		app.UpdateBy = userId
		if err := db.Create(&app).Error; err != nil {
			return gatewayResp.ApplicationView{}, err
		}
		return toApplicationView(ctx, app, m), nil
	case err != nil:
		return gatewayResp.ApplicationView{}, err
	case exist.Status == gateway.ApplicationStatusPending:
		return gatewayResp.ApplicationView{}, errors.New("已存在待审批的申请,请耐心等待")
	case exist.Status == gateway.ApplicationStatusApproved:
		return gatewayResp.ApplicationView{}, errors.New("您已拥有该模型,无需重复申请")
	default: // rejected → 复用原行重置为 pending(清审批字段;行永不软删,历史随重置覆盖)
		res := db.Model(&gateway.ResourceApplication{}).
			Where("application_id = ? AND status = ?", exist.ApplicationId, gateway.ApplicationStatusRejected).
			Updates(map[string]any{
				"status":       gateway.ApplicationStatusPending,
				"reason":       req.Reason,
				"reviewed_by":  0,
				"reviewed_at":  nil,
				"review_notes": "",
				"update_by":    userId,
			})
		if res.Error != nil {
			return gatewayResp.ApplicationView{}, res.Error
		}
		if res.RowsAffected == 0 { // 并发下已被他处处理,按当前状态重读分流
			return s.Create(ctx, req, userId)
		}
		exist.Status = gateway.ApplicationStatusPending
		exist.Reason = req.Reason
		return toApplicationView(ctx, exist, m), nil
	}
}

// GetMyList 我的申请(用户侧,强制本人;分页+状态/类型筛选)。
func (s *ResourceApplicationService) GetMyList(ctx context.Context, userId int64, q gatewayReq.ApplicationSearch) (list []gatewayResp.ApplicationView, total int64, err error) {
	return s.listApplication(ctx, q, userId)
}

// GetApplicationList 审批列表(管理端,可按申请人筛选)。
func (s *ResourceApplicationService) GetApplicationList(ctx context.Context, q gatewayReq.ApplicationSearch) (list []gatewayResp.ApplicationView, total int64, err error) {
	return s.listApplication(ctx, q, 0)
}

// listApplication 分页查询(userId>0 强制本人,0=管理端不限)。
func (s *ResourceApplicationService) listApplication(ctx context.Context, q gatewayReq.ApplicationSearch, userId int64) (list []gatewayResp.ApplicationView, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&gateway.ResourceApplication{})
	if userId > 0 {
		db = db.Where("user_id = ?", userId)
	} else if q.UserId > 0 {
		db = db.Where("user_id = ?", q.UserId)
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	if q.ResourceType != "" {
		db = db.Where("resource_type = ?", q.ResourceType)
	}
	var rows []gateway.ResourceApplication
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("application_id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	} else {
		err = db.Count(&total).Order("application_id DESC").Find(&rows).Error
	}
	if err != nil {
		return nil, 0, err
	}
	list, err = s.fillApplicationViews(ctx, rows)
	return list, total, err
}

// fillApplicationViews 批量回填视图(申请人/审批人昵称 + 模型名/路由名,每页三次 IN 查询防 N+1)。
func (s *ResourceApplicationService) fillApplicationViews(ctx context.Context, rows []gateway.ResourceApplication) ([]gatewayResp.ApplicationView, error) {
	db := global.OPS_DB.WithContext(ctx)
	userIds := make([]int64, 0, len(rows)*2)
	modelIds := make([]int64, 0, len(rows))
	for i := range rows {
		userIds = append(userIds, rows[i].UserId)
		if rows[i].ReviewedBy != 0 {
			userIds = append(userIds, rows[i].ReviewedBy)
		}
		if rows[i].ResourceType == gateway.ApplicationResourceModel {
			modelIds = append(modelIds, rows[i].ResourceId)
		}
	}
	userNames := map[int64]string{}
	if len(userIds) > 0 {
		var users []system.SysUser
		if err := db.Select("id", "nick_name").Where("id IN ?", userIds).Find(&users).Error; err != nil {
			return nil, err
		}
		for _, u := range users {
			userNames[u.UserId] = u.NickName
		}
	}
	models := map[int64]gateway.Model{}
	if len(modelIds) > 0 {
		var ms []gateway.Model
		if err := db.Where("model_id IN ?", modelIds).Find(&ms).Error; err != nil {
			return nil, err
		}
		for _, m := range ms {
			models[m.ModelId] = m
		}
	}

	list := make([]gatewayResp.ApplicationView, 0, len(rows))
	for i := range rows {
		r := rows[i]
		var m gateway.Model
		if r.ResourceType == gateway.ApplicationResourceModel {
			m = models[r.ResourceId]
		}
		list = append(list, gatewayResp.ApplicationView{
			ResourceApplication: r,
			UserName:            userNames[r.UserId],
			ResourceName:        m.Name,
			ResourceKey:         m.ModelKey,
			ReviewerName:        userNames[r.ReviewedBy],
		})
	}
	return list, nil
}

// toApplicationView 单条视图组装(Create 路径,模型已查出,免重复查询)。
func toApplicationView(ctx context.Context, app gateway.ResourceApplication, m gateway.Model) gatewayResp.ApplicationView {
	view := gatewayResp.ApplicationView{
		ResourceApplication: app,
		ResourceName:        m.Name,
		ResourceKey:         m.ModelKey,
	}
	var u system.SysUser
	if err := global.OPS_DB.WithContext(ctx).Select("nick_name").Where("id = ?", app.UserId).First(&u).Error; err == nil {
		view.UserName = u.NickName
	}
	return view
}

// Approve 审批通过：pending 条件更新(并发防双审)后,事务内把 model_key 授权进申请人
// 个人主 Key(scope 锁定该用户,只圈活跃主 Key;停用主 Key 与未建主 Key 由自愈差集源补上)。
// LiteLLM 推送失败记 warning 不回滚,由每日 ResyncAiKeys 兜底。
func (s *ResourceApplicationService) Approve(ctx context.Context, req gatewayReq.ApplicationReviewParams, reviewBy int64) (gatewayResp.ApplicationReviewResult, gatewayResp.ReviewNotification, error) {
	return s.review(ctx, req.ApplicationId, req.ReviewNotes, reviewBy, true)
}

// Reject 审批驳回：仅置状态与审批留痕,无授权动作(驳回时未授权过,无需回收)。
func (s *ResourceApplicationService) Reject(ctx context.Context, req gatewayReq.ApplicationReviewParams, reviewBy int64) (gatewayResp.ApplicationReviewResult, gatewayResp.ReviewNotification, error) {
	return s.review(ctx, req.ApplicationId, req.ReviewNotes, reviewBy, false)
}

// BatchReview 批量审批:逐条串行复用单条逻辑(每条独立事务),单条失败不中断。
// 返回成功条目的通知信息(api 层逐条发定向通知)。
func (s *ResourceApplicationService) BatchReview(ctx context.Context, req gatewayReq.ApplicationBatchReviewParams, reviewBy int64, approve bool) (gatewayResp.BatchReviewResult, []gatewayResp.ReviewNotification, error) {
	var result gatewayResp.BatchReviewResult
	var notices []gatewayResp.ReviewNotification
	for _, id := range req.ApplicationIds {
		_, notice, err := s.review(ctx, id, req.ReviewNotes, reviewBy, approve)
		if err != nil {
			result.Failed = append(result.Failed, gatewayResp.BatchReviewFailure{ApplicationId: id, Reason: err.Error()})
			continue
		}
		result.Success = append(result.Success, id)
		notices = append(notices, notice)
	}
	return result, notices, nil
}

// review 审批单条(通过/驳回共用)：pending 条件更新防并发双审;通过侧校验模型仍可授
// (下架/未发布让管理员驳回),事务内状态+授权原子提交。驳回侧模型仅查名(通知用,查不到不阻塞)。
func (s *ResourceApplicationService) review(ctx context.Context, applicationId int64, notes string, reviewBy int64, approve bool) (gatewayResp.ApplicationReviewResult, gatewayResp.ReviewNotification, error) {
	var app gateway.ResourceApplication
	err := global.OPS_DB.WithContext(ctx).Where("application_id = ?", applicationId).First(&app).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gatewayResp.ApplicationReviewResult{}, gatewayResp.ReviewNotification{}, errors.New("申请不存在")
	}
	if err != nil {
		return gatewayResp.ApplicationReviewResult{}, gatewayResp.ReviewNotification{}, err
	}
	if app.Status != gateway.ApplicationStatusPending {
		return gatewayResp.ApplicationReviewResult{}, gatewayResp.ReviewNotification{}, errors.New("该申请已处理")
	}

	newStatus := gateway.ApplicationStatusRejected
	if approve {
		newStatus = gateway.ApplicationStatusApproved
	}
	updates := map[string]any{
		"status":       newStatus,
		"reviewed_by":  reviewBy,
		"reviewed_at":  time.Now(),
		"review_notes": notes,
		"update_by":    reviewBy,
	}

	var modelKey string
	var m gateway.Model
	err = global.OPS_DB.WithContext(ctx).Where("model_id = ?", app.ResourceId).First(&m).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		if approve { // 模型已删除:批准也无法授权,让管理员驳回
			return gatewayResp.ApplicationReviewResult{}, gatewayResp.ReviewNotification{}, errors.New("关联模型不存在,请驳回该申请")
		}
		m.Name = ""
	case err != nil:
		return gatewayResp.ApplicationReviewResult{}, gatewayResp.ReviewNotification{}, err
	case approve && (!m.IsActive || !m.IsPublished || m.ModelKey == ""): // 模型仍须可授
		return gatewayResp.ApplicationReviewResult{}, gatewayResp.ReviewNotification{}, errors.New("关联模型已下架或未发布,请驳回该申请")
	}
	modelKey = m.ModelKey

	var warnings []string
	err = global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// pending 条件更新:并发双审时后到者 RowsAffected=0
		res := tx.Model(&gateway.ResourceApplication{}).
			Where("application_id = ? AND status = ?", applicationId, gateway.ApplicationStatusPending).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("该申请已处理")
		}
		if approve && modelKey != "" {
			warnings = syncModelToMainKeys(ctx, tx, modelKey, func(q *gorm.DB) *gorm.DB {
				return q.Where("key_type = ? AND owner_type = ? AND owner_id = ?",
					gateway.KeyTypePersonalMain, gateway.OwnerTypeUser, app.UserId)
			})
		}
		return nil
	})
	if err != nil {
		return gatewayResp.ApplicationReviewResult{}, gatewayResp.ReviewNotification{}, err
	}
	for _, w := range warnings {
		logger.WithCtx(ctx).Mod("gateway").Warn(fmt.Sprintf("申请 %d 审批通过但授权同步有警告: %s", applicationId, w))
	}
	return gatewayResp.ApplicationReviewResult{Warnings: warnings},
		gatewayResp.ReviewNotification{UserId: app.UserId, ResourceName: m.Name, Approved: approve, ReviewNotes: notes}, nil
}
