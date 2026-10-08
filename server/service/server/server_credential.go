package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/server"
	serverReq "github.com/hllkk/devops-admin/server/model/server/request"
	serverResp "github.com/hllkk/devops-admin/server/model/server/response"
	"github.com/hllkk/devops-admin/server/utils/crypto"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// CredentialService 服务器模块采集凭据管理(SSH/BMC/SNMP/DB 账号)。
// credential_values 落库 AES-256-GCM 加密(server.credential-key)、出网仅敏感键掩码；
// 掩码回传=未修改保留旧明文的合并语义(对齐 gateway credential 先例)。
type CredentialService struct{}

// GetCredentialList 分页查凭据列表(对齐前端 GET /server/credential/list)。
// credentialName 模糊、credentialType/isActive 精确(指针区分未传与 false)。
// 单条解密失败记日志置空 values，不阻断整页。
func (s *CredentialService) GetCredentialList(ctx context.Context, q serverReq.CredentialSearch) (list []serverResp.CredentialView, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&server.Credential{})
	if q.CredentialName != "" {
		db = db.Where("credential_name LIKE ?", "%"+q.CredentialName+"%")
	}
	if q.CredentialType != "" {
		db = db.Where("credential_type = ?", q.CredentialType)
	}
	if q.IsActive != nil {
		db = db.Where("is_active = ?", *q.IsActive)
	}
	var rows []server.Credential
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("credential_id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	} else {
		err = db.Count(&total).Order("credential_id DESC").Find(&rows).Error
	}
	if err != nil {
		return nil, 0, err
	}
	list = make([]serverResp.CredentialView, 0, len(rows))
	for i := range rows {
		list = append(list, s.toView(ctx, rows[i]))
	}
	return list, total, nil
}

// GetCredential 查凭据详情(对齐前端 GET /server/credential/:id)，返回出网视图。
func (s *CredentialService) GetCredential(ctx context.Context, id int64) (serverResp.CredentialView, error) {
	var c server.Credential
	if err := global.OPS_DB.WithContext(ctx).Where("credential_id = ?", id).First(&c).Error; err != nil {
		return serverResp.CredentialView{}, err
	}
	return s.toView(ctx, c), nil
}

// GetCredentialOptions 凭据下拉选项(资产表单选择；type 过滤，仅启用中)。
func (s *CredentialService) GetCredentialOptions(ctx context.Context, credType string) ([]serverResp.CredentialOptions, error) {
	db := global.OPS_DB.WithContext(ctx).Model(&server.Credential{}).
		Where("is_active = ?", true)
	if credType != "" {
		db = db.Where("credential_type = ?", credType)
	}
	var rows []server.Credential
	if err := db.Order("credential_id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	opts := make([]serverResp.CredentialOptions, 0, len(rows))
	for i := range rows {
		opts = append(opts, serverResp.CredentialOptions{
			CredentialId:   rows[i].CredentialId,
			CredentialName: rows[i].CredentialName,
			CredentialType: rows[i].CredentialType,
		})
	}
	return opts, nil
}

// CreateCredential 新增凭据；createBy 填审计字段。
func (s *CredentialService) CreateCredential(ctx context.Context, req serverReq.CredentialOperateParams, createBy int64) (serverResp.CredentialView, error) {
	if req.CredentialName == "" {
		return serverResp.CredentialView{}, errors.New("凭据名称不能为空")
	}
	if !ValidateCredentialType(req.CredentialType) {
		return serverResp.CredentialView{}, fmt.Errorf("凭据类型无效: %q(ssh/bmc/snmp/db)", req.CredentialType)
	}
	if len(req.CredentialValues) == 0 {
		return serverResp.CredentialView{}, errors.New("凭据键值不能为空")
	}
	// 未删行查重(软删不建唯一索引是项目成文决策，靠服务层保证)
	var dup int64
	if err := global.OPS_DB.WithContext(ctx).Model(&server.Credential{}).
		Where("credential_name = ?", req.CredentialName).Count(&dup).Error; err != nil {
		return serverResp.CredentialView{}, err
	}
	if dup > 0 {
		return serverResp.CredentialView{}, fmt.Errorf("凭据名 %q 已存在", req.CredentialName)
	}
	enc, err := encryptCredentialValues(req.CredentialValues)
	if err != nil {
		return serverResp.CredentialView{}, err
	}
	c := server.Credential{
		CredentialName:   req.CredentialName,
		CredentialType:   req.CredentialType,
		CredentialValues: enc,
		Description:      req.Description,
		IsActive:         req.IsActive == nil || *req.IsActive,
	}
	c.CreateBy = createBy
	c.UpdateBy = createBy
	if err := global.OPS_DB.WithContext(ctx).Create(&c).Error; err != nil {
		return serverResp.CredentialView{}, err
	}
	return s.toView(ctx, c), nil
}

// UpdateCredential 修改凭据；credentialId 必填。
// credential_type 建后不可改(域内字段集合不同)；credential_values 合并语义：
// 敏感键掩码回传=未修改保留旧明文，新值覆盖，可新增；旧键不在新值中即删除。
func (s *CredentialService) UpdateCredential(ctx context.Context, req serverReq.CredentialOperateParams, updateBy int64) (serverResp.CredentialView, error) {
	if req.CredentialId == 0 {
		return serverResp.CredentialView{}, errors.New("凭据ID不能为空")
	}
	var c server.Credential
	if err := global.OPS_DB.WithContext(ctx).Where("credential_id = ?", req.CredentialId).First(&c).Error; err != nil {
		return serverResp.CredentialView{}, err
	}
	if req.CredentialType != "" && req.CredentialType != c.CredentialType {
		return serverResp.CredentialView{}, errors.New("凭据类型不可修改(删除后按新类型重建)")
	}
	// 同名查重(改名与其他未软删行冲突)
	if req.CredentialName != "" && req.CredentialName != c.CredentialName {
		var dup int64
		if err := global.OPS_DB.WithContext(ctx).Model(&server.Credential{}).
			Where("credential_name = ? AND credential_id <> ?", req.CredentialName, req.CredentialId).
			Count(&dup).Error; err != nil {
			return serverResp.CredentialView{}, err
		}
		if dup > 0 {
			return serverResp.CredentialView{}, fmt.Errorf("凭据名 %q 已存在", req.CredentialName)
		}
	}
	oldValues, err := decryptCredentialValues(c.CredentialValues)
	if err != nil {
		return serverResp.CredentialView{}, fmt.Errorf("凭据旧值解密失败: %w", err)
	}
	newValues := MergeCredentialValues(oldValues, req.CredentialValues)
	if len(newValues) == 0 {
		return serverResp.CredentialView{}, errors.New("凭据键值不能为空")
	}
	enc, err := encryptCredentialValues(newValues)
	if err != nil {
		return serverResp.CredentialView{}, err
	}
	updates := map[string]any{
		"credential_values": enc,
		"update_by":         updateBy,
	}
	if req.CredentialName != "" {
		updates["credential_name"] = req.CredentialName
	}
	if req.Description != c.Description {
		updates["description"] = req.Description
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if err := global.OPS_DB.WithContext(ctx).Model(&server.Credential{}).
		Where("credential_id = ?", req.CredentialId).Updates(updates).Error; err != nil {
		return serverResp.CredentialView{}, err
	}
	// 回读最新行返回视图
	var fresh server.Credential
	if err := global.OPS_DB.WithContext(ctx).Where("credential_id = ?", req.CredentialId).First(&fresh).Error; err != nil {
		return serverResp.CredentialView{}, err
	}
	return s.toView(ctx, fresh), nil
}

// DeleteCredential 批量删除凭据(软删除)。删除前校验资产引用
// (纯逻辑关联不建外键，service 层保证)：被资产关联即拒绝。
func (s *CredentialService) DeleteCredential(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	var assetCnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&server.Asset{}).
		Where("credential_id IN ?", ids).Count(&assetCnt).Error; err != nil {
		return err
	}
	if assetCnt > 0 {
		return fmt.Errorf("凭据被 %d 个资产关联，请先解除资产关联", assetCnt)
	}
	return global.OPS_DB.WithContext(ctx).Where("credential_id IN ?", ids).Delete(&server.Credential{}).Error
}

// ----------------------------------------------------------------------------
// 内部工具
// ----------------------------------------------------------------------------

// toView 模型转出网视图：解密 credential_values 后仅掩码敏感键。
// 解密失败(如密钥轮换后历史密文)记日志置空 values，不阻断列表/详情。
func (s *CredentialService) toView(ctx context.Context, c server.Credential) serverResp.CredentialView {
	view := serverResp.CredentialView{Credential: c}
	values, err := decryptCredentialValues(c.CredentialValues)
	if err != nil {
		logger.WithCtx(ctx).Mod("server").Err(err).Field("credentialId", c.CredentialId).Error("凭据值解密失败")
		view.CredentialValues = map[string]string{}
		return view
	}
	view.CredentialValues = MaskCredentialValues(values)
	return view
}

// encryptCredentialValues 凭据键值序列化后 AES-256-GCM 加密；密钥未配置则拒绝写入。
func encryptCredentialValues(values map[string]string) (string, error) {
	key := global.OPS_CONFIG.ServerModule.CredentialKey
	if key == "" {
		return "", errors.New("凭据加密密钥未配置(server.credential-key)，拒绝写入")
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("凭据键值序列化失败: %w", err)
	}
	return crypto.AESGCMEncrypt(string(raw), key)
}

// decryptCredentialValues 解密凭据键值；空串返回空 map(允许无键值的历史行)。
func decryptCredentialValues(enc string) (map[string]string, error) {
	if enc == "" {
		return map[string]string{}, nil
	}
	key := global.OPS_CONFIG.ServerModule.CredentialKey
	if key == "" {
		return nil, errors.New("凭据加密密钥未配置(server.credential-key)")
	}
	raw, err := crypto.AESGCMDecrypt(enc, key)
	if err != nil {
		return nil, err
	}
	var values map[string]string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("凭据键值反序列化失败: %w", err)
	}
	return values, nil
}
