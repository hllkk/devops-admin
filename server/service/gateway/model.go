package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils/litellm"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// ModelService 模型管理(对齐前端 /gateway/model/* 资源)。
// 模型是管理元数据+发布控制，不同步 LiteLLM(部署才同步)；同 model_key 多部署构成
// LiteLLM 同名路由组 LB 池。发布字段只影响用户端可见性，不影响部署路由。
type ModelService struct{}

// GetModelList 分页查模型列表(带部署计数)。
func (s *ModelService) GetModelList(ctx context.Context, q gatewayReq.ModelSearch) (list []gatewayResp.ModelView, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&gateway.Model{})
	if q.Name != "" {
		db = db.Where("name LIKE ?", "%"+q.Name+"%")
	}
	if q.ModelKey != "" {
		db = db.Where("model_key LIKE ?", "%"+q.ModelKey+"%")
	}
	if q.Category != "" {
		db = db.Where("category = ?", q.Category)
	}
	if q.IsActive != nil {
		db = db.Where("is_active = ?", *q.IsActive)
	}
	if q.IsPublished != nil {
		db = db.Where("is_published = ?", *q.IsPublished)
	}
	var rows []gateway.Model
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("model_id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	} else {
		err = db.Count(&total).Order("model_id DESC").Find(&rows).Error
	}
	if err != nil {
		return nil, 0, err
	}

	modelIds := make([]int64, 0, len(rows))
	for i := range rows {
		modelIds = append(modelIds, rows[i].ModelId)
	}
	type depCount struct {
		ModelId int64 `gorm:"column:model_id"`
		Total   int64 `gorm:"column:total"`
		Active  int64 `gorm:"column:active"`
	}
	var counts []depCount
	totalMap := map[int64][2]int64{}
	if len(modelIds) > 0 {
		_ = global.OPS_DB.WithContext(ctx).Model(&gateway.ModelDeployment{}).
			Select("model_id, COUNT(*) AS total, SUM(CASE WHEN is_active THEN 1 ELSE 0 END) AS active").
			Where("model_id IN ?", modelIds).Group("model_id").Find(&counts).Error
		for _, c := range counts {
			totalMap[c.ModelId] = [2]int64{c.Total, c.Active}
		}
	}

	list = make([]gatewayResp.ModelView, 0, len(rows))
	for i := range rows {
		view := gatewayResp.ModelView{Model: rows[i], Capabilities: capabilitiesToList(rows[i].Capabilities)}
		if c, ok := totalMap[rows[i].ModelId]; ok {
			view.DeploymentCount, view.ActiveDeploymentCount = c[0], c[1]
		}
		list = append(list, view)
	}
	return list, total, nil
}

// GetActiveModels 对外激活模型列表(active+published)：附 anthropic 变体路由名，
// 供 home AI 身份/后续 AiKey 授权选择用。
func (s *ModelService) GetActiveModels(ctx context.Context) ([]gatewayResp.ActiveModelView, error) {
	var models []gateway.Model
	if err := global.OPS_DB.WithContext(ctx).
		Where("is_active = ? AND is_published = ?", true, true).
		Order("model_id DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	modelIds := make([]int64, 0, len(models))
	for i := range models {
		modelIds = append(modelIds, models[i].ModelId)
	}
	// 活跃部署 → 模型的 anthropic 标注（存在 active 且绑定 active anthropic 凭证的部署）
	anthropicModels := map[int64]bool{}
	if len(modelIds) > 0 {
		var deps []gateway.ModelDeployment
		_ = global.OPS_DB.WithContext(ctx).
			Where("is_active = ? AND credential_id <> 0 AND model_id IN ?", true, modelIds).
			Find(&deps).Error
		credIds := make([]int64, 0, len(deps))
		for i := range deps {
			credIds = append(credIds, deps[i].CredentialId)
		}
		creds := map[int64]*gateway.Credential{}
		if len(credIds) > 0 {
			var rows []gateway.Credential
			_ = global.OPS_DB.WithContext(ctx).Where("credential_id IN ?", credIds).Find(&rows).Error
			for i := range rows {
				creds[rows[i].CredentialId] = &rows[i]
			}
		}
		for i := range deps {
			if cred, ok := creds[deps[i].CredentialId]; ok && cred.IsActive && formatOf(cred) == "anthropic" {
				anthropicModels[deps[i].ModelId] = true
			}
		}
	}
	list := make([]gatewayResp.ActiveModelView, 0, len(models))
	for i := range models {
		m := models[i]
		view := gatewayResp.ActiveModelView{
			ModelId:          m.ModelId,
			ModelKey:         m.ModelKey,
			Name:             m.Name,
			Category:         m.Category,
			Description:      m.Description,
			LogoProviderType: m.LogoProviderType,
			Capabilities:     capabilitiesToList(m.Capabilities),
			RequiresApproval: m.RequiresApproval,
		}
		if anthropicModels[m.ModelId] {
			view.HasAnthropicDeployment = true
			view.ModelKeyAnthropic = m.ModelKey + gateway.ModelAnthropicSuffix
		}
		list = append(list, view)
	}
	return list, nil
}

// GetModel 模型详情(含部署列表，带路由名/掩码参数)。
func (s *ModelService) GetModel(ctx context.Context, id int64) (gatewayResp.ModelDetailView, error) {
	var m gateway.Model
	if err := global.OPS_DB.WithContext(ctx).Where("model_id = ?", id).First(&m).Error; err != nil {
		return gatewayResp.ModelDetailView{}, err
	}
	detail := gatewayResp.ModelDetailView{
		ModelView: gatewayResp.ModelView{Model: m, Capabilities: capabilitiesToList(m.Capabilities)},
	}
	var deps []gateway.ModelDeployment
	if err := global.OPS_DB.WithContext(ctx).Where("model_id = ?", id).Order("deployment_id ASC").Find(&deps).Error; err != nil {
		return gatewayResp.ModelDetailView{}, err
	}
	ds := DeploymentService{}
	detail.Deployments = make([]gatewayResp.DeploymentView, 0, len(deps))
	for i := range deps {
		detail.Deployments = append(detail.Deployments, ds.toView(ctx, deps[i]))
	}
	var total, active int64
	for i := range detail.Deployments {
		total++
		if detail.Deployments[i].IsActive {
			active++
		}
	}
	detail.DeploymentCount, detail.ActiveDeploymentCount = total, active
	return detail, nil
}

// CreateModel 新增模型。modelKey 可空(先建壳后部署时设置)；设置时未删行查重。
func (s *ModelService) CreateModel(ctx context.Context, req gatewayReq.ModelOperateParams, createBy int64) (gatewayResp.ModelView, error) {
	if req.Name == "" {
		return gatewayResp.ModelView{}, errors.New("模型名称不能为空")
	}
	if req.ModelKey != "" {
		if err := s.ensureModelKeyFree(ctx, req.ModelKey, 0); err != nil {
			return gatewayResp.ModelView{}, err
		}
	}
	m := gateway.Model{
		ModelKey:         req.ModelKey,
		Name:             req.Name,
		Category:         normalizeModelCategory(req.Category),
		Capabilities:     marshalStringList(req.Capabilities),
		Description:      req.Description,
		LogoProviderType: req.LogoProviderType,
		IsActive:         true,
		VisibilityType:   gateway.VisibilityTypeAll,
	}
	m.CreateBy = createBy
	m.UpdateBy = createBy
	if err := global.OPS_DB.WithContext(ctx).Create(&m).Error; err != nil {
		return gatewayResp.ModelView{}, err
	}
	return gatewayResp.ModelView{Model: m, Capabilities: capabilitiesToList(m.Capabilities)}, nil
}

// UpdateModel 修改模型。允许改 modelKey/category——两者都影响部署侧路由名/前缀解析，
// 触发关联部署的级联重建(尽力而为：单个失败记 warning 继续，LiteLLM 与 DB 可能短暂漂移，
// 与凭证启停级联的强一致语义不同，代码注释与日志均需体现)。
func (s *ModelService) UpdateModel(ctx context.Context, req gatewayReq.ModelOperateParams, updateBy int64) (gatewayResp.ModelView, error) {
	if req.ModelId == 0 {
		return gatewayResp.ModelView{}, errors.New("模型ID不能为空")
	}
	if req.Name == "" {
		return gatewayResp.ModelView{}, errors.New("模型名称不能为空")
	}
	var m gateway.Model
	if err := global.OPS_DB.WithContext(ctx).Where("model_id = ?", req.ModelId).First(&m).Error; err != nil {
		return gatewayResp.ModelView{}, err
	}
	newKey := m.ModelKey
	if req.ModelKey != "" {
		newKey = req.ModelKey
		if newKey != m.ModelKey {
			if err := s.ensureModelKeyFree(ctx, newKey, req.ModelId); err != nil {
				return gatewayResp.ModelView{}, err
			}
		}
	}
	newCategory := m.Category
	if req.Category != "" {
		newCategory = normalizeModelCategory(req.Category)
	}

	err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Model(&gateway.Model{}).Where("model_id = ?", req.ModelId).Updates(map[string]any{
			"model_key":          newKey,
			"name":               req.Name,
			"category":           newCategory,
			"capabilities":       marshalStringList(req.Capabilities),
			"description":        req.Description,
			"logo_provider_type": req.LogoProviderType,
			"update_by":          updateBy,
		}).Error
	})
	if err != nil {
		return gatewayResp.ModelView{}, err
	}

	// 级联重建：路由名或类别变化影响部署侧投影(前缀解析/路由组名)
	if (newKey != m.ModelKey || newCategory != m.Category) && newKey != "" {
		updated := m
		updated.ModelKey, updated.Category = newKey, newCategory
		cascadeRebuildModelDeployments(ctx, global.OPS_DB.WithContext(ctx), litellm.Default(), &updated)
	}

	var fresh gateway.Model
	if err := global.OPS_DB.WithContext(ctx).Where("model_id = ?", req.ModelId).First(&fresh).Error; err != nil {
		return gatewayResp.ModelView{}, err
	}
	return gatewayResp.ModelView{Model: fresh, Capabilities: capabilitiesToList(fresh.Capabilities)}, nil
}

// DeleteModels 批量删除模型(软删三连，任一 LiteLLM 禁用失败整体中止)：
// ①每个已同步部署先在 LiteLLM 侧禁用(active=false 留痕，不改名不删记录)
// ②关联部署全部置 is_active=false ③模型软删(is_active=false+deleted_at)+清发布投影(物理删)。
func (s *ModelService) DeleteModels(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	var deps []gateway.ModelDeployment
	if err := global.OPS_DB.WithContext(ctx).Where("model_id IN ?", ids).Find(&deps).Error; err != nil {
		return err
	}
	cli := litellm.Default()
	for i := range deps {
		if deps[i].LitellmModelId == "" || cli == nil {
			continue
		}
		if err := cli.UpdateModel(ctx, deps[i].LitellmModelId, "", nil, withActive(jsonToMap(deps[i].ModelInfo), false)); err != nil {
			return fmt.Errorf("部署 %q 在 LiteLLM 侧禁用失败，已中止删除: %w", deps[i].DeployName, err)
		}
	}
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 部署一并软删(软删行默认查询不可见，避免模型已删后部署列表出现孤儿行)
		if err := tx.Where("model_id IN ?", ids).Delete(&gateway.ModelDeployment{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&gateway.Model{}).Where("model_id IN ?", ids).
			Updates(map[string]any{"is_active": false, "is_published": false}).Error; err != nil {
			return err
		}
		if err := tx.Where("model_id IN ?", ids).Delete(&gateway.Model{}).Error; err != nil {
			return err
		}
		return tx.Unscoped().Where("model_id IN ?", ids).Delete(&gateway.ModelVisibility{}).Error
	})
}

// GetModelPublish 查模型发布设置(含 selected 模式的可见部门)。
func (s *ModelService) GetModelPublish(ctx context.Context, id int64) (gatewayResp.ModelPublishView, error) {
	var m gateway.Model
	if err := global.OPS_DB.WithContext(ctx).Where("model_id = ?", id).First(&m).Error; err != nil {
		return gatewayResp.ModelPublishView{}, err
	}
	view := gatewayResp.ModelPublishView{
		ModelId:          m.ModelId,
		IsPublished:      m.IsPublished,
		VisibilityType:   m.VisibilityType,
		RequiresApproval: m.RequiresApproval,
		DepartmentIds:    []int64{},
	}
	var rows []gateway.ModelVisibility
	if err := global.OPS_DB.WithContext(ctx).Where("model_id = ?", id).Find(&rows).Error; err != nil {
		return gatewayResp.ModelPublishView{}, err
	}
	for _, r := range rows {
		view.DepartmentIds = append(view.DepartmentIds, r.DepartmentId)
	}
	return view, nil
}

// PublishModel 更新发布设置：selected+发布 必须指定可见部门(存在性校验)，
// 可见性行重建(物理删+插，投影表不软删)；取消发布或全员可见时清空可见行。
func (s *ModelService) PublishModel(ctx context.Context, req gatewayReq.ModelPublishParams, updateBy int64) error {
	if req.ModelId == 0 {
		return errors.New("模型ID不能为空")
	}
	visibility := req.VisibilityType
	if visibility == "" {
		visibility = gateway.VisibilityTypeAll
	}
	if visibility != gateway.VisibilityTypeAll && visibility != gateway.VisibilityTypeSelected {
		return errors.New("可见范围取值非法(all/selected)")
	}
	if req.IsPublished && visibility == gateway.VisibilityTypeSelected && len(req.DepartmentIds) == 0 {
		return errors.New("指定部门可见时必须选择至少一个部门")
	}
	var m gateway.Model
	if err := global.OPS_DB.WithContext(ctx).Where("model_id = ?", req.ModelId).First(&m).Error; err != nil {
		return errors.New("模型不存在")
	}
	if len(req.DepartmentIds) > 0 {
		var cnt int64
		if err := global.OPS_DB.WithContext(ctx).Model(&system.SysDepartment{}).
			Where("dept_id IN ?", req.DepartmentIds).Count(&cnt).Error; err != nil {
			return err
		}
		if cnt != int64(len(req.DepartmentIds)) {
			return errors.New("可见部门列表包含不存在的部门")
		}
	}
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&gateway.Model{}).Where("model_id = ?", req.ModelId).Updates(map[string]any{
			"is_published":      req.IsPublished,
			"visibility_type":   visibility,
			"requires_approval": req.RequiresApproval,
			"update_by":         updateBy,
		}).Error; err != nil {
			return err
		}
		// 可见性行重建(物理删：软删行会占唯一索引挡住同组合重新发布)
		if err := tx.Unscoped().Where("model_id = ?", req.ModelId).Delete(&gateway.ModelVisibility{}).Error; err != nil {
			return err
		}
		if req.IsPublished && visibility == gateway.VisibilityTypeSelected {
			rows := make([]gateway.ModelVisibility, 0, len(req.DepartmentIds))
			for _, deptId := range req.DepartmentIds {
				rows = append(rows, gateway.ModelVisibility{ModelId: req.ModelId, DepartmentId: deptId})
			}
			return tx.Create(&rows).Error
		}
		return nil
	})
}

// ----------------------------------------------------------------------------
// 内部工具
// ----------------------------------------------------------------------------

// cascadeRebuildModelDeployments 模型改名/改类后的部署级联重建(尽力而为)：
// 逐个重建投影并推送(路由名切换到新 model_key/新前缀)，单个失败记 warning 继续。
// 注意：与凭证启停级联(事务内强一致)不同，本级联是最终一致——LiteLLM 与 DB 可能短暂漂移。
func cascadeRebuildModelDeployments(ctx context.Context, db *gorm.DB, cli *litellm.Client, model *gateway.Model) []string {
	if cli == nil {
		return nil
	}
	var deps []gateway.ModelDeployment
	if err := db.Where("model_id = ? AND litellm_model_id <> ''", model.ModelId).Find(&deps).Error; err != nil {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Field("modelId", model.ModelId).Error("模型级联重建：查部署失败")
		return nil
	}
	if len(deps) > 20 {
		logger.WithCtx(ctx).Mod("gateway").Field("count", len(deps)).
			Warn("模型级联重建涉及部署较多，同步耗时可能上升(建议后续切片迁异步)")
	}
	var warnings []string
	for i := range deps {
		dep := &deps[i]
		var cred *gateway.Credential
		if dep.CredentialId != 0 {
			var c gateway.Credential
			if err := db.Where("credential_id = ?", dep.CredentialId).First(&c).Error; err == nil {
				cred = &c
			}
		}
		format := "openai"
		if cred != nil {
			if f := formatOf(cred); f != "" {
				format = f
			}
		}
		params, modelInfo := buildDeploymentParams(db, dep, model, cred)
		if err := pushDeployment(ctx, cli, dep, model.ModelKey, format, routableOf(dep.IsActive, cred), params, modelInfo); err != nil {
			warnings = append(warnings, fmt.Sprintf("部署 %q 路由名级联切换失败: %v", dep.DeployName, err))
			continue
		}
		if err := db.Model(&gateway.ModelDeployment{}).Where("deployment_id = ?", dep.DeploymentId).
			Updates(map[string]any{
				"litellm_params":   marshalJSON(params),
				"model_info":       marshalJSON(modelInfo),
				"litellm_model_id": dep.LitellmModelId,
			}).Error; err != nil {
			warnings = append(warnings, fmt.Sprintf("部署 %q 投影写回失败: %v", dep.DeployName, err))
		}
	}
	for _, w := range warnings {
		logger.WithCtx(ctx).Mod("gateway").Warn(w)
	}
	return warnings
}

// ensureModelKeyFree modelKey 未删行查重(excludeId 排除自身)。
func (s *ModelService) ensureModelKeyFree(ctx context.Context, modelKey string, excludeId int64) error {
	var dup int64
	q := global.OPS_DB.WithContext(ctx).Model(&gateway.Model{}).Where("model_key = ?", modelKey)
	if excludeId != 0 {
		q = q.Where("model_id <> ?", excludeId)
	}
	if err := q.Count(&dup).Error; err != nil {
		return err
	}
	if dup > 0 {
		return fmt.Errorf("模型路由名 %q 已存在", modelKey)
	}
	return nil
}

// normalizeModelCategory 类别归一：空串回落 chat。
func normalizeModelCategory(c string) string {
	if c == "" {
		return gateway.ModelCategoryChat
	}
	return c
}

// marshalStringList 字符串数组 → JSONB(nil 给空数组)。
func marshalStringList(list []string) []byte {
	if list == nil {
		list = []string{}
	}
	raw, err := json.Marshal(list)
	if err != nil {
		return []byte("[]")
	}
	return raw
}

// capabilitiesToList JSONB 能力标签 → 字符串数组(空给空数组)。
func capabilitiesToList(raw []byte) []string {
	out := []string{}
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}
