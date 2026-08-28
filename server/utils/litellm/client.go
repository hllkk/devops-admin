// Package litellm 封装与 LiteLLM 代理底座的 HTTP 交互（管理面）。
//
// 转发由 LiteLLM 容器承担，本客户端只调 LiteLLM 管理 API（用 LITELLM_MASTER_KEY 鉴权），
// 供 gateway 业务 service 同步凭证/模型部署/AI 密钥。管理面侧表只存"管理元数据 + LiteLLM
// 引用 ID"，原始转发逻辑不在 devops-admin。详见 aiDoc/modules/business-modules.md「AI 网关模块」节。
package litellm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// ErrNotConfigured LiteLLM 未配置（同步未启用或 base-url/master-key 缺失）。
// 调用方应在同步关闭时静默跳过，而非把该错当业务失败。
var ErrNotConfigured = errors.New("litellm: 未配置或同步未启用")

// Client LiteLLM 管理 API 客户端。
type Client struct {
	baseURL    string
	masterKey  string
	httpClient *http.Client
}

var (
	defaultClient *Client
	defaultOnce   sync.Once
)

// Default 单例客户端，按 global.OPS_CONFIG.Litellm 构造。
// sync-enabled=false 或 base-url/master-key 为空时返回 nil，调用方据此跳过同步。
func Default() *Client {
	defaultOnce.Do(func() {
		defaultClient = newClient()
	})
	return defaultClient
}

func newClient() *Client {
	cfg := global.OPS_CONFIG.Litellm
	if cfg.BaseURL == "" || cfg.MasterKey == "" || !cfg.SyncEnabled {
		return nil
	}
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		masterKey:  cfg.MasterKey,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// do 低层请求：Bearer master-key 鉴权，JSON 收发，>=400 记日志并返回错误。
// out 非 nil 时把响应体反序列化进 out；body 可为 nil（GET/DELETE 无体）。
func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	if c == nil {
		return ErrNotConfigured
	}
	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("litellm: 序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("litellm: 构造请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.masterKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Error(fmt.Sprintf("litellm %s %s 调用失败", method, path))
		return fmt.Errorf("litellm: %s %s 调用失败: %w", method, path, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logger.WithCtx(ctx).Mod("gateway").
			Field("status", resp.StatusCode).
			Error(fmt.Sprintf("litellm %s %s 返回 %d: %s", method, path, resp.StatusCode, truncate(string(respBody), 500)))
		return fmt.Errorf("litellm: %s %s 返回状态 %d", method, path, resp.StatusCode)
	}
	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("litellm: 解析响应失败: %w", err)
		}
	}
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// encodeSegment 对 path 段做百分号编码（对齐 Python quote(safe="")：空格=%20、/=  %2F）。
func encodeSegment(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

// Ping 连通性检查（GET /health/readiness）。供管理面"底座连通"探测与模型部署连通测试复用。
func (c *Client) Ping(ctx context.Context) error {
	return c.do(ctx, http.MethodGet, "/health/readiness", nil, nil)
}

// RawPost 数据面原始 POST：Bearer master-key、JSON 收发，**>=400 不转 error**
// （返回 status+respBody 供调用方自行分类脱敏，如部署连通性测试）。
// 仅传输层错误（连接失败/超时）返回 err。注意：本客户端 DeleteModel 在网关管理面
// 明确不使用（删除=禁用留痕，见 model_service 设计），仅保留方法完整性。
func (c *Client) RawPost(ctx context.Context, path string, body any) (status int, respBody []byte, err error) {
	if c == nil {
		return 0, nil, ErrNotConfigured
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return 0, nil, fmt.Errorf("litellm: 序列化请求体失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, fmt.Errorf("litellm: 构造请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.masterKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.WithCtx(ctx).Mod("gateway").Err(err).Error(fmt.Sprintf("litellm POST %s 调用失败", path))
		return 0, nil, fmt.Errorf("litellm: POST %s 调用失败: %w", path, err)
	}
	defer resp.Body.Close()
	respBody, _ = io.ReadAll(resp.Body)
	return resp.StatusCode, respBody, nil
}

// ----------------------------------------------------------------------------
// 凭证管理（/credentials）—— Credential 同步用
// ----------------------------------------------------------------------------

// CredentialItem LiteLLM 凭证返回项（GET /credentials）。
type CredentialItem struct {
	CredentialName   string         `json:"credential_name"`
	CredentialValues map[string]any `json:"credential_values"`
	CredentialInfo   map[string]any `json:"credential_info"`
}

// CreateCredential 创建 LiteLLM 凭证（POST /credentials）。
// credential_values 形如 {api_key, api_base}；credential_info 可为 nil。
func (c *Client) CreateCredential(ctx context.Context, name string, values, info map[string]any) error {
	body := map[string]any{
		"credential_name":   name,
		"credential_values": values,
		"credential_info":   info, // LiteLLM schema 要求三字段齐全，nil 会 marshal 成 null
	}
	return c.do(ctx, http.MethodPost, "/credentials", body, nil)
}

// UpdateCredential 更新 LiteLLM 凭证（PATCH /credentials/{name}）。
// LiteLLM 的 CredentialItem schema 在 PATCH 时三字段都必填。
func (c *Client) UpdateCredential(ctx context.Context, name string, values, info map[string]any) error {
	body := map[string]any{
		"credential_name":   name,
		"credential_values": values,
		"credential_info":   info,
	}
	return c.do(ctx, http.MethodPatch, "/credentials/"+encodeSegment(name), body, nil)
}

// DeleteCredential 删除 LiteLLM 凭证（DELETE /credentials/{name}）。
func (c *Client) DeleteCredential(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodDelete, "/credentials/"+encodeSegment(name), nil, nil)
}

// ListCredentials 列出 LiteLLM 凭证（GET /credentials）。
func (c *Client) ListCredentials(ctx context.Context) (items []CredentialItem, err error) {
	var resp struct {
		Credentials []CredentialItem `json:"credentials"`
	}
	if err = c.do(ctx, http.MethodGet, "/credentials", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Credentials, nil
}

// GetProviderFields 拉取各 provider_type 的凭证字段定义（GET /public/providers/fields）。
// 供前端动态渲染凭证表单：按 provider_type 知道需要哪些字段（api_key/api_base 等）。
func (c *Client) GetProviderFields(ctx context.Context) (fields []map[string]any, err error) {
	if c == nil {
		return nil, ErrNotConfigured
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/public/providers/fields", nil)
	if err != nil {
		return nil, fmt.Errorf("litellm: 构造请求失败: %w", err)
	}
	// public 端点不需要 master-key，附上无妨
	req.Header.Set("Authorization", "Bearer "+c.masterKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("litellm: get provider fields 调用失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("litellm: get provider fields 返回状态 %d", resp.StatusCode)
	}
	if err := json.Unmarshal(respBody, &fields); err != nil {
		return nil, fmt.Errorf("litellm: 解析 provider fields 失败: %w", err)
	}
	return fields, nil
}

// ----------------------------------------------------------------------------
// 模型部署管理（/model/*）—— ModelDeployment 同步用
// ----------------------------------------------------------------------------

// AddModelResp /model/new 响应。LiteLLM 把新部署的 id 放在 model_info.id。
type AddModelResp struct {
	ModelInfo struct {
		ID            string         `json:"id"`
		ModelName     string         `json:"model_name"`
		LitellmParams map[string]any `json:"litellm_params"`
	} `json:"model_info"`
}

// AddModel 新增 LiteLLM 模型部署（POST /model/new）。
// litellm_params 形如 {model, api_base, api_key, custom_llm_provider...}；model_info 为内外定价等元数据。
func (c *Client) AddModel(ctx context.Context, modelName string, litellmParams, modelInfo map[string]any) (AddModelResp, error) {
	body := map[string]any{
		"model_name":      modelName,
		"litellm_params":  litellmParams,
		"model_info":      modelInfo,
	}
	var resp AddModelResp
	if err := c.do(ctx, http.MethodPost, "/model/new", body, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// DeleteModel 删除 LiteLLM 模型部署（POST /model/delete {id}）。
func (c *Client) DeleteModel(ctx context.Context, litellmModelID string) error {
	body := map[string]any{"id": litellmModelID}
	return c.do(ctx, http.MethodPost, "/model/delete", body, nil)
}

// UpdateModel 更新 LiteLLM 模型部署（PATCH /model/{id}/update）。
// 传 zero 值字段表示不改：仅非空字段才写入请求体（对齐 Python 的可选字段语义）。
func (c *Client) UpdateModel(ctx context.Context, litellmModelID, modelName string, litellmParams, modelInfo map[string]any) error {
	body := map[string]any{}
	if modelName != "" {
		body["model_name"] = modelName
	}
	if len(litellmParams) > 0 {
		body["litellm_params"] = litellmParams
	}
	if modelInfo != nil {
		body["model_info"] = modelInfo
	}
	return c.do(ctx, http.MethodPatch, "/model/"+encodeSegment(litellmModelID)+"/update", body, nil)
}

// ListModels 列出 LiteLLM 已注册的模型部署（GET /model/info）。
func (c *Client) ListModels(ctx context.Context) (models []map[string]any, err error) {
	var resp struct {
		Data []map[string]any `json:"data"`
	}
	if err = c.do(ctx, http.MethodGet, "/model/info", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// ----------------------------------------------------------------------------
// AI 密钥管理（/key/*）—— AiKey 同步用
// ----------------------------------------------------------------------------

// KeyCreateReq /key/generate 请求。零值字段不写入请求体。
type KeyCreateReq struct {
	KeyAlias           string   `json:"key_alias"`
	UserID             string   `json:"user_id,omitempty"`
	TeamID             string   `json:"team_id,omitempty"`
	Models             []string `json:"models,omitempty"`
	MaxBudget          *float64 `json:"max_budget,omitempty"`          // USD 口径(LiteLLM spend 记 USD，平台 ¥ 须先换算)
	BudgetDuration     string   `json:"budget_duration,omitempty"`     // 预算重置窗口("1d"/"7d"/"30d"，LiteLLM 到期重置 spend)
	Metadata           map[string]any `json:"metadata,omitempty"`
	Duration           string   `json:"duration,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"` // 过期时间(ISO，nil=永不过期)
	AllowedMCPServers  []string `json:"allowed_mcp_servers,omitempty"`
	TPMLimit           *int     `json:"tpm_limit,omitempty"`
	RPMLimit           *int     `json:"rpm_limit,omitempty"`
	MaxParallelReqs    *int     `json:"max_parallel_requests,omitempty"`
}

// KeyCreateResp /key/generate 响应。Key 仅在创建时返回一次，须即时落库。
// LiteLLM 1.98 用 token_id 作内部密钥标识(后续 /key/update、/key/delete、/key/info 均用它)，
// 早期版本用 key_id——CreateKey 兼容两者，优先 token_id。
type KeyCreateResp struct {
	Key      string `json:"key"`
	KeyID    string `json:"key_id"`
	TokenID  string `json:"token_id"`
	KeyAlias string `json:"key_alias"`
}

// CreateKey 生成 LiteLLM 虚拟 Key（POST /key/generate）。返回的 Key 仅此一次，调用方须落库。
func (c *Client) CreateKey(ctx context.Context, req KeyCreateReq) (KeyCreateResp, error) {
	var resp KeyCreateResp
	if err := c.do(ctx, http.MethodPost, "/key/generate", req, &resp); err != nil {
		return resp, err
	}
	if resp.KeyID == "" && resp.TokenID != "" {
		resp.KeyID = resp.TokenID // 1.98 兼容
	}
	return resp, nil
}

// DeleteKey 删除 LiteLLM 虚拟 Key（POST /key/delete {keys:[id]}）。
func (c *Client) DeleteKey(ctx context.Context, keyID string) error {
	body := map[string]any{"keys": []string{keyID}}
	return c.do(ctx, http.MethodPost, "/key/delete", body, nil)
}

// KeyUpdateReq /key/update 请求。SyncRateLimits=true 强制刷限流字段(即便为零值)；
// SyncBudget=true 强制刷 max_budget(含 nil→null 清空，用于停用→启用无硬限恢复)；
// SyncExpiry=true 强制刷 expires_at(含 nil→null 清空，用于改回永不过期)。
type KeyUpdateReq struct {
	Models            []string `json:"models,omitempty"`
	MaxBudget         *float64 `json:"max_budget,omitempty"`      // USD 口径(LiteLLM spend 记 USD，平台 ¥ 须先换算)
	BudgetDuration    string   `json:"budget_duration,omitempty"` // 预算重置窗口("1d"/"7d"/"30d"，LiteLLM 到期重置 spend)
	Metadata          map[string]any `json:"metadata,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	AllowedMCPServers []string `json:"allowed_mcp_servers,omitempty"`
	TPMLimit          *int     `json:"tpm_limit,omitempty"`
	RPMLimit          *int     `json:"rpm_limit,omitempty"`
	MaxParallelReqs   *int     `json:"max_parallel_requests,omitempty"`
	SyncRateLimits    bool     `json:"-"`
	SyncBudget        bool     `json:"-"`
	SyncExpiry        bool     `json:"-"`
}

// UpdateKey 更新 LiteLLM 虚拟 Key（POST /key/update {key:id, ...}）。
func (c *Client) UpdateKey(ctx context.Context, keyID string, req KeyUpdateReq) error {
	body := map[string]any{"key": keyID}
	if len(req.Models) > 0 {
		body["models"] = req.Models
	}
	if req.MaxBudget != nil || req.SyncBudget {
		body["max_budget"] = req.MaxBudget // nil + SyncBudget → JSON null 清空
	}
	if req.BudgetDuration != "" {
		body["budget_duration"] = req.BudgetDuration // 平台预算周期恒有值(1d/7d/30d)，与平台侧重算窗口对齐
	}
	if req.ExpiresAt != nil || req.SyncExpiry {
		body["expires_at"] = req.ExpiresAt // nil + SyncExpiry → JSON null 清空
	}
	if len(req.Metadata) > 0 {
		body["metadata"] = req.Metadata
	}
	if len(req.AllowedMCPServers) > 0 {
		body["allowed_mcp_servers"] = req.AllowedMCPServers
	}
	if req.SyncRateLimits || req.TPMLimit != nil {
		body["tpm_limit"] = req.TPMLimit
	}
	if req.SyncRateLimits || req.RPMLimit != nil {
		body["rpm_limit"] = req.RPMLimit
	}
	if req.SyncRateLimits || req.MaxParallelReqs != nil {
		body["max_parallel_requests"] = req.MaxParallelReqs
	}
	return c.do(ctx, http.MethodPost, "/key/update", body, nil)
}

// KeyInfo /key/info 响应（精简，按需扩展）。
type KeyInfo struct {
	KeyID    string  `json:"key_id"`
	MaxBudget float64 `json:"max_budget"`
	TPMLimit  int     `json:"tpm_limit"`
	RPMLimit  int     `json:"rpm_limit"`
}

// GetKeyInfo 查 LiteLLM 虚拟 Key 信息（GET /key/info?key=id）。
func (c *Client) GetKeyInfo(ctx context.Context, keyID string) (KeyInfo, error) {
	var info KeyInfo
	if err := c.do(ctx, http.MethodGet, "/key/info?key="+encodeSegment(keyID), nil, &info); err != nil {
		return info, err
	}
	return info, nil
}

// ----------------------------------------------------------------------------
// 路由策略（/router/settings）—— RouterSettings 同步用
// ----------------------------------------------------------------------------

// GetRouterSettings 获取 LiteLLM 路由策略（GET /router/settings）。
// 返回 LiteLLM 当前 router_settings（routing_strategy/allowed_fails/cooldown_time/...）。
func (c *Client) GetRouterSettings(ctx context.Context) (map[string]any, error) {
	var resp struct {
		RouterSettings map[string]any `json:"router_settings"`
	}
	if err := c.do(ctx, http.MethodGet, "/router/settings", nil, &resp); err != nil {
		return nil, err
	}
	return resp.RouterSettings, nil
}

// UpdateRouterSettings 更新 LiteLLM 路由策略（POST /router/settings）。
// settings 为 LiteLLM 蛇形键 map（routing_strategy/allowed_fails/cooldown_time/
// num_retries/timeout/fallbacks/config），热更新即时生效。
func (c *Client) UpdateRouterSettings(ctx context.Context, settings map[string]any) error {
	return c.do(ctx, http.MethodPost, "/router/settings", settings, nil)
}
