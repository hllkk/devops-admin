package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	"github.com/hllkk/devops-admin/server/utils/crypto"
	"github.com/hllkk/devops-admin/server/utils/litellm"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// CredentialService 凭证管理(对齐前端 /gateway/credential/* 资源)。
// 同步遵循「平台 DB 是唯一事实源，LiteLLM 是投影」：CRUD 事务内联单向推送，
// 推送失败整体回滚不留半成品；credential_values 落库 AES-256-GCM 加密、出网仅敏感 key 掩码。
type CredentialService struct{}

// GetCredentialList 分页查凭证列表(对齐前端 GET /gateway/credential/list)。
// credentialName 模糊、providerId 精确(0=不限)、isActive/litellmSynced 精确(指针区分未传与 false)。
// 返回出网视图：单条解密失败记日志置空 values，不阻断整页。
func (s *CredentialService) GetCredentialList(ctx context.Context, q gatewayReq.CredentialSearch) (list []gatewayResp.CredentialView, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&gateway.Credential{})
	if q.CredentialName != "" {
		db = db.Where("credential_name LIKE ?", "%"+q.CredentialName+"%")
	}
	if q.ProviderId != 0 {
		db = db.Where("provider_id = ?", q.ProviderId)
	}
	if q.IsActive != nil {
		db = db.Where("is_active = ?", *q.IsActive)
	}
	if q.LitellmSynced != nil {
		db = db.Where("litellm_synced = ?", *q.LitellmSynced)
	}
	var rows []gateway.Credential
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("credential_id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	} else {
		err = db.Count(&total).Order("credential_id DESC").Find(&rows).Error
	}
	if err != nil {
		return nil, 0, err
	}
	list = make([]gatewayResp.CredentialView, 0, len(rows))
	for i := range rows {
		list = append(list, s.toView(ctx, rows[i]))
	}
	return list, total, nil
}

// GetCredential 查凭证详情(对齐前端 GET /gateway/credential/:id)，返回出网视图。
func (s *CredentialService) GetCredential(ctx context.Context, id int64) (gatewayResp.CredentialView, error) {
	var c gateway.Credential
	if err := global.OPS_DB.WithContext(ctx).Where("credential_id = ?", id).First(&c).Error; err != nil {
		return gatewayResp.CredentialView{}, err
	}
	return s.toView(ctx, c), nil
}

// CreateCredential 新增凭证；createBy 填审计字段。
// 事务内：落库(密文) → 推送 LiteLLM(POST 失败转 PATCH 的 upsert，双败回滚) → 置 litellm_synced。
// LiteLLM 未配置(单机模式)时跳过推送，synced 保持 false。
func (s *CredentialService) CreateCredential(ctx context.Context, req gatewayReq.CredentialOperateParams, createBy int64) (gatewayResp.CredentialView, error) {
	if req.CredentialName == "" {
		return gatewayResp.CredentialView{}, errors.New("凭证名称不能为空")
	}
	if len(req.CredentialValues) == 0 {
		return gatewayResp.CredentialView{}, errors.New("凭证键值不能为空")
	}
	if err := s.ensureProviderExists(ctx, req.ProviderId); err != nil {
		return gatewayResp.CredentialView{}, err
	}
	// 未删行查重(软删不建唯一索引是项目成文决策，靠服务层保证；LiteLLM 侧删除时已清，同名可重建)
	var dup int64
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.Credential{}).
		Where("credential_name = ?", req.CredentialName).Count(&dup).Error; err != nil {
		return gatewayResp.CredentialView{}, err
	}
	if dup > 0 {
		return gatewayResp.CredentialView{}, fmt.Errorf("凭证名 %q 已存在", req.CredentialName)
	}

	info := normalizeCredentialInfo(req.CredentialInfo)
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return gatewayResp.CredentialView{}, err
	}
	enc, err := encryptCredentialValues(req.CredentialValues)
	if err != nil {
		return gatewayResp.CredentialView{}, err
	}
	c := gateway.Credential{
		CredentialName:   req.CredentialName,
		ProviderId:       req.ProviderId,
		CredentialValues: enc,
		CredentialInfo:   datatypes.JSON(infoJSON),
		IsActive:         req.IsActive == nil || *req.IsActive, // nil/true → true，显式 false → false
		Description:      req.Description,
	}
	c.CreateBy = createBy
	c.UpdateBy = createBy

	cli := litellm.Default()
	providerType := s.providerTypeOf(ctx, req.ProviderId)
	err = global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&c).Error; err != nil {
			return err
		}
		if cli == nil {
			return nil // 单机模式：不推送，synced 保持 false
		}
		payload := BuildLitellmCredentialValues(req.CredentialValues, info, providerType)
		if err := pushCredential(ctx, cli, c.CredentialName, payload, info); err != nil {
			return err
		}
		return tx.Model(&gateway.Credential{}).Where("credential_id = ?", c.CredentialId).
			Update("litellm_synced", true).Error
	})
	if err != nil {
		return gatewayResp.CredentialView{}, err
	}
	c.LitellmSynced = cli != nil
	return s.toView(ctx, c), nil
}

// UpdateCredential 修改凭证；credentialId 必填。credential_name 是 LiteLLM 侧键，
// 改名=删旧建新会瞬时摘除全部引用部署，故不允许修改。
// credential_values 合并语义：敏感 key 掩码回传=未修改保留旧明文，新值覆盖，可新增；
// 仅投影变化才重推 LiteLLM(懒同步)；启停不推凭证本身(凭证无 active 概念，可用性由部署路由控制)。
// 级联(强一致，事务内收口)：投影/启停/换绑供应商任一变化 → 同步关联部署路由
// (__disabled__ 后缀摘出/回 LB 组 + params 重建，如 api_base/前缀变化)。
func (s *CredentialService) UpdateCredential(ctx context.Context, req gatewayReq.CredentialOperateParams, updateBy int64) (gatewayResp.CredentialUpdateResult, error) {
	if req.CredentialId == 0 {
		return gatewayResp.CredentialUpdateResult{}, errors.New("凭证ID不能为空")
	}
	if req.CredentialName != "" {
		var old gateway.Credential
		if err := global.OPS_DB.WithContext(ctx).Where("credential_id = ?", req.CredentialId).First(&old).Error; err != nil {
			return gatewayResp.CredentialUpdateResult{}, err
		}
		if req.CredentialName != old.CredentialName {
			return gatewayResp.CredentialUpdateResult{}, errors.New("凭证名称不可修改(LiteLLM 凭证键，改名需删旧建新)")
		}
	}
	if err := s.ensureProviderExists(ctx, req.ProviderId); err != nil {
		return gatewayResp.CredentialUpdateResult{}, err
	}

	var c gateway.Credential
	if err := global.OPS_DB.WithContext(ctx).Where("credential_id = ?", req.CredentialId).First(&c).Error; err != nil {
		return gatewayResp.CredentialUpdateResult{}, err
	}
	oldValues, err := decryptCredentialValues(c.CredentialValues)
	if err != nil {
		return gatewayResp.CredentialUpdateResult{}, fmt.Errorf("凭证旧值解密失败: %w", err)
	}
	oldInfo := credentialInfoToMap(c.CredentialInfo)
	newValues := MergeCredentialValues(oldValues, req.CredentialValues)
	newInfo := oldInfo
	if req.CredentialInfo != nil {
		newInfo = normalizeCredentialInfo(req.CredentialInfo)
	}
	enc, err := encryptCredentialValues(newValues)
	if err != nil {
		return gatewayResp.CredentialUpdateResult{}, err
	}
	newInfoJSON, err := json.Marshal(newInfo)
	if err != nil {
		return gatewayResp.CredentialUpdateResult{}, err
	}

	// 懒同步判定：新旧投影不一致才推送(派生值只存在于投影，比对在投影侧)
	cli := litellm.Default()
	oldProviderType := s.providerTypeOf(ctx, c.ProviderId)
	newProviderType := s.providerTypeOf(ctx, req.ProviderId)
	payloadChanged := !reflect.DeepEqual(
		BuildLitellmCredentialValues(newValues, newInfo, newProviderType),
		BuildLitellmCredentialValues(oldValues, oldInfo, oldProviderType),
	)
	// 级联触发：投影/启停/换绑供应商(前缀解析随 provider_type 变)任一变化
	activeChanged := req.IsActive != nil && *req.IsActive != c.IsActive
	providerChanged := req.ProviderId != c.ProviderId

	updates := map[string]any{
		"provider_id":       req.ProviderId,
		"credential_values": enc,
		"credential_info":   datatypes.JSON(newInfoJSON),
		"description":       req.Description,
		"update_by":         updateBy,
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	result := gatewayResp.CredentialUpdateResult{DeploymentErrors: []string{}}
	err = global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&gateway.Credential{}).Where("credential_id = ?", req.CredentialId).Updates(updates).Error; err != nil {
			return err
		}
		if payloadChanged && cli != nil {
			payload := BuildLitellmCredentialValues(newValues, newInfo, newProviderType)
			if err := pushCredential(ctx, cli, c.CredentialName, payload, newInfo); err != nil {
				return err
			}
			if err := tx.Model(&gateway.Credential{}).Where("credential_id = ?", req.CredentialId).
				Update("litellm_synced", true).Error; err != nil {
				return err
			}
		}
		// 部署路由级联(强一致，事务内收口)：单个失败计 errs 上报，不中断整体
		if cli != nil && (payloadChanged || activeChanged || providerChanged) {
			cascCred := c // 更新后状态快照(routable 判定与解密新值用)
			cascCred.ProviderId = req.ProviderId
			cascCred.CredentialValues = enc
			if req.IsActive != nil {
				cascCred.IsActive = *req.IsActive
			}
			result.DeploymentsSynced, result.DeploymentErrors = syncCredentialRouting(ctx, tx, cli, &cascCred)
		}
		return nil
	})
	if err != nil {
		return gatewayResp.CredentialUpdateResult{}, err
	}
	// 回读最新行返回视图(含 is_active/审计字段更新)
	var fresh gateway.Credential
	if err := global.OPS_DB.WithContext(ctx).Where("credential_id = ?", req.CredentialId).First(&fresh).Error; err != nil {
		return gatewayResp.CredentialUpdateResult{}, err
	}
	result.CredentialView = s.toView(ctx, fresh)
	return result, nil
}

// syncCredentialRouting 凭证变化后的部署路由级联(强一致，事务内收口不独立 commit)：
// 遍历绑定该凭证且已同步的部署 → 重建投影(params 含凭证新 api_base/前缀) → 推送
// (路由名三态：不可路由加 __disabled__ 摘出 LB 组 + model_info.active 双写，litellm_model_id 不变)。
// 单个失败计数上报不中断(个别部署可能漂移，由响应 errs 暴露给管理员)。
func syncCredentialRouting(ctx context.Context, tx *gorm.DB, cli *litellm.Client, cred *gateway.Credential) (int, []string) {
	var deps []gateway.ModelDeployment
	if err := tx.Where("credential_id = ? AND litellm_model_id <> ''", cred.CredentialId).
		Find(&deps).Error; err != nil {
		return 0, []string{err.Error()}
	}
	format := formatOf(cred)
	if format == "" {
		format = "openai"
	}
	synced, errs := 0, []string{}
	for i := range deps {
		dep := &deps[i]
		var model gateway.Model
		if err := tx.Where("model_id = ?", dep.ModelId).First(&model).Error; err != nil {
			errs = append(errs, fmt.Sprintf("部署 %q: 关联模型缺失: %v", dep.DeployName, err))
			continue
		}
		params, modelInfo := buildDeploymentParams(tx, dep, &model, cred)
		if err := pushDeployment(ctx, cli, dep, model.ModelKey, format, routableOf(dep.IsActive, cred), params, modelInfo); err != nil {
			errs = append(errs, fmt.Sprintf("部署 %q: %v", dep.DeployName, err))
			continue
		}
		if err := tx.Model(&gateway.ModelDeployment{}).Where("deployment_id = ?", dep.DeploymentId).
			Updates(map[string]any{
				"litellm_params":   marshalJSON(params),
				"model_info":       marshalJSON(modelInfo),
				"litellm_model_id": dep.LitellmModelId,
			}).Error; err != nil {
			errs = append(errs, fmt.Sprintf("部署 %q: 投影写回失败: %v", dep.DeployName, err))
			continue
		}
		synced++
	}
	return synced, errs
}

// DeleteCredential 批量删除凭证(软删除，业务实体一致)。
// 删除前校验部署引用(纯逻辑关联不建外键，service 层保证)：被部署引用即拒绝；
// 删除顺序：先删 LiteLLM 投影(失败则本地不动)，全部成功后再软删本地行；
// synced=false 的行(单机模式建的)跳过 LiteLLM 直接删本地。
func (s *CredentialService) DeleteCredential(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	var depCnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.ModelDeployment{}).
		Where("credential_id IN ?", ids).Count(&depCnt).Error; err != nil {
		return err
	}
	if depCnt > 0 {
		return fmt.Errorf("凭证被 %d 个部署引用，请先删除或解除部署关联", depCnt)
	}
	var rows []gateway.Credential
	if err := global.OPS_DB.WithContext(ctx).Where("credential_id IN ?", ids).Find(&rows).Error; err != nil {
		return err
	}
	cli := litellm.Default()
	for i := range rows {
		if rows[i].LitellmSynced && cli != nil {
			if err := cli.DeleteCredential(ctx, rows[i].CredentialName); err != nil {
				return fmt.Errorf("凭证 %q 删除 LiteLLM 投影失败，已中止(本地未动): %w", rows[i].CredentialName, err)
			}
		}
	}
	return global.OPS_DB.WithContext(ctx).Where("credential_id IN ?", ids).Delete(&gateway.Credential{}).Error
}

// GetProviderFields 拉取 LiteLLM 各供应商的凭证表单字段定义(GET /public/providers/fields)，
// 供前端动态渲染凭证表单。LiteLLM 未配置时返回空列表(前端降级为手填)。
func (s *CredentialService) GetProviderFields(ctx context.Context) ([]map[string]any, error) {
	cli := litellm.Default()
	if cli == nil {
		return []map[string]any{}, nil
	}
	return cli.GetProviderFields(ctx)
}

// ResyncCredentials 手动重同步全部凭证到 LiteLLM(兜底：静默漂移/远端被误删后的补偿)。
// 逐行投影比对：远端缺失或 synced=false → 推送(POST 失败转 PATCH)；投影与远端不一致 → PATCH；
// 单条失败记入 Failed 不中断整体。
func (s *CredentialService) ResyncCredentials(ctx context.Context) (gatewayResp.ResyncResult, error) {
	result := gatewayResp.ResyncResult{Failed: []string{}}
	cli := litellm.Default()
	if cli == nil {
		return result, litellm.ErrNotConfigured
	}
	remote, err := cli.ListCredentials(ctx)
	if err != nil {
		return result, err
	}
	remoteMap := make(map[string]litellm.CredentialItem, len(remote))
	for _, item := range remote {
		remoteMap[item.CredentialName] = item
	}
	var rows []gateway.Credential
	if err := global.OPS_DB.WithContext(ctx).Find(&rows).Error; err != nil {
		return result, err
	}
	result.Total = len(rows)
	for i := range rows {
		row := rows[i]
		values, err := decryptCredentialValues(row.CredentialValues)
		if err != nil {
			logger.WithCtx(ctx).Mod("gateway").Err(err).Field("credentialId", row.CredentialId).Error("resync: 凭证值解密失败")
			result.Failed = append(result.Failed, row.CredentialName)
			continue
		}
		info := credentialInfoToMap(row.CredentialInfo)
		payload := BuildLitellmCredentialValues(values, info, s.providerTypeOf(ctx, row.ProviderId))
		item, exists := remoteMap[row.CredentialName]
		needPush := !exists || !row.LitellmSynced || !reflect.DeepEqual(payload, item.CredentialValues)
		if !needPush {
			result.Skipped++
			continue
		}
		if err := pushCredential(ctx, cli, row.CredentialName, payload, info); err != nil {
			logger.WithCtx(ctx).Mod("gateway").Err(err).Field("credentialName", row.CredentialName).Error("resync: 凭证投影推送失败")
			result.Failed = append(result.Failed, row.CredentialName)
			continue
		}
		if err := global.OPS_DB.WithContext(ctx).Model(&gateway.Credential{}).
			Where("credential_id = ?", row.CredentialId).Update("litellm_synced", true).Error; err != nil {
			result.Failed = append(result.Failed, row.CredentialName)
			continue
		}
		result.Pushed++
	}
	return result, nil
}

// ----------------------------------------------------------------------------
// 内部工具
// ----------------------------------------------------------------------------

// pushCredential 推送凭证投影到 LiteLLM：POST 失败转 PATCH 的 upsert 语义
// (超时后 LiteLLM 实际已建成功等残留场景，重名 POST 冲突转全量 PATCH 即可对齐)。
func pushCredential(ctx context.Context, cli *litellm.Client, name string, payload, info map[string]any) error {
	if err := cli.CreateCredential(ctx, name, payload, info); err != nil {
		if err2 := cli.UpdateCredential(ctx, name, payload, info); err2 != nil {
			return fmt.Errorf("凭证同步 LiteLLM 失败: %w", err2)
		}
	}
	return nil
}

// toView 模型转出网视图：解密 credential_values 后仅掩码敏感 key。
// 解密失败(如密钥轮换后历史密文)记日志置空 values，不阻断列表/详情。
func (s *CredentialService) toView(ctx context.Context, c gateway.Credential) gatewayResp.CredentialView {
	view := gatewayResp.CredentialView{Credential: c}
	values, err := decryptCredentialValues(c.CredentialValues)
	if err != nil {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Field("credentialId", c.CredentialId).Error("凭证值解密失败")
		view.CredentialValues = map[string]any{}
		return view
	}
	view.CredentialValues = MaskCredentialValues(values)
	return view
}

// encryptValues 凭证键值序列化后 AES-256-GCM 加密；密钥未配置则拒绝写入(对齐 sys_social 先例)。
func encryptCredentialValues(values map[string]any) (string, error) {
	key := global.OPS_CONFIG.Litellm.CredentialKey
	if key == "" {
		return "", errors.New("凭证加密密钥未配置(litellm.credential-key)，拒绝写入")
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("凭证键值序列化失败: %w", err)
	}
	return crypto.AESGCMEncrypt(string(raw), key)
}

// decryptValues 解密凭证键值；空串返回空 map(允许无键值的历史行)。
func decryptCredentialValues(enc string) (map[string]any, error) {
	if enc == "" {
		return map[string]any{}, nil
	}
	key := global.OPS_CONFIG.Litellm.CredentialKey
	if key == "" {
		return nil, errors.New("凭证加密密钥未配置(litellm.credential-key)")
	}
	raw, err := crypto.AESGCMDecrypt(enc, key)
	if err != nil {
		return nil, err
	}
	var values map[string]any
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("凭证键值反序列化失败: %w", err)
	}
	return values, nil
}

// normalizeCredentialInfo 归一凭证元数据：nil 给空 map，format 空补默认 openai。
func normalizeCredentialInfo(info map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range info {
		out[k] = v
	}
	if s, ok := out["format"].(string); !ok || s == "" {
		out["format"] = gateway.CredentialFormatOpenai
	}
	return out
}

// credentialInfoToMap datatypes.JSON 转 map；空/解析失败给空 map。
func credentialInfoToMap(info datatypes.JSON) map[string]any {
	out := map[string]any{}
	if len(info) == 0 {
		return out
	}
	_ = json.Unmarshal(info, &out)
	return out
}

// ensureProviderExists 校验关联供应商存在(纯逻辑关联不建外键，service 层保证)。
func (s *CredentialService) ensureProviderExists(ctx context.Context, providerId int64) error {
	if providerId == 0 {
		return nil // 0=未关联，允许
	}
	var cnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.Provider{}).
		Where("provider_id = ?", providerId).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt == 0 {
		return errors.New("关联供应商不存在")
	}
	return nil
}

// providerTypeOf 查供应商类型(投影构建用)；未关联/查询失败返回 ""(走非 vllm 分支，不派生)。
func (s *CredentialService) providerTypeOf(ctx context.Context, providerId int64) string {
	if providerId == 0 {
		return ""
	}
	var p gateway.Provider
	if err := global.OPS_DB.WithContext(ctx).Select("provider_type").
		Where("provider_id = ?", providerId).First(&p).Error; err != nil {
		return ""
	}
	return p.ProviderType
}
