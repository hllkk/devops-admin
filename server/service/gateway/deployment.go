package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	"github.com/hllkk/devops-admin/server/utils/litellm"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// DeploymentService 模型部署管理(对齐前端 /gateway/model/deployment/* 资源)。
// 同步遵循「平台 DB 是唯一事实源，LiteLLM 是投影」：部署 CRUD 事务内联推送，
// 推送失败整体回滚；禁用=路由名加 __disabled__ 后缀出池+model_info.active 双写，
// litellm_model_id 永不重置(成本/日志归因锚点)；LiteLLM DeleteModel 全程不用(删除=禁用留痕)。
type DeploymentService struct{}

// ----------------------------------------------------------------------------
// 包级共享管线（ModelService/凭证级联/部署服务复用）
// ----------------------------------------------------------------------------

// formatOf 凭证协议格式（credential_info.format）。nil 凭证/未设置返回空串(调用方默认 openai)。
func formatOf(cred *gateway.Credential) string {
	if cred == nil {
		return ""
	}
	if s, ok := credentialInfoToMap(cred.CredentialInfo)["format"].(string); ok {
		return s
	}
	return ""
}

// routableOf 部署可路由判定：部署启用 且 (未绑凭证 或 凭证启用)。
func routableOf(depIsActive bool, cred *gateway.Credential) bool {
	return depIsActive && (cred == nil || cred.IsActive)
}

// resolvePrefix 查差异表 (provider_type, format, category) → (prefix, needs_v1)。
// 未命中兜底：仅 format==anthropic 且 category==chat 返回 ("anthropic", false)（协议隔离必须分组）。
func resolvePrefix(db *gorm.DB, providerType, format, category string) (string, bool) {
	if providerType == "" {
		return "", false
	}
	var row gateway.ProviderPrefix
	err := db.Where("provider_type = ? AND format = ? AND category = ?", providerType, format, category).
		First(&row).Error
	if err == nil {
		return row.Prefix, row.NeedsV1
	}
	if format == "anthropic" && category == gateway.ModelCategoryChat {
		return "anthropic", false
	}
	return "", false
}

// ensureCredentialSynced 部署操作前的凭证懒同步兜底：未同步(如单机模式建的)则补推并置位。
// 凭证 CRUD 本身是强一致推送(失败回滚)，此处只兜 synced=false 的漏网；不独立 commit。
func ensureCredentialSynced(ctx context.Context, tx *gorm.DB, cli *litellm.Client, cred *gateway.Credential) error {
	if cli == nil || cred == nil || cred.LitellmSynced {
		return nil
	}
	values, err := decryptCredentialValues(cred.CredentialValues)
	if err != nil {
		return fmt.Errorf("凭证 %q 值解密失败: %w", cred.CredentialName, err)
	}
	info := credentialInfoToMap(cred.CredentialInfo)
	payload := BuildLitellmCredentialValues(values, info, providerTypeOfTx(tx, cred.ProviderId))
	if err := pushCredential(ctx, cli, cred.CredentialName, payload, info); err != nil {
		return fmt.Errorf("凭证 %q 同步 LiteLLM 失败(请检查凭证配置): %w", cred.CredentialName, err)
	}
	return tx.Model(&gateway.Credential{}).Where("credential_id = ?", cred.CredentialId).
		Update("litellm_synced", true).Error
}

// providerTypeOfTx 事务内查供应商类型(投影构建用)；未关联/查不到返回 ""。
func providerTypeOfTx(db *gorm.DB, providerId int64) string {
	if providerId == 0 {
		return ""
	}
	var p gateway.Provider
	if err := db.Select("provider_type").Where("provider_id = ?", providerId).First(&p).Error; err != nil {
		return ""
	}
	return p.ProviderType
}

// buildDeploymentParams 部署投影的 DB 侧事实构建(人民币口径，写回平台 DB)：
// ①绑定凭证: 剔 inline api_key → 写 litellm_credential_name → api_base 归凭证
// ④定价镜像: params 四键 → model_info (input_cost 等)
// 管线②③(前缀解析/前缀化 model/补 /v1)是 LiteLLM 投影层派生值，不在此构建、不落库——
// pushDeployment 推送时经 resolveDeploymentPrefix + ApplyPrefixProjection 临时投影，
// DB 的 litellm_params.model 始终存用户填的原始厂商模型名。
func buildDeploymentParams(dep *gateway.ModelDeployment, cred *gateway.Credential) (params, modelInfo map[string]any) {
	params = jsonToMap(dep.LitellmParams)
	modelInfo = jsonToMap(dep.ModelInfo)
	if cred != nil {
		values, err := decryptCredentialValues(cred.CredentialValues)
		if err == nil {
			params = ApplyCredentialToParams(params, cred.CredentialName, values)
		} else {
			// 解密失败仍保留引用关系(凭证名)，仅 api_base 不覆盖——推送侧凭证同步会先失败暴露问题
			params = ApplyCredentialToParams(params, cred.CredentialName, map[string]any{})
		}
	}
	modelInfo = MergeCostsToModelInfo(modelInfo, params)
	return params, modelInfo
}

// resolveDeploymentPrefix 投影层前缀解析(封装 providerTypeOfTx + resolvePrefix)：cred 为 nil
// 或未关联供应商 → 返 ("", false)；仅 pushDeployment 投影用，不写回 DB。
func resolveDeploymentPrefix(db *gorm.DB, cred *gateway.Credential, format, category string) (prefix string, needsV1 bool) {
	if cred == nil {
		return "", false
	}
	providerType := providerTypeOfTx(db, cred.ProviderId)
	return resolvePrefix(db, providerType, format, category)
}

// pushDeployment 推送部署投影到 LiteLLM：路由名两态 + 投影层前缀化 + active 双写 + USD/token 换算副本。
// litellm_model_id 为空 → AddModel 并写回(dep.LitellmModelId)；否则 UpdateModel(改名+全量)。
// prefix/needsV1 来自 resolveDeploymentPrefix，经 ApplyPrefixProjection 临时投影(不写回 DB)。
func pushDeployment(ctx context.Context, cli *litellm.Client, dep *gateway.ModelDeployment, modelKey string, routable bool, prefix string, needsV1 bool, params, modelInfo map[string]any) error {
	routeName := BuildModelRouteName(modelKey, routable)
	pushParams := ConvertCostsForLitellm(ApplyPrefixProjection(params, prefix, needsV1), global.OPS_CONFIG.Litellm.UsdToCnyRate)
	pushInfo := withActive(modelInfo, routable)
	if dep.LitellmModelId == "" {
		resp, err := cli.AddModel(ctx, routeName, pushParams, pushInfo)
		if err != nil {
			return fmt.Errorf("部署同步 LiteLLM 失败: %w", err)
		}
		if resp.ModelInfo.ID != "" {
			dep.LitellmModelId = resp.ModelInfo.ID
		}
		return nil
	}
	return cli.UpdateModel(ctx, dep.LitellmModelId, routeName, pushParams, pushInfo)
}

// withActive 拷贝并置 active 标志(DB 以 is_active 列为唯一事实源，active 只写投影)。
func withActive(modelInfo map[string]any, active bool) map[string]any {
	out := make(map[string]any, len(modelInfo)+1)
	for k, v := range modelInfo {
		out[k] = v
	}
	out["active"] = active
	return out
}

// jsonToMap datatypes.JSON → map；空/解析失败给空 map。
func jsonToMap(raw datatypes.JSON) map[string]any {
	out := map[string]any{}
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}

// marshalJSON map → datatypes.JSON；nil 给 {}。
func marshalJSON(m map[string]any) datatypes.JSON {
	if m == nil {
		return datatypes.JSON([]byte("{}"))
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return datatypes.JSON([]byte("{}"))
	}
	return datatypes.JSON(raw)
}

// ----------------------------------------------------------------------------
// 部署 CRUD
// ----------------------------------------------------------------------------

// GetDeploymentList 分页查部署列表(带模型/凭证上下文与掩码参数)。
func (s *DeploymentService) GetDeploymentList(ctx context.Context, q gatewayReq.DeploymentSearch) (list []gatewayResp.DeploymentView, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&gateway.ModelDeployment{})
	if q.ModelId != 0 {
		db = db.Where("model_id = ?", q.ModelId)
	}
	if q.CredentialId != 0 {
		db = db.Where("credential_id = ?", q.CredentialId)
	}
	if q.Keyword != "" {
		db = db.Where("deploy_name LIKE ?", "%"+q.Keyword+"%")
	}
	if q.IsActive != nil {
		db = db.Where("is_active = ?", *q.IsActive)
	}
	var rows []gateway.ModelDeployment
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("deployment_id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	} else {
		err = db.Count(&total).Order("deployment_id DESC").Find(&rows).Error
	}
	if err != nil {
		return nil, 0, err
	}
	list = make([]gatewayResp.DeploymentView, 0, len(rows))
	for i := range rows {
		list = append(list, s.toView(ctx, rows[i]))
	}
	return list, total, nil
}

// CreateDeployment 新增部署：事务内落库 → 凭证懒同步兜底 → 投影构建 → routable 才推送
// (AddModel 写回 litellm_model_id) → 处理后人民币口径写回 DB。推送失败整体回滚。
func (s *DeploymentService) CreateDeployment(ctx context.Context, req gatewayReq.DeploymentOperateParams, createBy int64) (gatewayResp.DeploymentView, error) {
	if req.ModelId == 0 {
		return gatewayResp.DeploymentView{}, errors.New("关联模型不能为空")
	}
	var model gateway.Model
	if err := global.OPS_DB.WithContext(ctx).Where("model_id = ?", req.ModelId).First(&model).Error; err != nil {
		return gatewayResp.DeploymentView{}, errors.New("关联模型不存在")
	}
	if raw, ok := req.LitellmParams["model"].(string); !ok || raw == "" {
		return gatewayResp.DeploymentView{}, errors.New("路由参数必须包含 model 键(上游模型名)")
	}
	var cred *gateway.Credential
	if req.CredentialId != 0 {
		var c gateway.Credential
		if err := global.OPS_DB.WithContext(ctx).Where("credential_id = ?", req.CredentialId).First(&c).Error; err != nil {
			return gatewayResp.DeploymentView{}, errors.New("关联凭证不存在")
		}
		cred = &c
	}
	// 模型路由名：model_key 未设置时拒绝(路由组名锚点)
	if model.ModelKey == "" {
		return gatewayResp.DeploymentView{}, errors.New("请先为模型设置路由名(ModelKey)再创建部署")
	}

	dep := gateway.ModelDeployment{
		ModelId:          req.ModelId,
		CredentialId:     req.CredentialId,
		LitellmParams:    marshalJSON(req.LitellmParams),
		ModelInfo:        marshalJSON(req.ModelInfo),
		DeployName:       req.DeployName,
		BillingType:      normalizeBillingType(req.BillingType),
		CostPerCall:      req.CostPerCall,
		MonthlyCallQuota: req.MonthlyCallQuota,
		IsActive:         req.IsActive == nil || *req.IsActive,
	}
	dep.CreateBy = createBy
	dep.UpdateBy = createBy

	cli := litellm.Default()
	format := "openai"
	if cred != nil {
		format = formatOf(cred)
		if format == "" {
			format = "openai"
		}
	}
	routable := routableOf(dep.IsActive, cred)
	err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&dep).Error; err != nil {
			return err
		}
		if cred != nil {
			if err := ensureCredentialSynced(ctx, tx, cli, cred); err != nil {
				return err
			}
		}
		params, modelInfo := buildDeploymentParams(&dep, cred)
		if cli != nil && routable {
			prefix, needsV1 := resolveDeploymentPrefix(tx, cred, format, model.Category)
			if err := pushDeployment(ctx, cli, &dep, model.ModelKey, routable, prefix, needsV1, params, modelInfo); err != nil {
				return err
			}
		}
		return tx.Model(&gateway.ModelDeployment{}).Where("deployment_id = ?", dep.DeploymentId).
			Updates(map[string]any{
				"litellm_params":   marshalJSON(params),
				"model_info":       marshalJSON(modelInfo),
				"litellm_model_id": dep.LitellmModelId,
				"update_by":        createBy,
			}).Error
	})
	if err != nil {
		return gatewayResp.DeploymentView{}, err
	}
	return s.toView(ctx, dep), nil
}

// UpdateDeployment 修改部署：掩码还原合并参数 → 基础列覆盖 → 投影重建推送(改名+active 双写)。
// 从未同步(单机建的)且当前 active → 内部补 AddModel；不允许改 model 归属(换模型=删了重建)。
func (s *DeploymentService) UpdateDeployment(ctx context.Context, req gatewayReq.DeploymentOperateParams, updateBy int64) (gatewayResp.DeploymentView, error) {
	if req.DeploymentId == 0 {
		return gatewayResp.DeploymentView{}, errors.New("部署ID不能为空")
	}
	if req.ModelId != 0 {
		var dep gateway.ModelDeployment
		_ = global.OPS_DB.WithContext(ctx).Where("deployment_id = ?", req.DeploymentId).First(&dep).Error
		if dep.DeploymentId != 0 && req.ModelId != dep.ModelId {
			return gatewayResp.DeploymentView{}, errors.New("不允许修改部署的模型归属(请删除后新建)")
		}
	}
	var dep gateway.ModelDeployment
	if err := global.OPS_DB.WithContext(ctx).Where("deployment_id = ?", req.DeploymentId).First(&dep).Error; err != nil {
		return gatewayResp.DeploymentView{}, err
	}
	var model gateway.Model
	if err := global.OPS_DB.WithContext(ctx).Where("model_id = ?", dep.ModelId).First(&model).Error; err != nil {
		return gatewayResp.DeploymentView{}, errors.New("关联模型不存在")
	}
	var cred *gateway.Credential
	if req.CredentialId != 0 {
		var c gateway.Credential
		if err := global.OPS_DB.WithContext(ctx).Where("credential_id = ?", req.CredentialId).First(&c).Error; err != nil {
			return gatewayResp.DeploymentView{}, errors.New("关联凭证不存在")
		}
		cred = &c
	} else if dep.CredentialId != 0 {
		// 未传凭证 = 维持原绑定
		var c gateway.Credential
		if err := global.OPS_DB.WithContext(ctx).Where("credential_id = ?", dep.CredentialId).First(&c).Error; err == nil {
			cred = &c
		}
	}

	newParams := jsonToMap(dep.LitellmParams)
	if req.LitellmParams != nil {
		if raw, ok := req.LitellmParams["model"].(string); !ok || raw == "" {
			return gatewayResp.DeploymentView{}, errors.New("路由参数必须包含 model 键(上游模型名)")
		}
		newParams = UnmaskIncomingParams(newParams, req.LitellmParams)
	}
	newModelInfo := jsonToMap(dep.ModelInfo)
	if req.ModelInfo != nil {
		newModelInfo = req.ModelInfo
	}

	updates := map[string]any{
		"deploy_name":        req.DeployName,
		"billing_type":       normalizeBillingType(req.BillingType),
		"credential_id":      req.CredentialId,
		"litellm_params":     marshalJSON(newParams),
		"model_info":         marshalJSON(newModelInfo),
		"cost_per_call":      req.CostPerCall,
		"monthly_call_quota": req.MonthlyCallQuota,
		"update_by":          updateBy,
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
		dep.IsActive = *req.IsActive
	}
	dep.CredentialId = req.CredentialId
	dep.LitellmParams = marshalJSON(newParams)
	dep.ModelInfo = marshalJSON(newModelInfo)

	cli := litellm.Default()
	format := "openai"
	if cred != nil {
		format = formatOf(cred)
		if format == "" {
			format = "openai"
		}
	}
	routable := routableOf(dep.IsActive, cred)
	err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&gateway.ModelDeployment{}).Where("deployment_id = ?", req.DeploymentId).Updates(updates).Error; err != nil {
			return err
		}
		if cred != nil {
			if err := ensureCredentialSynced(ctx, tx, cli, cred); err != nil {
				return err
			}
		}
		params, modelInfo := buildDeploymentParams(&dep, cred)
		if cli != nil && (routable || dep.LitellmModelId != "") {
			prefix, needsV1 := resolveDeploymentPrefix(tx, cred, format, model.Category)
			if err := pushDeployment(ctx, cli, &dep, model.ModelKey, routable, prefix, needsV1, params, modelInfo); err != nil {
				return err
			}
		}
		return tx.Model(&gateway.ModelDeployment{}).Where("deployment_id = ?", req.DeploymentId).
			Updates(map[string]any{
				"litellm_params":   marshalJSON(params),
				"model_info":       marshalJSON(modelInfo),
				"litellm_model_id": dep.LitellmModelId,
				"update_by":        updateBy,
			}).Error
	})
	if err != nil {
		return gatewayResp.DeploymentView{}, err
	}
	var fresh gateway.ModelDeployment
	if err := global.OPS_DB.WithContext(ctx).Where("deployment_id = ?", req.DeploymentId).First(&fresh).Error; err != nil {
		return gatewayResp.DeploymentView{}, err
	}
	return s.toView(ctx, fresh), nil
}

// DeleteDeployments 批量删除部署(本地软删)：先在 LiteLLM 侧禁用(active=false 不改名，
// 留痕保归因)，禁用失败则本地不动；LiteLLM DeleteModel 全程不用。
func (s *DeploymentService) DeleteDeployments(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	var rows []gateway.ModelDeployment
	if err := global.OPS_DB.WithContext(ctx).Where("deployment_id IN ?", ids).Find(&rows).Error; err != nil {
		return err
	}
	cli := litellm.Default()
	for i := range rows {
		if rows[i].LitellmModelId == "" || cli == nil {
			continue
		}
		if err := cli.UpdateModel(ctx, rows[i].LitellmModelId, "", nil, withActive(jsonToMap(rows[i].ModelInfo), false)); err != nil {
			return fmt.Errorf("部署 %q 在 LiteLLM 侧禁用失败，已中止(本地未动): %w", rows[i].DeployName, err)
		}
	}
	return global.OPS_DB.WithContext(ctx).Where("deployment_id IN ?", ids).Delete(&gateway.ModelDeployment{}).Error
}

// asrProbeAudio 语音识别连通性探测用的极小静音 wav(data URL 形态，0.1s 8kHz 单声道 16bit)。
const asrProbeAudio = "data:audio/wav;base64,UklGRmQGAABXQVZFZm10IBAAAAABAAEAQB8AAIA+AAACABAAZGF0YUAGAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

// speechRecognitionKeywords 语音识别类能力标签关键词(小写包含匹配，覆盖中英文常见写法)。
// 语音识别模型不可用纯文本 ping 探测：上游业务校验要求消息含 input_audio part。
var speechRecognitionKeywords = []string{"语音识别", "语音转", "语音听", "语音输入", "asr", "stt", "transcri", "speech"}

// isSpeechRecognitionModel 能力标签含语音识别类关键词(标签为用户自由输入，按启发式识别)。
func isSpeechRecognitionModel(capabilitiesJSON datatypes.JSON) bool {
	if len(capabilitiesJSON) == 0 {
		return false
	}
	var caps []string
	if err := json.Unmarshal(capabilitiesJSON, &caps); err != nil {
		return false
	}
	for _, capability := range caps {
		s := strings.ToLower(capability)
		for _, kw := range speechRecognitionKeywords {
			if strings.Contains(s, kw) {
				return true
			}
		}
	}
	return false
}

// TestDeployment 部署连通性测试(管理员视角)：master key 经 LiteLLM 数据面按 category 分流，
// 测 routable 形态路由名；错误分类+技术详情脱敏。
func (s *DeploymentService) TestDeployment(ctx context.Context, req gatewayReq.DeploymentTestParams) (gatewayResp.DeploymentTestResult, error) {
	var dep gateway.ModelDeployment
	if err := global.OPS_DB.WithContext(ctx).Where("deployment_id = ?", req.DeploymentId).First(&dep).Error; err != nil {
		return gatewayResp.DeploymentTestResult{}, err
	}
	var model gateway.Model
	if err := global.OPS_DB.WithContext(ctx).Where("model_id = ?", dep.ModelId).First(&model).Error; err != nil {
		return gatewayResp.DeploymentTestResult{}, errors.New("关联模型不存在")
	}
	cli := litellm.Default()
	if cli == nil {
		return gatewayResp.DeploymentTestResult{}, litellm.ErrNotConfigured
	}
	routeName := BuildModelRouteName(model.ModelKey, true)
	isASR := isSpeechRecognitionModel(model.Capabilities)

	var path string
	var body map[string]any
	switch model.Category {
	case gateway.ModelCategoryEmbedding:
		path = "/v1/embeddings"
		body = map[string]any{"model": routeName, "input": "connectivity-test"}
	case gateway.ModelCategoryRerank:
		path = "/v1/rerank"
		body = map[string]any{"model": routeName, "query": "test", "documents": []string{"test"}}
	default: // chat 及其余类别按对话端点探测
		path = "/v1/chat/completions"
		if isASR {
			// 语音识别模型：消息须含 input_audio part(OpenAI 兼容格式)，纯文本 ping 被上游 400
			body = map[string]any{
				"model":      routeName,
				"max_tokens": 16,
				"messages": []map[string]any{{
					"role": "user",
					"content": []map[string]any{{
						"type":        "input_audio",
						"input_audio": map[string]any{"data": asrProbeAudio, "format": "wav"},
					}},
				}},
			}
		} else {
			body = map[string]any{"model": routeName, "messages": []map[string]any{{"role": "user", "content": "ping"}}, "max_tokens": 1}
		}
	}

	start := time.Now()
	status, respBody, err := cli.RawPost(ctx, path, body)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return gatewayResp.DeploymentTestResult{Success: false, LatencyMs: latency,
			ErrorCategory: "network_error", Message: "网络错误或超时",
			TechnicalDetail: SanitizeTechnicalDetail(err.Error())}, nil
	}
	if status < 400 {
		return gatewayResp.DeploymentTestResult{Success: true, LatencyMs: latency}, nil
	}
	category, msg := classifyUpstreamError(status)
	if isASR && status == 400 {
		// anthropic 协议通道会丢弃 input_audio part，语音识别部署必须走 OpenAI 兼容端点
		msg += "；语音识别部署须走 OpenAI 兼容端点，请核对该部署凭证的 api_base 与格式(openai)"
	}
	return gatewayResp.DeploymentTestResult{Success: false, LatencyMs: latency,
		ErrorCategory: category, Message: msg,
		TechnicalDetail: SanitizeTechnicalDetail(string(respBody))}, nil
}

// ResyncDeployments 全量重推部署投影到 LiteLLM(漂移兜底 + 存量路由名治理，管理员手动触发)。
// 路由名协议隔离((Anthropic) 后缀)已下线：存量远端条目经本端点按 litellm_model_id 原位改名
// (UpdateModel 改 model_name)，anthropic/openai 部署归并同名 LB 组；litellm_model_id 为空的
// 部署走 AddModel 补建。无条件幂等重推(全量下发)，单条失败记入 Failed 不中断整体。
func (s *DeploymentService) ResyncDeployments(ctx context.Context) (gatewayResp.ResyncResult, error) {
	result := gatewayResp.ResyncResult{Failed: []string{}}
	cli := litellm.Default()
	if cli == nil {
		return result, litellm.ErrNotConfigured
	}
	db := global.OPS_DB.WithContext(ctx)
	var deps []gateway.ModelDeployment
	if err := db.Find(&deps).Error; err != nil {
		return result, err
	}
	// 凭证批量预载(解密一次复用；缺失/停用凭证按 nil 处理，routableOf 自会摘池)
	credIds := make([]int64, 0, len(deps))
	for i := range deps {
		if deps[i].CredentialId != 0 {
			credIds = append(credIds, deps[i].CredentialId)
		}
	}
	creds := map[int64]*gateway.Credential{}
	if len(credIds) > 0 {
		var rows []gateway.Credential
		if err := db.Where("credential_id IN ?", credIds).Find(&rows).Error; err == nil {
			for i := range rows {
				creds[rows[i].CredentialId] = &rows[i]
			}
		}
	}
	result.Total = len(deps)
	for i := range deps {
		dep := deps[i]
		var model gateway.Model
		if err := db.Where("model_id = ?", dep.ModelId).First(&model).Error; err != nil {
			result.Failed = append(result.Failed, dep.DeployName)
			continue
		}
		cred := creds[dep.CredentialId]
		format := formatOf(cred)
		if format == "" {
			format = "openai"
		}
		params, modelInfo := buildDeploymentParams(&dep, cred)
		prefix, needsV1 := resolveDeploymentPrefix(db, cred, format, model.Category)
		if err := pushDeployment(ctx, cli, &dep, model.ModelKey, routableOf(dep.IsActive, cred), prefix, needsV1, params, modelInfo); err != nil {
			logger.WithCtx(ctx).Mod("gateway").Err(err).Field("deploymentId", dep.DeploymentId).Error("resync: 部署投影推送失败")
			result.Failed = append(result.Failed, dep.DeployName)
			continue
		}
		if err := db.Model(&gateway.ModelDeployment{}).Where("deployment_id = ?", dep.DeploymentId).
			Updates(map[string]any{
				"litellm_params": marshalJSON(params),
				"model_info":     marshalJSON(modelInfo),
			}).Error; err != nil {
			result.Failed = append(result.Failed, dep.DeployName)
			continue
		}
		result.Pushed++
	}
	return result, nil
}

// classifyUpstreamError 上游 HTTP 状态粗分类(P1 简化版；16 类细模板留前端体验优化)。
func classifyUpstreamError(status int) (category, message string) {
	switch {
	case status == 401 || status == 403:
		return "auth_error", "上游认证失败，请检查凭证(API Key)"
	case status == 404:
		return "model_not_found", "上游模型不存在，请核对模型名"
	case status == 429:
		return "rate_limited", "触发限流，请稍后重试或调整限流配置"
	case status >= 500:
		return "provider_error", "上游服务异常"
	default:
		return "unknown", "请求失败"
	}
}

// toView 部署转出网视图：关联上下文+当前路由名+掩码参数。
func (s *DeploymentService) toView(ctx context.Context, dep gateway.ModelDeployment) gatewayResp.DeploymentView {
	view := gatewayResp.DeploymentView{ModelDeployment: dep, LitellmParams: MaskCredentialValues(jsonToMap(dep.LitellmParams))}
	var model gateway.Model
	if err := global.OPS_DB.WithContext(ctx).Select("model_key", "category").
		Where("model_id = ?", dep.ModelId).First(&model).Error; err == nil {
		view.ModelKey = model.ModelKey
	}
	format := ""
	var cred *gateway.Credential
	if dep.CredentialId != 0 {
		var c gateway.Credential
		if err := global.OPS_DB.WithContext(ctx).Where("credential_id = ?", dep.CredentialId).First(&c).Error; err == nil {
			cred = &c
			view.CredentialName = c.CredentialName
			view.ProviderId = c.ProviderId
			format = formatOf(cred)
			view.ProviderType = providerTypeOfTx(global.OPS_DB.WithContext(ctx), c.ProviderId)
		}
	}
	view.Format = format
	view.RouteName = BuildModelRouteName(view.ModelKey, routableOf(dep.IsActive, cred))
	return view
}

// normalizeBillingType 计费类型归一：空串回落默认 token。
func normalizeBillingType(b string) string {
	if b == "" {
		return gateway.BillingTypeToken
	}
	return b
}
