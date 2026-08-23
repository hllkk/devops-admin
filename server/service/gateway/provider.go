package gateway

import (
	"context"
	"errors"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
)

// ProviderService AI 供应商管理(对齐前端 /gateway/provider/* 资源)。
// 供应商是纯管理元数据，不同步 LiteLLM（其下凭证/部署才同步）。
type ProviderService struct{}

// GetProviderList 分页查供应商列表(对齐前端 GET /gateway/provider/list)。
// name 模糊、providerType/billingType 精确、isActive 精确(指针区分未传与 false)。
func (s *ProviderService) GetProviderList(ctx context.Context, q gatewayReq.ProviderSearch) (list []gateway.Provider, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&gateway.Provider{})
	if q.Name != "" {
		db = db.Where("name LIKE ?", "%"+q.Name+"%")
	}
	if q.ProviderType != "" {
		db = db.Where("provider_type = ?", q.ProviderType)
	}
	if q.BillingType != "" {
		db = db.Where("billing_type = ?", q.BillingType)
	}
	if q.IsActive != nil {
		db = db.Where("is_active = ?", *q.IsActive)
	}
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("provider_id DESC").Limit(limit).Offset(offset).Find(&list).Error
	} else {
		err = db.Count(&total).Order("provider_id DESC").Find(&list).Error
	}
	return
}

// GetProvider 查供应商详情(对齐前端 GET /gateway/provider/:id)。
func (s *ProviderService) GetProvider(ctx context.Context, id int64) (gateway.Provider, error) {
	var p gateway.Provider
	err := global.OPS_DB.WithContext(ctx).Where("provider_id = ?", id).First(&p).Error
	return p, err
}

// CreateProvider 新增供应商;createBy 填审计字段。billingType 空走默认 token，isActive 空走默认 true。
func (s *ProviderService) CreateProvider(ctx context.Context, req gatewayReq.ProviderOperateParams, createBy int64) (gateway.Provider, error) {
	if req.Name == "" {
		return gateway.Provider{}, errors.New("供应商名称不能为空")
	}
	if req.ProviderType == "" {
		return gateway.Provider{}, errors.New("供应商类型不能为空")
	}
	p := gateway.Provider{
		Name:         req.Name,
		ProviderType: req.ProviderType,
		BillingType:  normalizeBillingType(req.BillingType),
		MonthlyBudget: req.MonthlyBudget,
		IsActive:      req.IsActive == nil || *req.IsActive, // nil/true → true，显式 false → false
		Description:   req.Description,
	}
	p.CreateBy = createBy
	p.UpdateBy = createBy
	if err := global.OPS_DB.WithContext(ctx).Create(&p).Error; err != nil {
		return gateway.Provider{}, err
	}
	return p, nil
}

// UpdateProvider 修改供应商;providerId 必填。用 map 显式覆盖可编辑字段(含零值)。
// monthlyBudget/isActive 为 nil 时表示不改（不写入 map）。
func (s *ProviderService) UpdateProvider(ctx context.Context, req gatewayReq.ProviderOperateParams, updateBy int64) error {
	if req.ProviderId == 0 {
		return errors.New("供应商ID不能为空")
	}
	if req.Name == "" {
		return errors.New("供应商名称不能为空")
	}
	updates := map[string]any{
		"name":          req.Name,
		"provider_type": req.ProviderType,
		"billing_type":  normalizeBillingType(req.BillingType),
		"description":   req.Description,
		"update_by":     updateBy,
	}
	if req.MonthlyBudget != nil {
		updates["monthly_budget"] = *req.MonthlyBudget
	} else {
		updates["monthly_budget"] = nil // 显式清空预算
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	return global.OPS_DB.WithContext(ctx).Model(&gateway.Provider{}).
		Where("provider_id = ?", req.ProviderId).Updates(updates).Error
}

// DeleteProvider 批量删除供应商(软删除，业务实体一致)。
// 删除前校验关联凭证：provider_id 是纯逻辑关联(不建外键)，此处阻止删除仍有凭证挂靠的供应商。
func (s *ProviderService) DeleteProvider(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	var cnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.Credential{}).
		Where("provider_id IN ?", ids).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return errors.New("供应商下存在关联凭证，请先删除或迁移凭证")
	}
	return global.OPS_DB.WithContext(ctx).Where("provider_id IN ?", ids).Delete(&gateway.Provider{}).Error
}

// normalizeBillingType 计费类型归一：空串回落默认 token。
func normalizeBillingType(b string) string {
	if b == "" {
		return gateway.BillingTypeToken
	}
	return b
}
