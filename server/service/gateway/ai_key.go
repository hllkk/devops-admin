package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils/litellm"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// AiKeyService AI 密钥管理(对齐前端 /gateway/ai-key/* 与 /gateway/ai-key/identity/* 资源)。
// LiteLLM 是 Key 管理中心(生成/鉴权/限流)，devops-admin 存管理元数据+加密 key_value(偏离
// AIHelms 不存：home 需明文展示、LiteLLM 只返回一次、已有 AES-GCM 基座)。
type AiKeyService struct{}

// ----------------------------------------------------------------------------
// 身份与对外接口（home 切真实接口的契约）
// ----------------------------------------------------------------------------

// GetMyIdentity 我的 AI 身份：惰性建主 Key(无则建，含公开模型+anthropic 扩展) →
// 解密主 Key 明文 + 我的场景 Key 列表 + 可用模型。仅 owner 本人可调(userId 从 JWT 取)。
func (s *AiKeyService) GetMyIdentity(ctx context.Context, userId int64) (gatewayResp.MyIdentityView, error) {
	mainKey, err := ensureMainKeyExists(ctx, userId)
	if err != nil {
		return gatewayResp.MyIdentityView{}, err
	}

	view := gatewayResp.MyIdentityView{
		IsActive:        mainKey.IsActive,
		BudgetLimit:     mainKey.BudgetLimit,
		BudgetHardLimit: mainKey.BudgetHardLimit,
		BudgetDuration:  mainKey.BudgetDuration,
		Models:          jsonToSlice(mainKey.Models),
		ModelBudgets:    jsonToMap(mainKey.ModelBudgets),
		RateLimitMode:   mainKey.RateLimitMode,
		TpmLimit:        mainKey.TpmLimit,
		RpmLimit:        mainKey.RpmLimit,
		SceneKeys:       []gatewayResp.AiKeyView{},
	}
	// 主 Key 明文(仅此接口解密返回)；单机模式(litellm_key_id 空)无可用 Key 返回空
	if mainKey.LitellmKeyId != "" {
		if plain, err := decryptCredentialValues(mainKey.KeyValue); err == nil {
			if v, ok := plain["k"].(string); ok {
				view.KeyValue = v
			}
		}
	}

	// 我的场景 Key 列表(personal_scene, owner=user)
	var sceneRows []gateway.AiKey
	if err := global.OPS_DB.WithContext(ctx).
		Where("owner_type = ? AND owner_id = ? AND key_type = ?", gateway.OwnerTypeUser, userId, gateway.KeyTypePersonalScene).
		Order("ai_key_id DESC").Find(&sceneRows).Error; err != nil {
		return gatewayResp.MyIdentityView{}, err
	}
	for i := range sceneRows {
		view.SceneKeys = append(view.SceneKeys, toAiKeyView(sceneRows[i]))
	}

	view.AvailableModels, _ = s.GetAvailableModels(ctx)
	return view, nil
}

// GetAvailableModels 可授权模型(active+published，含 anthropic 变体标注)，复用 slice3 逻辑。
func (s *AiKeyService) GetAvailableModels(ctx context.Context) ([]gatewayResp.AvailableModelView, error) {
	var models []gateway.Model
	if err := global.OPS_DB.WithContext(ctx).
		Where("is_active = ? AND is_published = ?", true, true).
		Order("model_id DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	// 活跃部署→anthropic 标注(复用 slice3 GetActiveModels 同款联查)
	anthropicKeys := map[int64]bool{}
	modelIds := make([]int64, 0, len(models))
	for i := range models {
		modelIds = append(modelIds, models[i].ModelId)
	}
	if len(modelIds) > 0 {
		var deps []gateway.ModelDeployment
		_ = global.OPS_DB.WithContext(ctx).
			Where("is_active = ? AND credential_id <> 0 AND model_id IN ?", true, modelIds).Find(&deps).Error
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
				anthropicKeys[deps[i].ModelId] = true
			}
		}
	}

	list := make([]gatewayResp.AvailableModelView, 0, len(models))
	for i := range models {
		m := models[i]
		v := gatewayResp.AvailableModelView{
			ModelId:          m.ModelId,
			ModelKey:         m.ModelKey,
			Name:             m.Name,
			Category:         m.Category,
			RequiresApproval: m.RequiresApproval,
		}
		if anthropicKeys[m.ModelId] {
			v.HasAnthropicDeployment = true
			v.ModelKeyAnthropic = m.ModelKey + gateway.ModelAnthropicSuffix
		}
		list = append(list, v)
	}
	return list, nil
}

// ----------------------------------------------------------------------------
// 管理员 CRUD
// ----------------------------------------------------------------------------

// GetAiKeyList 分页查密钥列表(管理员视角，不返回 KeyValue)。
func (s *AiKeyService) GetAiKeyList(ctx context.Context, q gatewayReq.AiKeySearch) (list []gatewayResp.AiKeyView, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&gateway.AiKey{})
	if q.KeyType != "" {
		db = db.Where("key_type = ?", q.KeyType)
	}
	if q.OwnerType != "" {
		db = db.Where("owner_type = ?", q.OwnerType)
	}
	if q.OwnerId != 0 {
		db = db.Where("owner_id = ?", q.OwnerId)
	}
	if q.Name != "" {
		db = db.Where("name LIKE ?", "%"+q.Name+"%")
	}
	if q.IsActive != nil {
		db = db.Where("is_active = ?", *q.IsActive)
	}
	var rows []gateway.AiKey
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("ai_key_id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	} else {
		err = db.Count(&total).Order("ai_key_id DESC").Find(&rows).Error
	}
	if err != nil {
		return nil, 0, err
	}
	list = make([]gatewayResp.AiKeyView, 0, len(rows))
	for i := range rows {
		list = append(list, toAiKeyView(rows[i]))
	}
	return list, total, nil
}

// GetAiKey 查密钥详情(管理员视角，不返回 KeyValue)。
func (s *AiKeyService) GetAiKey(ctx context.Context, id int64) (gatewayResp.AiKeyView, error) {
	var k gateway.AiKey
	if err := global.OPS_DB.WithContext(ctx).Where("ai_key_id = ?", id).First(&k).Error; err != nil {
		return gatewayResp.AiKeyView{}, err
	}
	return toAiKeyView(k), nil
}

// CreateSceneKey 创建密钥(场景 Key 或管理员手动建部门主 Key)。
// 校验 owner 存在 + 主 Key 唯一性(主 Key 类型每 owner 仅一个) → buildKeyAlias →
// anthropic 扩展 → LiteLLM CreateKey(返回明文一次) → key_value 加密落库 → 返回视图。
func (s *AiKeyService) CreateSceneKey(ctx context.Context, req gatewayReq.AiKeyOperateParams, createBy int64) (gatewayResp.AiKeyView, error) {
	if req.KeyType == "" || req.OwnerType == "" || req.OwnerId == 0 {
		return gatewayResp.AiKeyView{}, errors.New("密钥类型/归属类型/归属ID不能为空")
	}
	if req.Name == "" {
		return gatewayResp.AiKeyView{}, errors.New("密钥名称不能为空")
	}
	if !validKeyType(req.KeyType, req.OwnerType) {
		return gatewayResp.AiKeyView{}, errors.New("密钥类型与归属类型不匹配(user→personal/dept→dept)")
	}
	if err := s.ensureOwnerExists(ctx, req.OwnerType, req.OwnerId); err != nil {
		return gatewayResp.AiKeyView{}, err
	}
	// 主 Key 唯一性(每 owner 仅一个 main)
	if gateway.MainKeyType(req.KeyType) {
		var cnt int64
		if err := global.OPS_DB.WithContext(ctx).Model(&gateway.AiKey{}).
			Where("key_type = ? AND owner_type = ? AND owner_id = ?", req.KeyType, req.OwnerType, req.OwnerId).
			Count(&cnt).Error; err != nil {
			return gatewayResp.AiKeyView{}, err
		}
		if cnt > 0 {
			return gatewayResp.AiKeyView{}, errors.New("该归属已有主 Key")
		}
	}

	alias := buildKeyAlias(req.KeyType, req.OwnerType, req.OwnerId, req.Name)
	models := req.Models
	if models == nil {
		models = []string{}
	}

	k := gateway.AiKey{
		Name:            req.Name,
		Description:     req.Description,
		KeyType:         req.KeyType,
		OwnerType:       req.OwnerType,
		OwnerId:         req.OwnerId,
		LitellmKeyAlias: alias,
		Models:          marshalJSONStringSlice(models),
		ModelBudgets:   marshalJSONMap(req.ModelBudgets),
		Mcps:            datatypes.JSON([]byte("[]")),
		Skills:          datatypes.JSON([]byte("[]")),
		BudgetLimit:     req.BudgetLimit,
		BudgetHardLimit: req.BudgetHardLimit != nil && *req.BudgetHardLimit,
		BudgetDuration:  normalizeBudgetDuration(req.BudgetDuration),
		RateLimitMode:   normalizeRateLimitMode(req.RateLimitMode),
		TpmLimit:        req.TpmLimit,
		RpmLimit:        req.RpmLimit,
		ModelLimits:     marshalJSONMap(req.ModelLimits),
		IsActive:        req.IsActive == nil || *req.IsActive,
	}
	k.CreateBy = createBy
	k.UpdateBy = createBy

	cli := litellm.Default()
	err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&k).Error; err != nil {
			return err
		}
		if cli == nil {
			return nil // 单机模式：不调 LiteLLM，key_value 空
		}
		return syncKeyToLitellm(ctx, cli, tx, &k, true)
	})
	if err != nil {
		return gatewayResp.AiKeyView{}, err
	}
	return toAiKeyView(k), nil
}

// UpdateAiKey 修改密钥：改授权/预算/限流/启停 → 重建 LiteLLM 投影并 UpdateKey。
// key_type/owner 不可改(改=删旧建新)。
func (s *AiKeyService) UpdateAiKey(ctx context.Context, req gatewayReq.AiKeyOperateParams, updateBy int64) (gatewayResp.AiKeyView, error) {
	if req.AiKeyId == 0 {
		return gatewayResp.AiKeyView{}, errors.New("密钥ID不能为空")
	}
	var k gateway.AiKey
	if err := global.OPS_DB.WithContext(ctx).Where("ai_key_id = ?", req.AiKeyId).First(&k).Error; err != nil {
		return gatewayResp.AiKeyView{}, err
	}
	if req.KeyType != "" && req.KeyType != k.KeyType {
		return gatewayResp.AiKeyView{}, errors.New("密钥类型不可修改(需删旧建新)")
	}

	models := req.Models
	if models == nil {
		models = jsonToSlice(k.Models)
	}
	updates := map[string]any{
		"name":              req.Name,
		"description":       req.Description,
		"models":            marshalJSONStringSlice(models),
		"model_budgets":    marshalJSONMap(req.ModelBudgets),
		"budget_duration":   normalizeBudgetDuration(req.BudgetDuration),
		"rate_limit_mode":   normalizeRateLimitMode(req.RateLimitMode),
		"tpm_limit":         req.TpmLimit,
		"rpm_limit":         req.RpmLimit,
		"model_limits":      marshalJSONMap(req.ModelLimits),
		"update_by":         updateBy,
	}
	if req.BudgetLimit != nil {
		updates["budget_limit"] = *req.BudgetLimit
		k.BudgetLimit = req.BudgetLimit
	}
	if req.BudgetHardLimit != nil {
		updates["budget_hard_limit"] = *req.BudgetHardLimit
		k.BudgetHardLimit = *req.BudgetHardLimit
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
		k.IsActive = *req.IsActive
	}
	if req.Name != "" {
		k.Name = req.Name
	}
	k.Models = marshalJSONStringSlice(models)
	k.ModelBudgets = marshalJSONMap(req.ModelBudgets)
	k.ModelLimits = marshalJSONMap(req.ModelLimits)
	k.RateLimitMode = normalizeRateLimitMode(req.RateLimitMode)
	k.TpmLimit = req.TpmLimit
	k.RpmLimit = req.RpmLimit
	k.BudgetDuration = normalizeBudgetDuration(req.BudgetDuration)

	cli := litellm.Default()
	err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&gateway.AiKey{}).Where("ai_key_id = ?", req.AiKeyId).Updates(updates).Error; err != nil {
			return err
		}
		if cli == nil || k.LitellmKeyId == "" {
			return nil
		}
		return syncKeyToLitellm(ctx, cli, tx, &k, false)
	})
	if err != nil {
		return gatewayResp.AiKeyView{}, err
	}
	var fresh gateway.AiKey
	if err := global.OPS_DB.WithContext(ctx).Where("ai_key_id = ?", req.AiKeyId).First(&fresh).Error; err != nil {
		return gatewayResp.AiKeyView{}, err
	}
	return toAiKeyView(fresh), nil
}

// DeleteAiKey 批量删除密钥：先删 LiteLLM(失败本地不动)→本地软删。
func (s *AiKeyService) DeleteAiKey(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	var rows []gateway.AiKey
	if err := global.OPS_DB.WithContext(ctx).Where("ai_key_id IN ?", ids).Find(&rows).Error; err != nil {
		return err
	}
	cli := litellm.Default()
	for i := range rows {
		if rows[i].LitellmKeyId == "" || cli == nil {
			continue
		}
		if err := cli.DeleteKey(ctx, rows[i].LitellmKeyId); err != nil {
			return fmt.Errorf("密钥 %q 删除 LiteLLM 失败，已中止(本地未动): %w", rows[i].Name, err)
		}
	}
	return global.OPS_DB.WithContext(ctx).Where("ai_key_id IN ?", ids).Delete(&gateway.AiKey{}).Error
}

// ----------------------------------------------------------------------------
// 包级共享：同步管线与惰性建主 Key
// ----------------------------------------------------------------------------

// ensureMainKeyExists 惰性建/补全个人主 Key：无则建(含所有公开模型+anthropic 扩展)，
// 有则补缺失的公开模型(幂等，发布配置变化后下次访问 home 自愈)。
func ensureMainKeyExists(ctx context.Context, userId int64) (*gateway.AiKey, error) {
	var k gateway.AiKey
	err := global.OPS_DB.WithContext(ctx).
		Where("key_type = ? AND owner_type = ? AND owner_id = ?", gateway.KeyTypePersonalMain, gateway.OwnerTypeUser, userId).
		First(&k).Error

	if err == nil {
		// 已有主 Key：补缺失公开模型(幂等自愈)
		publicKeys := publicModelKeys(global.OPS_DB.WithContext(ctx))
		current := jsonToSlice(k.Models)
		missing := []string{}
		seen := map[string]bool{}
		for _, m := range current {
			seen[m] = true
		}
		for _, pk := range publicKeys {
			if !seen[pk] {
				missing = append(missing, pk)
				current = append(current, pk)
				seen[pk] = true
			}
		}
		if len(missing) > 0 {
			cli := litellm.Default()
			_ = global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				if err := tx.Model(&gateway.AiKey{}).Where("ai_key_id = ?", k.AiKeyId).
					Update("models", marshalJSONStringSlice(current)).Error; err != nil {
					return err
				}
				k.Models = marshalJSONStringSlice(current)
				if cli != nil && k.LitellmKeyId != "" {
					_ = syncKeyToLitellm(ctx, cli, tx, &k, false)
				}
				return nil
			})
		}
		return &k, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 建主 Key：含所有公开模型
	publicKeys := publicModelKeys(global.OPS_DB.WithContext(ctx))
	k = gateway.AiKey{
		Name:            "主 Key",
		Description:     "个人主 Key",
		KeyType:         gateway.KeyTypePersonalMain,
		OwnerType:       gateway.OwnerTypeUser,
		OwnerId:         userId,
		LitellmKeyAlias: buildKeyAlias(gateway.KeyTypePersonalMain, gateway.OwnerTypeUser, userId, "main"),
		Models:          marshalJSONStringSlice(publicKeys),
		ModelBudgets:    datatypes.JSON([]byte("{}")),
		Mcps:            datatypes.JSON([]byte("[]")),
		Skills:          datatypes.JSON([]byte("[]")),
		BudgetDuration:  gateway.BudgetDuration30d,
		RateLimitMode:   gateway.RateLimitModeNone,
		ModelLimits:     datatypes.JSON([]byte("{}")),
		IsActive:        true,
	}
	k.CreateBy = userId
	k.UpdateBy = userId

	cli := litellm.Default()
	err = global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&k).Error; err != nil {
			return err
		}
		if cli == nil {
			return nil
		}
		return syncKeyToLitellm(ctx, cli, tx, &k, true)
	})
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// syncPublicModelToMainKeys 发布公开模型时遍历所有 active 主 Key 追加 modelKey 并同步
// LiteLLM(事务内，单个失败 warning 继续)。回补 slice3 PublishModel 的 TODO。
func syncPublicModelToMainKeys(ctx context.Context, tx *gorm.DB, modelKey string) []string {
	var keys []gateway.AiKey
	if err := tx.
		Where("key_type IN ? AND is_active = ?", []string{gateway.KeyTypePersonalMain, gateway.KeyTypeDeptMain}, true).
		Find(&keys).Error; err != nil {
		return []string{err.Error()}
	}
	cli := litellm.Default()
	var warnings []string
	for i := range keys {
		current := jsonToSlice(keys[i].Models)
		if sliceContains(current, modelKey) {
			continue // 已授权
		}
		current = append(current, modelKey)
		keys[i].Models = marshalJSONStringSlice(current)
		if err := tx.Model(&gateway.AiKey{}).Where("ai_key_id = ?", keys[i].AiKeyId).
			Update("models", keys[i].Models).Error; err != nil {
			warnings = append(warnings, fmt.Sprintf("主Key %d: %v", keys[i].AiKeyId, err))
			continue
		}
		if cli != nil && keys[i].LitellmKeyId != "" {
			if err := syncKeyToLitellm(ctx, cli, tx, &keys[i], false); err != nil {
				warnings = append(warnings, fmt.Sprintf("主Key %d: %v", keys[i].AiKeyId, err))
			}
		}
	}
	for _, w := range warnings {
		logger.WithCtx(ctx).Mod("gateway").Warn(w)
	}
	return warnings
}

// syncKeyToLitellm 同步密钥到 LiteLLM：isCreate=true 走 CreateKey(返回明文加密落库)，
// 否则 UpdateKey(改授权/预算/限流/启停)。models 经 anthropic 扩展；max_budget 按硬限/启停语义。
func syncKeyToLitellm(ctx context.Context, cli *litellm.Client, tx *gorm.DB, k *gateway.AiKey, isCreate bool) error {
	db := tx
	expandedModels := expandModelsWithAnthropic(db, jsonToSlice(k.Models))

	// max_budget 语义：停用→0；硬限+有额度→limit；否则→nil(不限)
	var maxBudget *float64
	syncBudget := false
	if !k.IsActive {
		zero := 0.0
		maxBudget = &zero
		syncBudget = true
	} else if k.BudgetHardLimit && k.BudgetLimit != nil {
		maxBudget = k.BudgetLimit
		syncBudget = true
	} else if k.BudgetHardLimit && k.BudgetLimit == nil {
		// 硬限但无额度=停用语义
		zero := 0.0
		maxBudget = &zero
		syncBudget = true
	} else {
		syncBudget = true // 启用无硬限→清空(nil→null)
	}

	metadata := map[string]any{
		"aiKeyId":  k.AiKeyId,
		"keyType":  k.KeyType,
	}
	// per-model 限流(per_model 模式，含 anthropic 变体复制)
	if k.RateLimitMode == gateway.RateLimitModePerModel {
		limits := jsonToMap(k.ModelLimits)
		if tpmMap, rpmMap := buildPerModelLimitMaps(db, limits, expandedModels); tpmMap != nil || rpmMap != nil {
			if tpmMap != nil {
				metadata["model_tpm_limit"] = tpmMap
			}
			if rpmMap != nil {
				metadata["model_rpm_limit"] = rpmMap
			}
		}
	}

	if isCreate {
		req := litellm.KeyCreateReq{
			KeyAlias:  k.LitellmKeyAlias,
			Models:    expandedModels,
			MaxBudget: maxBudget,
			Metadata:  metadata,
		}
		if k.RateLimitMode == gateway.RateLimitModeTotal {
			req.TPMLimit = k.TpmLimit
			req.RPMLimit = k.RpmLimit
		}
		if k.OwnerType == gateway.OwnerTypeUser {
			req.UserID = fmt.Sprintf("devops_user_%d", k.OwnerId)
		}
		resp, err := cli.CreateKey(ctx, req)
		if err != nil {
			return fmt.Errorf("密钥同步 LiteLLM 失败: %w", err)
		}
		k.LitellmKeyId = resp.KeyID
		// 加密 key_value(明文仅此一次返回，加密落库)
		enc, err := encryptCredentialValues(map[string]any{"k": resp.Key})
		if err != nil {
			return fmt.Errorf("密钥明文加密失败: %w", err)
		}
		k.KeyValue = enc
		k.KeyPrefix = keyPrefixOf(resp.Key)
		return tx.Model(&gateway.AiKey{}).Where("ai_key_id = ?", k.AiKeyId).
			Updates(map[string]any{
				"litellm_key_id":    k.LitellmKeyId,
				"key_value":         k.KeyValue,
				"key_prefix":        k.KeyPrefix,
			}).Error
	}

	// Update
	req := litellm.KeyUpdateReq{
		Models:      expandedModels,
		MaxBudget:   maxBudget,
		Metadata:    metadata,
		SyncBudget:  syncBudget,
	}
	if k.RateLimitMode == gateway.RateLimitModeTotal {
		req.TPMLimit = k.TpmLimit
		req.RPMLimit = k.RpmLimit
		req.SyncRateLimits = true
	}
	return cli.UpdateKey(ctx, k.LitellmKeyId, req)
}

// ----------------------------------------------------------------------------
// 包级工具
// ----------------------------------------------------------------------------

// buildKeyAlias LiteLLM key_alias 格式：{ownerType}:{ownerId}/{name}，主 Key 末段固定 main。
func buildKeyAlias(keyType, ownerType string, ownerId int64, name string) string {
	prefix := ownerType
	if gateway.MainKeyType(keyType) {
		name = "main"
	}
	return fmt.Sprintf("%s:%d/%s", prefix, ownerId, name)
}

// queryAnthropicModelKeys 给定 modelKey 列表，查出存在 anthropic 活跃部署的 modelKey 集合。
func queryAnthropicModelKeys(db *gorm.DB, modelKeys []string) map[string]bool {
	out := map[string]bool{}
	if len(modelKeys) == 0 {
		return out
	}
	var models []gateway.Model
	db.Where("model_key IN ?", modelKeys).Find(&models)
	if len(models) == 0 {
		return out
	}
	modelIds := make([]int64, 0, len(models))
	keyById := map[int64]string{}
	for i := range models {
		modelIds = append(modelIds, models[i].ModelId)
		keyById[models[i].ModelId] = models[i].ModelKey
	}
	var deps []gateway.ModelDeployment
	db.Where("is_active = ? AND credential_id <> 0 AND model_id IN ?", true, modelIds).Find(&deps)
	if len(deps) == 0 {
		return out
	}
	credIds := make([]int64, 0, len(deps))
	for i := range deps {
		credIds = append(credIds, deps[i].CredentialId)
	}
	creds := map[int64]*gateway.Credential{}
	var credRows []gateway.Credential
	db.Where("credential_id IN ?", credIds).Find(&credRows)
	for i := range credRows {
		creds[credRows[i].CredentialId] = &credRows[i]
	}
	for i := range deps {
		if cred, ok := creds[deps[i].CredentialId]; ok && cred.IsActive && formatOf(cred) == "anthropic" {
			if mk, ok := keyById[deps[i].ModelId]; ok {
				out[mk] = true
			}
		}
	}
	return out
}

// expandModelsWithAnthropic 对有 anthropic 活跃部署的模型追加 {modelKey}(Anthropic) 变体。
func expandModelsWithAnthropic(db *gorm.DB, models []string) []string {
	anthropicSet := queryAnthropicModelKeys(db, models)
	out := make([]string, 0, len(models)+len(anthropicSet))
	out = append(out, models...)
	seen := map[string]bool{}
	for _, m := range models {
		seen[m] = true
	}
	for _, m := range models {
		if anthropicSet[m] {
			variant := m + gateway.ModelAnthropicSuffix
			if !seen[variant] {
				out = append(out, variant)
				seen[variant] = true
			}
		}
	}
	return out
}

// buildPerModelLimitMaps 从 model_limits({modelKey:{tpm,rpm}}) 构建 LiteLLM metadata 的
// model_tpm_limit/model_rpm_limit map，并对 anthropic 变体复制同额限流。
func buildPerModelLimitMaps(db *gorm.DB, limits map[string]any, expandedModels []string) (tpmMap, rpmMap map[string]any) {
	anthropicVariants := map[string]string{} // 变体名 → 原 modelKey
	for _, m := range expandedModels {
		if strings.HasSuffix(m, gateway.ModelAnthropicSuffix) {
			anthropicVariants[m] = strings.TrimSuffix(m, gateway.ModelAnthropicSuffix)
		}
	}
	for mk, v := range limits {
		tpm, rpm := extractTpmRpm(v)
		if tpmMap == nil && tpm != nil {
			tpmMap = map[string]any{}
		}
		if rpmMap == nil && rpm != nil {
			rpmMap = map[string]any{}
		}
		if tpm != nil {
			tpmMap[mk] = *tpm
		}
		if rpm != nil {
			rpmMap[mk] = *rpm
		}
		// 复制到 anthropic 变体
		for variant, orig := range anthropicVariants {
			if orig == mk {
				if tpm != nil {
					if tpmMap == nil {
						tpmMap = map[string]any{}
					}
					tpmMap[variant] = *tpm
				}
				if rpm != nil {
					if rpmMap == nil {
						rpmMap = map[string]any{}
					}
					rpmMap[variant] = *rpm
				}
			}
		}
	}
	return
}

// extractTpmRpm 从 model_limits 值({tpm,rpm})提取。
func extractTpmRpm(v any) (tpm, rpm *int) {
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	if t, ok := m["tpm"]; ok {
		if n, ok := toInt(t); ok {
			tpm = &n
		}
	}
	if r, ok := m["rpm"]; ok {
		if n, ok := toInt(r); ok {
			rpm = &n
		}
	}
	return
}

// toInt JSON 数值容错转 int。
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		return int(i), err == nil
	}
	return 0, false
}

// publicModelKeys 公开模型 modelKey 列表(is_published AND !requires_approval AND is_active)。
func publicModelKeys(db *gorm.DB) []string {
	var keys []string
	db.Model(&gateway.Model{}).
		Where("is_active = ? AND is_published = ? AND requires_approval = ?", true, true, false).
		Where("model_key <> ''").Pluck("model_key", &keys)
	return keys
}

// keyPrefixOf Key 明文前缀(前8位+****，短于8位全*)。
func keyPrefixOf(plain string) string {
	if len(plain) <= 8 {
		return strings.Repeat("*", len(plain))
	}
	return plain[:8] + "****"
}

// validKeyType 校验 key_type 与 owner_type 匹配。
func validKeyType(keyType, ownerType string) bool {
	switch keyType {
	case gateway.KeyTypePersonalMain, gateway.KeyTypePersonalScene:
		return ownerType == gateway.OwnerTypeUser
	case gateway.KeyTypeDeptMain, gateway.KeyTypeDeptScene:
		return ownerType == gateway.OwnerTypeDept
	}
	return false
}

// normalizeBudgetDuration 预算周期归一。
func normalizeBudgetDuration(d string) string {
	switch d {
	case gateway.BudgetDuration1d, gateway.BudgetDuration7d, gateway.BudgetDuration30d:
		return d
	}
	return gateway.BudgetDuration30d
}

// normalizeRateLimitMode 限流模式归一。
func normalizeRateLimitMode(m string) string {
	switch m {
	case gateway.RateLimitModeNone, gateway.RateLimitModeTotal, gateway.RateLimitModePerModel:
		return m
	}
	return gateway.RateLimitModeNone
}

// ensureOwnerExists 校验归属存在(user/dept)。
func (s *AiKeyService) ensureOwnerExists(ctx context.Context, ownerType string, ownerId int64) error {
	switch ownerType {
	case gateway.OwnerTypeUser:
		var cnt int64
		if err := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).
			Where("id = ?", ownerId).Count(&cnt).Error; err != nil { // SysUser.UserId 列复用 id
			return err
		}
		if cnt == 0 {
			return errors.New("归属用户不存在")
		}
	case gateway.OwnerTypeDept:
		var cnt int64
		if err := global.OPS_DB.WithContext(ctx).Model(&system.SysDepartment{}).
			Where("dept_id = ?", ownerId).Count(&cnt).Error; err != nil {
			return err
		}
		if cnt == 0 {
			return errors.New("归属部门不存在")
		}
	default:
		return errors.New("归属类型非法(user/dept)")
	}
	return nil
}

// toAiKeyView 模型转出网视图(不返回 KeyValue)。
func toAiKeyView(k gateway.AiKey) gatewayResp.AiKeyView {
	return gatewayResp.AiKeyView{
		AiKey:        k,
		Models:       jsonToSlice(k.Models),
		ModelBudgets: jsonToMap(k.ModelBudgets),
		ModelLimits:  jsonToMap(k.ModelLimits),
	}
}

// jsonToSlice JSONB → []string(空给空数组)。
func jsonToSlice(raw datatypes.JSON) []string {
	out := []string{}
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}

// marshalJSONStringSlice []string → datatypes.JSON(nil 给空数组)。
func marshalJSONStringSlice(list []string) datatypes.JSON {
	if list == nil {
		list = []string{}
	}
	raw, _ := json.Marshal(list)
	return datatypes.JSON(raw)
}

// marshalJSONMap map → datatypes.JSON(nil 给空对象)。
func marshalJSONMap(m map[string]any) datatypes.JSON {
	if m == nil {
		return datatypes.JSON([]byte("{}"))
	}
	raw, _ := json.Marshal(m)
	return datatypes.JSON(raw)
}

// sliceContains 切片包含判定。
func sliceContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
