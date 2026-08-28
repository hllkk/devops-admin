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
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
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

// GetMyIdentity 我的 AI 身份：查主 Key(管理员创建制，无则返回未开通态 opened=false) →
// 解密主 Key 明文 + 我的场景 Key 列表 + 可用模型。仅 owner 本人可调(userId 从 JWT 取)。
func (s *AiKeyService) GetMyIdentity(ctx context.Context, userId int64) (gatewayResp.MyIdentityView, error) {
	mainKey, err := loadMainKey(ctx, userId)
	if err != nil {
		return gatewayResp.MyIdentityView{}, err
	}

	view := gatewayResp.MyIdentityView{
		SceneKeys:      []gatewayResp.AiKeyView{},
		Models:         []string{},
		ModelBudgets:   jsonToMap(datatypes.JSON(nil)),
		RateLimitMode:  gateway.RateLimitModeNone,
		BudgetDuration: gateway.BudgetDuration30d,
	}
	view.AvailableModels, _ = s.GetMyVisibleModels(ctx, userId)
	// 网关接入点(litellm public-url)：客户端直连 Base URL，接入信息展示用
	view.GatewayUrl = global.OPS_CONFIG.Litellm.PublicURL
	if mainKey == nil {
		return view, nil // 未开通：等管理员在后台创建主 Key
	}
	view.Opened = true
	view.IsActive = mainKey.IsActive
	view.BudgetLimit = mainKey.BudgetLimit
	view.BudgetHardLimit = mainKey.BudgetHardLimit
	view.BudgetDuration = mainKey.BudgetDuration
	view.ExpiresAt = mainKey.ExpiresAt
	view.Models = jsonToSlice(mainKey.Models)
	view.ModelBudgets = jsonToMap(mainKey.ModelBudgets)
	view.RateLimitMode = mainKey.RateLimitMode
	view.TpmLimit = mainKey.TpmLimit
	view.RpmLimit = mainKey.RpmLimit
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
	fillScenarioNames(ctx, view.SceneKeys)
	return view, nil
}

// GetAvailableModels 可授权模型(active+published，含 anthropic 变体标注)——管理员全量视角
// (建 Key 授权下拉用)，不按用户可见性过滤。
func (s *AiKeyService) GetAvailableModels(ctx context.Context) ([]gatewayResp.AvailableModelView, error) {
	return s.listModelsAsAvailable(ctx, func(q *gorm.DB) *gorm.DB { return q })
}

// GetMyVisibleModels 当前用户可见的激活模型(按发布可见性过滤)：all 档直通/selected 档
// 命中归属部门/user 档命中用户投影。home「可用模型」卡数据源，与 GetAvailableModels
// (管理员全量)区分。
func (s *AiKeyService) GetMyVisibleModels(ctx context.Context, userId int64) ([]gatewayResp.AvailableModelView, error) {
	db := global.OPS_DB.WithContext(ctx)
	return s.listModelsAsAvailable(ctx, func(q *gorm.DB) *gorm.DB {
		return visibleModelScope(q, userId, userDeptIdOf(db, userId))
	})
}

// listModelsAsAvailable 激活模型 → AvailableModelView 贫字段列表(scope 注入过滤条件)。
func (s *AiKeyService) listModelsAsAvailable(ctx context.Context, scope func(*gorm.DB) *gorm.DB) ([]gatewayResp.AvailableModelView, error) {
	var models []gateway.Model
	if err := scope(global.OPS_DB.WithContext(ctx).Model(&gateway.Model{}).
		Where("is_active = ? AND is_published = ?", true, true)).
		Order("model_id DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	modelIds := make([]int64, 0, len(models))
	for i := range models {
		modelIds = append(modelIds, models[i].ModelId)
	}
	anthropicKeys := annotateAnthropic(ctx, global.OPS_DB, modelIds)

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
	if q.ScenarioId != 0 {
		db = db.Where("scenario_id = ?", q.ScenarioId)
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
	fillOwnerNames(ctx, list)
	fillScenarioNames(ctx, list)
	return list, total, nil
}

// GetAiKey 查密钥详情(管理员视角，不返回 KeyValue)。
func (s *AiKeyService) GetAiKey(ctx context.Context, id int64) (gatewayResp.AiKeyView, error) {
	var k gateway.AiKey
	if err := global.OPS_DB.WithContext(ctx).Where("ai_key_id = ?", id).First(&k).Error; err != nil {
		return gatewayResp.AiKeyView{}, err
	}
	v := toAiKeyView(k)
	fillOwnerNames(ctx, []gatewayResp.AiKeyView{v})
	fillScenarioNames(ctx, []gatewayResp.AiKeyView{v})
	return v, nil
}

// RevealAiKeyValue 查密钥完整明文(管理员把 Key 复制给用户的支撑接口)：解密 key_value 返回。
// 仅超管/管理员可调；单机模式(litellm_key_id 空)与未同步 LiteLLM 的行无明文可用。
func (s *AiKeyService) RevealAiKeyValue(ctx context.Context, id int64, claims *systemReq.CustomClaims) (gatewayResp.AiKeyRevealView, error) {
	if !keyRevealAllowed(ctx, claims) {
		return gatewayResp.AiKeyRevealView{}, errors.New("仅管理员/超管可查看密钥明文")
	}
	var k gateway.AiKey
	if err := global.OPS_DB.WithContext(ctx).Where("ai_key_id = ?", id).First(&k).Error; err != nil {
		return gatewayResp.AiKeyRevealView{}, err
	}
	if k.LitellmKeyId == "" {
		return gatewayResp.AiKeyRevealView{}, errors.New("密钥未同步 LiteLLM，无可用明文")
	}
	plain, err := decryptCredentialValues(k.KeyValue)
	if err != nil {
		return gatewayResp.AiKeyRevealView{}, fmt.Errorf("密钥明文解密失败: %w", err)
	}
	v, _ := plain["k"].(string)
	if v == "" {
		return gatewayResp.AiKeyRevealView{}, errors.New("密钥明文为空")
	}
	return gatewayResp.AiKeyRevealView{KeyValue: v}, nil
}

// keyRevealAllowed 查看明文权限：JWT 超管标志直接放行；其余查启用角色，
// 任一角色带 SuperAdmin 标志或 RoleKey=admin(系统管理员)即视为管理员。
func keyRevealAllowed(ctx context.Context, claims *systemReq.CustomClaims) bool {
	if claims == nil {
		return false
	}
	if claims.SuperAdmin {
		return true
	}
	var user system.SysUser
	if err := global.OPS_DB.WithContext(ctx).
		Preload("Roles", "status = ?", "0"). // 与 getUserInfo 组装口径一致：仅启用角色参与
		Where("id = ?", claims.BaseClaims.ID).First(&user).Error; err != nil {
		return false
	}
	for i := range user.Roles {
		if user.Roles[i].SuperAdmin || user.Roles[i].RoleKey == "admin" {
			return true
		}
	}
	return false
}

// fillOwnerNames 批量填充 view.OwnerName/OwnerUsername(user→SysUser.NickName+Username;
// dept→SysDepartment.DeptName)，按 ownerType 分组一次性 IN 查询，避免逐行 N+1；
// 查不到留空(不影响主流程)。
func fillOwnerNames(ctx context.Context, list []gatewayResp.AiKeyView) {
	if len(list) == 0 {
		return
	}
	userIDs := make(map[int64]struct{})
	deptIDs := make(map[int64]struct{})
	for i := range list {
		switch list[i].OwnerType {
		case gateway.OwnerTypeUser:
			userIDs[list[i].OwnerId] = struct{}{}
		case gateway.OwnerTypeDept:
			deptIDs[list[i].OwnerId] = struct{}{}
		}
	}
	type userBrief struct {
		NickName string
		UserName string
	}
	userMap := make(map[int64]userBrief, len(userIDs))
	deptMap := make(map[int64]string, len(deptIDs))
	if len(userIDs) > 0 {
		ids := make([]int64, 0, len(userIDs))
		for id := range userIDs {
			ids = append(ids, id)
		}
		var us []system.SysUser
		if err := global.OPS_DB.WithContext(ctx).Select("id, nick_name, user_name").Where("id IN ?", ids).Find(&us).Error; err == nil {
			for _, u := range us {
				userMap[u.UserId] = userBrief{NickName: u.NickName, UserName: u.UserName}
			}
		}
	}
	if len(deptIDs) > 0 {
		ids := make([]int64, 0, len(deptIDs))
		for id := range deptIDs {
			ids = append(ids, id)
		}
		var ds []system.SysDepartment
		if err := global.OPS_DB.WithContext(ctx).Select("dept_id, dept_name").Where("dept_id IN ?", ids).Find(&ds).Error; err == nil {
			for _, d := range ds {
				deptMap[d.DeptId] = d.DeptName
			}
		}
	}
	for i := range list {
		switch list[i].OwnerType {
		case gateway.OwnerTypeUser:
			list[i].OwnerName = userMap[list[i].OwnerId].NickName
			list[i].OwnerUsername = userMap[list[i].OwnerId].UserName
		case gateway.OwnerTypeDept:
			list[i].OwnerName = deptMap[list[i].OwnerId]
		}
	}
}

// fillScenarioNames 批量填充 view.ScenarioName(逻辑关联 gateway_key_scenario，一次 IN 查询；
// 场景查不到留空，不影响主流程)。
func fillScenarioNames(ctx context.Context, list []gatewayResp.AiKeyView) {
	ids := make(map[int64]struct{})
	for i := range list {
		if list[i].ScenarioId != 0 {
			ids[list[i].ScenarioId] = struct{}{}
		}
	}
	if len(ids) == 0 {
		return
	}
	slice := make([]int64, 0, len(ids))
	for id := range ids {
		slice = append(slice, id)
	}
	names := map[int64]string{}
	var rows []gateway.KeyScenario
	if err := global.OPS_DB.WithContext(ctx).Select("scenario_id, name").
		Where("scenario_id IN ?", slice).Find(&rows).Error; err == nil {
		for _, r := range rows {
			names[r.ScenarioId] = r.Name
		}
	}
	for i := range list {
		if list[i].ScenarioId != 0 {
			list[i].ScenarioName = names[list[i].ScenarioId]
		}
	}
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

	// 名称同类归属下唯一(防 LiteLLM key_alias 撞车:alias={ownerType}:{ownerId}/{name})；
	// 软删记录由 gorm 自动过滤(deleted_at IS NULL)，删后可重建同名。
	var nameCnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.AiKey{}).
		Where("key_type = ? AND owner_type = ? AND owner_id = ? AND name = ?", req.KeyType, req.OwnerType, req.OwnerId, req.Name).
		Count(&nameCnt).Error; err != nil {
		return gatewayResp.AiKeyView{}, err
	}
	if nameCnt > 0 {
		return gatewayResp.AiKeyView{}, errors.New("该归属下已存在同名密钥")
	}

	alias := buildKeyAlias(req.KeyType, req.OwnerType, req.OwnerId, req.Name)
	models := req.Models
	if models == nil {
		models = []string{}
	}
	// 主 Key 未显式指定授权模型时默认含对其可见的全部免审批模型(对齐原"全部公开模型"
	// 默认语义；定向发布对可见范围同样生效——个人主 Key 按用户可见,部门主 Key 按部门可见)
	if gateway.MainKeyType(req.KeyType) && len(models) == 0 {
		db := global.OPS_DB.WithContext(ctx)
		if req.OwnerType == gateway.OwnerTypeUser {
			models = visibleModelKeys(db, req.OwnerId, userDeptIdOf(db, req.OwnerId))
		} else {
			models = visibleModelKeys(db, 0, req.OwnerId)
		}
	}

	// 场景归属(仅场景 Key；主 Key 恒无场景)
	scenarioId := int64(0)
	if req.ScenarioId != nil && *req.ScenarioId != 0 {
		if gateway.MainKeyType(req.KeyType) {
			return gatewayResp.AiKeyView{}, errors.New("主 Key 不能关联场景")
		}
		if err := ensureScenarioUsable(ctx, global.OPS_DB, *req.ScenarioId); err != nil {
			return gatewayResp.AiKeyView{}, err
		}
		scenarioId = *req.ScenarioId
	}

	k := gateway.AiKey{
		Name:            req.Name,
		Description:     req.Description,
		KeyType:         req.KeyType,
		OwnerType:       req.OwnerType,
		OwnerId:         req.OwnerId,
		LitellmKeyAlias: alias,
		ScenarioId:      scenarioId,
		Models:          marshalJSONStringSlice(models),
		ModelBudgets:    marshalJSONMap(req.ModelBudgets),
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
		ExpiresAt:       req.ExpiresAt,
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

	// 名称变更时查重(防 LiteLLM key_alias 撞车)，排除自身；软删由 gorm 自动过滤。
	if req.Name != "" && req.Name != k.Name {
		var nameCnt int64
		if err := global.OPS_DB.WithContext(ctx).Model(&gateway.AiKey{}).
			Where("key_type = ? AND owner_type = ? AND owner_id = ? AND name = ? AND ai_key_id <> ?",
				k.KeyType, k.OwnerType, k.OwnerId, req.Name, req.AiKeyId).
			Count(&nameCnt).Error; err != nil {
			return gatewayResp.AiKeyView{}, err
		}
		if nameCnt > 0 {
			return gatewayResp.AiKeyView{}, errors.New("该归属下已存在同名密钥")
		}
	}

	models := req.Models
	if models == nil {
		models = jsonToSlice(k.Models)
	}

	// 场景归属(nil=清空为无场景；非空须为启用场景；主 Key 恒清空)
	if req.ScenarioId != nil && *req.ScenarioId != 0 {
		if gateway.MainKeyType(k.KeyType) {
			return gatewayResp.AiKeyView{}, errors.New("主 Key 不能关联场景")
		}
		if err := ensureScenarioUsable(ctx, global.OPS_DB, *req.ScenarioId); err != nil {
			return gatewayResp.AiKeyView{}, err
		}
	}
	scenarioId := int64(0)
	if req.ScenarioId != nil {
		scenarioId = *req.ScenarioId
	}

	updates := map[string]any{
		"name":            req.Name,
		"description":     req.Description,
		"scenario_id":     scenarioId,
		"models":          marshalJSONStringSlice(models),
		"model_budgets":   marshalJSONMap(req.ModelBudgets),
		"budget_duration": normalizeBudgetDuration(req.BudgetDuration),
		"rate_limit_mode": normalizeRateLimitMode(req.RateLimitMode),
		"tpm_limit":       req.TpmLimit,
		"rpm_limit":       req.RpmLimit,
		"model_limits":    marshalJSONMap(req.ModelLimits),
		"expires_at":      req.ExpiresAt, // 过期时间覆盖式更新(nil=改回永不过期)
		"update_by":       updateBy,
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
	k.ExpiresAt = req.ExpiresAt
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

// RotateAiKey 轮换密钥：LiteLLM 删旧 Key → 复用 syncKeyToLitellm(create=true) 建新 Key 并
// 原地更新同一行的 litellm_key_id/key_value(加密)/key_prefix。AiKeyId 与 key_alias 均不变，
// 历史用量归因(alias 匹配)保持连续——优于手工删旧建新(换行断归因、场景引用丢失)。
// 旧 Key 在 LiteLLM 删除成功瞬间即失效(轮换的安全语义：宁可短暂不可用，不留旧值)；
// 建新失败则本地行指向已删 Key(用户 Key 失效)，返回错误提示重试。管理员视角只回 KeyPrefix，
// 新明文仅 owner 本人经 identity/my 查看。
func (s *AiKeyService) RotateAiKey(ctx context.Context, id, updateBy int64) (gatewayResp.AiKeyView, error) {
	var k gateway.AiKey
	if err := global.OPS_DB.WithContext(ctx).Where("ai_key_id = ?", id).First(&k).Error; err != nil {
		return gatewayResp.AiKeyView{}, err
	}
	cli := litellm.Default()
	if cli == nil {
		return gatewayResp.AiKeyView{}, errors.New("单机模式(LiteLLM 未配置)不支持轮换")
	}
	if k.LitellmKeyId == "" {
		return gatewayResp.AiKeyView{}, errors.New("密钥未同步 LiteLLM，无法轮换")
	}
	// 先删旧(失败中止，本地不动)
	if err := cli.DeleteKey(ctx, k.LitellmKeyId); err != nil {
		return gatewayResp.AiKeyView{}, fmt.Errorf("轮换删除旧 Key 失败，已中止(本地未动): %w", err)
	}
	// 建新并原地更新(复用创建管线：加密落库/prefix/anthropic 扩展/max_budget/expires_at 语义全保留)
	if err := syncKeyToLitellm(ctx, cli, global.OPS_DB.WithContext(ctx), &k, true); err != nil {
		return gatewayResp.AiKeyView{}, fmt.Errorf("轮换生成新 Key 失败(旧 Key 已失效，请重试): %w", err)
	}
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.AiKey{}).Where("ai_key_id = ?", id).
		Update("update_by", updateBy).Error; err != nil {
		return gatewayResp.AiKeyView{}, err
	}
	var fresh gateway.AiKey
	if err := global.OPS_DB.WithContext(ctx).Where("ai_key_id = ?", id).First(&fresh).Error; err != nil {
		return gatewayResp.AiKeyView{}, err
	}
	return toAiKeyView(fresh), nil
}

// BatchCreateMainKeys 批量开通个人主 Key(管理员创建制的效率件)：目标 = deptId 部门下全部
// 用户 ∪ userIds；已有 personal_main 跳过，停用用户标记失败(建了也会被级联停用)；单用户
// 创建失败不中断(每用户独立事务)。部分成功语义：结果经 data 标记返回(created/skipped/failed)。
func (s *AiKeyService) BatchCreateMainKeys(ctx context.Context, req gatewayReq.AiKeyBatchCreateParams, createBy int64) (gatewayResp.BatchCreateMainKeysResult, error) {
	users, err := s.resolveBatchTargets(ctx, req)
	if err != nil {
		return gatewayResp.BatchCreateMainKeysResult{}, err
	}
	result := gatewayResp.BatchCreateMainKeysResult{Total: len(users), Failed: []gatewayResp.BatchCreateMainKeysFailure{}}

	// 已有主 Key 的目标(一次 IN 查询去重集)
	ids := make([]int64, 0, len(users))
	for i := range users {
		ids = append(ids, users[i].UserId)
	}
	var existing []int64
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.AiKey{}).
		Where("key_type = ? AND owner_type = ? AND owner_id IN ?", gateway.KeyTypePersonalMain, gateway.OwnerTypeUser, ids).
		Pluck("owner_id", &existing).Error; err != nil {
		return gatewayResp.BatchCreateMainKeysResult{}, err
	}
	existingSet := map[int64]bool{}
	for _, id := range existing {
		existingSet[id] = true
	}

	toCreate, skipped, failed := classifyBatchTargets(users, existingSet)
	result.Skipped = skipped
	for i := range toCreate {
		u := toCreate[i]
		isActive := true
		params := gatewayReq.AiKeyOperateParams{
			KeyType:     gateway.KeyTypePersonalMain,
			OwnerType:   gateway.OwnerTypeUser,
			OwnerId:     u.UserId,
			Name:        "main",
			Description: "个人主 Key(批量开通)",
			IsActive:    &isActive,
		}
		if _, err := s.CreateSceneKey(ctx, params, createBy); err != nil {
			failed = append(failed, gatewayResp.BatchCreateMainKeysFailure{
				UserId: u.UserId, Name: u.NickName, Reason: err.Error(),
			})
			continue
		}
		result.Created++
	}
	result.Failed = failed
	return result, nil
}

// classifyBatchTargets 批量开通目标分类(纯函数,可单测)：停用用户→失败(建了也会被
// 用户级联停用，无意义)、已有主 Key→跳过、其余→待创建。判定顺序：停用优先于已存在
// (状态异常先报，避免"看似跳过实则账号不可用"的误判)。
func classifyBatchTargets(users []system.SysUser, existingSet map[int64]bool) (
	toCreate []system.SysUser, skipped int, failed []gatewayResp.BatchCreateMainKeysFailure) {
	// failed 初始化为空切片：nil 切片 JSON 序列化为 null，违反「空数组=全部成功」契约，
	// 前端 data.failed.length 会 TypeError 导致批量弹窗渲染崩溃冻结。
	failed = make([]gatewayResp.BatchCreateMainKeysFailure, 0)
	for i := range users {
		u := users[i]
		if u.Status != "0" {
			failed = append(failed, gatewayResp.BatchCreateMainKeysFailure{
				UserId: u.UserId, Name: u.NickName, Reason: "用户已停用",
			})
			continue
		}
		if existingSet[u.UserId] {
			skipped++
			continue
		}
		toCreate = append(toCreate, u)
		existingSet[u.UserId] = true // 防御同批重复ID
	}
	return toCreate, skipped, failed
}

// resolveBatchTargets 解析批量开通目标用户：deptId 优先(部门下全部)，userIds 补充，
// 两者并集按行过滤天然去重。仅取 id/nick_name/status 最小列集。
func (s *AiKeyService) resolveBatchTargets(ctx context.Context, req gatewayReq.AiKeyBatchCreateParams) ([]system.SysUser, error) {
	if req.DeptId == nil && len(req.UserIds) == 0 {
		return nil, errors.New("请指定目标用户或部门")
	}
	q := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{})
	switch {
	case req.DeptId != nil && len(req.UserIds) > 0:
		q = q.Where("dept_id = ? OR id IN ?", *req.DeptId, req.UserIds)
	case req.DeptId != nil:
		q = q.Where("dept_id = ?", *req.DeptId)
	default:
		q = q.Where("id IN ?", req.UserIds)
	}
	var users []system.SysUser
	if err := q.Select("id, nick_name, status").Order("id ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// SyncUserKeysActive 用户启停/删除级联：同步其名下全部 AiKey(主+场景，软删行除外)的启停状态。
// 用户禁用后 Key 若保持启用，明文 Key 仍可直连 LiteLLM 网关(仅登录被拦)，必须联动卡死；
// LiteLLM 侧复用 is_active=false → max_budget=0 停用语义。启用时全量恢复(对齐 AIHelms
// sync_user_keys_active；管理员此前手动停用的 Key 会被一并恢复，属已知取舍)。单个失败仅
// 告警不中断，返回告警列表供调用方提示。
func (s *AiKeyService) SyncUserKeysActive(ctx context.Context, userId, updateBy int64, active bool) []string {
	var keys []gateway.AiKey
	if err := global.OPS_DB.WithContext(ctx).
		Where("owner_type = ? AND owner_id = ?", gateway.OwnerTypeUser, userId).
		Find(&keys).Error; err != nil {
		return []string{err.Error()}
	}
	cli := litellm.Default()
	var warnings []string
	for i := range keys {
		keys[i].IsActive = active
		if err := global.OPS_DB.WithContext(ctx).Model(&gateway.AiKey{}).
			Where("ai_key_id = ?", keys[i].AiKeyId).
			Updates(map[string]any{"is_active": active, "update_by": updateBy}).Error; err != nil {
			warnings = append(warnings, fmt.Sprintf("密钥 %d: %v", keys[i].AiKeyId, err))
			continue
		}
		if cli == nil || keys[i].LitellmKeyId == "" {
			continue
		}
		if err := syncKeyToLitellm(ctx, cli, global.OPS_DB.WithContext(ctx), &keys[i], false); err != nil {
			warnings = append(warnings, fmt.Sprintf("密钥 %d 同步 LiteLLM: %v", keys[i].AiKeyId, err))
		}
	}
	state := "启用"
	if !active {
		state = "停用"
	}
	for _, w := range warnings {
		logger.WithCtx(ctx).Mod("gateway").Warn(fmt.Sprintf("用户 %d 级联%s密钥失败: %s", userId, state, w))
	}
	return warnings
}

// ----------------------------------------------------------------------------
// 包级共享：同步管线与主 Key 自愈
// ----------------------------------------------------------------------------

// loadMainKey 取个人主 Key(管理员创建制，此处不创建)：无则返回 nil；
// 有则补缺失的公开模型(幂等，发布配置变化后下次访问 home 自愈)。
func loadMainKey(ctx context.Context, userId int64) (*gateway.AiKey, error) {
	var k gateway.AiKey
	err := global.OPS_DB.WithContext(ctx).
		Where("key_type = ? AND owner_type = ? AND owner_id = ?", gateway.KeyTypePersonalMain, gateway.OwnerTypeUser, userId).
		First(&k).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// 已有主 Key：补缺失的可见免审批模型(幂等自愈；定向发布对可见范围内的个人主 Key 同样生效)
	publicKeys := visibleModelKeys(global.OPS_DB.WithContext(ctx), userId, userDeptIdOf(global.OPS_DB.WithContext(ctx), userId))
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

// mainKeyScopeOf 按发布可见档构造自动授权的主 Key 目标集合查询(scope 加在密钥查询上)：
// all=全部主 Key；selected=可见部门成员(sys_users 主部门∪多部门)的个人主 Key+可见部门
// 的部门主 Key；user=指定用户的个人主 Key。与 visibleModelScope 的可见口径对称。
func mainKeyScopeOf(visibility string, deptIds, userIds []int64) func(*gorm.DB) *gorm.DB {
	switch visibility {
	case gateway.VisibilityTypeSelected:
		return func(q *gorm.DB) *gorm.DB {
			return q.Where(
				`(key_type = ? AND owner_type = ? AND owner_id IN (
					SELECT u.id FROM sys_users u WHERE u.deleted_at IS NULL AND (
						u.dept_id IN ?
						OR u.id IN (SELECT ud.sys_user_id FROM sys_user_departments ud WHERE ud.sys_department_id IN ?))))
				OR (key_type = ? AND owner_type = ? AND owner_id IN ?)`,
				gateway.KeyTypePersonalMain, gateway.OwnerTypeUser, deptIds, deptIds,
				gateway.KeyTypeDeptMain, gateway.OwnerTypeDept, deptIds)
		}
	case gateway.VisibilityTypeUser:
		return func(q *gorm.DB) *gorm.DB {
			return q.Where("key_type = ? AND owner_type = ? AND owner_id IN ?",
				gateway.KeyTypePersonalMain, gateway.OwnerTypeUser, userIds)
		}
	default: // all
		return func(q *gorm.DB) *gorm.DB {
			return q.Where("key_type IN ?", []string{gateway.KeyTypePersonalMain, gateway.KeyTypeDeptMain})
		}
	}
}

// syncModelToMainKeys 发布免审批模型时向目标活跃主 Key 集合追加 modelKey 并同步
// LiteLLM(事务内，单个失败 warning 继续)。目标集合由 mainKeyScopeOf 按发布可见档构造
// (all=全部主 Key/selected=可见部门成员+部门主Key/user=指定用户个人主Key)。
func syncModelToMainKeys(ctx context.Context, tx *gorm.DB, modelKey string, scope func(*gorm.DB) *gorm.DB) []string {
	var keys []gateway.AiKey
	if err := scope(tx).Where("is_active = ?", true).Find(&keys).Error; err != nil {
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

// cascadeRenameKeyModels 模型 modelKey 改名级联：改写引用旧 modelKey 的密钥
// models/model_budgets/model_limits 三个 JSONB(oldKey→newKey,值保持)并同步 LiteLLM。
// 不做则改名后密钥授权/按模型预算/限流全部随旧名漂移(用户调新名被网关拒)。
// 与部署侧级联重建同口径(尽力而为：单个失败记 warning 继续)；AiKey 是小表,
// 全量拉取后内存过滤,不建 JSON 查询条件(跨库方言)。DB 存原始 modelKey,
// (Anthropic) 变体仅在 syncKeyToLitellm 下发时扩展,无需处理。
func cascadeRenameKeyModels(ctx context.Context, db *gorm.DB, oldKey, newKey string) []string {
	var keys []gateway.AiKey
	if err := db.Find(&keys).Error; err != nil {
		return []string{err.Error()}
	}
	cli := litellm.Default()
	var warnings []string
	for i := range keys {
		models, budgets, limits, changed := renameKeyReferences(
			jsonToSlice(keys[i].Models), jsonToMap(keys[i].ModelBudgets), jsonToMap(keys[i].ModelLimits), oldKey, newKey)
		if !changed {
			continue
		}
		keys[i].Models = marshalJSONStringSlice(models)
		updates := map[string]any{"models": keys[i].Models}
		if budgets != nil {
			keys[i].ModelBudgets = marshalJSONMap(budgets)
			updates["model_budgets"] = keys[i].ModelBudgets
		}
		if limits != nil {
			keys[i].ModelLimits = marshalJSONMap(limits)
			updates["model_limits"] = keys[i].ModelLimits
		}
		if err := db.Model(&gateway.AiKey{}).Where("ai_key_id = ?", keys[i].AiKeyId).
			Updates(updates).Error; err != nil {
			warnings = append(warnings, fmt.Sprintf("密钥 %d: %v", keys[i].AiKeyId, err))
			continue
		}
		if cli == nil || keys[i].LitellmKeyId == "" {
			continue
		}
		if err := syncKeyToLitellm(ctx, cli, db, &keys[i], false); err != nil {
			warnings = append(warnings, fmt.Sprintf("密钥 %d 同步 LiteLLM: %v", keys[i].AiKeyId, err))
		}
	}
	for _, w := range warnings {
		logger.WithCtx(ctx).Mod("gateway").Warn(fmt.Sprintf("模型改名 %q→%q 级联密钥失败: %s", oldKey, newKey, w))
	}
	return warnings
}

// renameKeyReferences 改名三处引用的纯函数(可单测)：models 列表元素替换、
// budgets/limits map 键替换(值不动)。返回改写后的值与是否有任何变更。
func renameKeyReferences(models []string, budgets, limits map[string]any, oldKey, newKey string) ([]string, map[string]any, map[string]any, bool) {
	changed := false
	renamed := make([]string, len(models))
	for i, mk := range models {
		if mk == oldKey {
			renamed[i] = newKey
			changed = true
			continue
		}
		renamed[i] = mk
	}
	renameMap := func(m map[string]any) (map[string]any, bool) {
		if _, ok := m[oldKey]; !ok {
			return m, false
		}
		nm := make(map[string]any, len(m))
		for k, v := range m {
			if k == oldKey {
				nm[newKey] = v
			} else {
				nm[k] = v
			}
		}
		return nm, true
	}
	nb, bChanged := renameMap(budgets)
	nl, lChanged := renameMap(limits)
	return renamed, nb, nl, changed || bChanged || lChanged
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
		"aiKeyId": k.AiKeyId,
		"keyType": k.KeyType,
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
			ExpiresAt: k.ExpiresAt,
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
				"litellm_key_id": k.LitellmKeyId,
				"key_value":      k.KeyValue,
				"key_prefix":     k.KeyPrefix,
			}).Error
	}

	// Update
	req := litellm.KeyUpdateReq{
		Models:     expandedModels,
		MaxBudget:  maxBudget,
		Metadata:   metadata,
		ExpiresAt:  k.ExpiresAt,
		SyncBudget: syncBudget,
		SyncExpiry: true, // 过期时间始终与 DB 行同步(含 nil→null 清空)
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

// userDeptIdOf 用户主部门ID(查不到/无部门给 0)。
func userDeptIdOf(db *gorm.DB, userId int64) int64 {
	var deptId int64
	db.Model(&system.SysUser{}).Select("dept_id").Where("id = ?", userId).Scan(&deptId)
	return deptId
}

// visibleModelKeys 对指定主 Key owner 可见的免审批模型 modelKey 列表(主 Key 自愈差集
// 与新建主 Key 默认授权的数据源)：个人 owner 传 (userId,主部门) 三档全生效；部门 owner
// 传 (0,deptId) 仅 all/selected 档生效(user 档对部门无意义,userId=0 不命中投影)。
// 定向发布(selected/user 档)对可见范围内的主 Key 同样生效，与发布时自动授权口径一致。
func visibleModelKeys(db *gorm.DB, userId, deptId int64) []string {
	var keys []string
	visibleModelScope(
		db.Model(&gateway.Model{}).
			Where("is_active = ? AND is_published = ? AND requires_approval = ?", true, true, false).
			Where("model_key <> ''"),
		userId, deptId,
	).Pluck("model_key", &keys)
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
