package gateway

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
)

// KeyScenarioService 使用场景字典(密钥域内，对齐 AIHelms key_scenarios：极简 name+description+启停)。
// 场景是场景 Key 的分类标签而非资源配置模板(那是 P4 业务场景的职责)；管理员在密钥管理页场景 Tab 维护，
// 建 Key 表单下拉选择。偏离 AIHelms 一点：删除走软删(gorm)而非置 is_active=false——
// 停用行占名防同名二义，软删行不占名可重建(规避其 unique 撞死坑)。
type KeyScenarioService struct{}

// GetKeyScenarioList 分页查场景列表(场景 Tab)。
func (s *KeyScenarioService) GetKeyScenarioList(ctx context.Context, q gatewayReq.KeyScenarioSearch) (list []gateway.KeyScenario, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&gateway.KeyScenario{})
	if q.Name != "" {
		db = db.Where("name LIKE ?", "%"+q.Name+"%")
	}
	if q.IsActive != nil {
		db = db.Where("is_active = ?", *q.IsActive)
	}
	var rows []gateway.KeyScenario
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("scenario_id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	} else {
		err = db.Count(&total).Order("scenario_id DESC").Find(&rows).Error
	}
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// GetAllScenarios 启用中的场景全量(建 Key 表单下拉/筛选用，不分页)。
func (s *KeyScenarioService) GetAllScenarios(ctx context.Context) ([]gateway.KeyScenario, error) {
	var rows []gateway.KeyScenario
	err := global.OPS_DB.WithContext(ctx).
		Where("is_active = ?", true).Order("scenario_id ASC").Find(&rows).Error
	return rows, err
}

// CreateKeyScenario 新增场景(name 未软删行内唯一)。
func (s *KeyScenarioService) CreateKeyScenario(ctx context.Context, req gatewayReq.KeyScenarioOperateParams, createBy int64) (gateway.KeyScenario, error) {
	if req.Name == "" {
		return gateway.KeyScenario{}, errors.New("场景名称不能为空")
	}
	if err := ensureScenarioNameFree(ctx, req.Name, 0); err != nil {
		return gateway.KeyScenario{}, err
	}
	sc := gateway.KeyScenario{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    req.IsActive == nil || *req.IsActive,
	}
	sc.CreateBy = createBy
	sc.UpdateBy = createBy
	if err := global.OPS_DB.WithContext(ctx).Create(&sc).Error; err != nil {
		return gateway.KeyScenario{}, err
	}
	return sc, nil
}

// UpdateKeyScenario 修改场景(改名查重；停用后新建 Key 不可选，存量 Key 不受影响)。
func (s *KeyScenarioService) UpdateKeyScenario(ctx context.Context, req gatewayReq.KeyScenarioOperateParams, updateBy int64) (gateway.KeyScenario, error) {
	if req.ScenarioId == 0 {
		return gateway.KeyScenario{}, errors.New("场景ID不能为空")
	}
	var sc gateway.KeyScenario
	if err := global.OPS_DB.WithContext(ctx).Where("scenario_id = ?", req.ScenarioId).First(&sc).Error; err != nil {
		return gateway.KeyScenario{}, err
	}
	if req.Name != "" && req.Name != sc.Name {
		if err := ensureScenarioNameFree(ctx, req.Name, req.ScenarioId); err != nil {
			return gateway.KeyScenario{}, err
		}
	}
	updates := map[string]any{
		"description": req.Description,
		"update_by":   updateBy,
	}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.KeyScenario{}).
		Where("scenario_id = ?", req.ScenarioId).Updates(updates).Error; err != nil {
		return gateway.KeyScenario{}, err
	}
	return sc, nil
}

// DeleteKeyScenario 批量删除场景：被未软删密钥引用时拒删(先在密钥列表解除引用)，否则软删。
func (s *KeyScenarioService) DeleteKeyScenario(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	var refCnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.AiKey{}).
		Where("scenario_id IN ?", ids).Count(&refCnt).Error; err != nil {
		return err
	}
	if refCnt > 0 {
		return fmt.Errorf("所选场景仍被 %d 个密钥引用，请先在密钥列表解除引用", refCnt)
	}
	return global.OPS_DB.WithContext(ctx).Where("scenario_id IN ?", ids).Delete(&gateway.KeyScenario{}).Error
}

// ensureScenarioNameFree 场景名查重(未软删行口径，排除自身)。
func ensureScenarioNameFree(ctx context.Context, name string, excludeId int64) error {
	var cnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.KeyScenario{}).
		Where("name = ? AND scenario_id <> ?", name, excludeId).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return errors.New("场景名称已存在")
	}
	return nil
}

// ensureScenarioUsable 校验场景存在且启用(建 Key/改 Key 选场景时)。
func ensureScenarioUsable(ctx context.Context, g *gorm.DB, scenarioId int64) error {
	var cnt int64
	if err := g.WithContext(ctx).Model(&gateway.KeyScenario{}).
		Where("scenario_id = ? AND is_active = ?", scenarioId, true).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt == 0 {
		return errors.New("所选场景不存在或已停用")
	}
	return nil
}
