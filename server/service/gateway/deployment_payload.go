package gateway

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/hllkk/devops-admin/server/model/gateway"
)

// 本文件是部署投影的纯函数层（零 DB/配置依赖，可单测）。
// 总原则：平台 DB 永远存人民币口径原始值，派生值（前缀化 model、USD/token 换算、
// active 标志、api_base 的 /v1 补齐）只存在于发往 LiteLLM 前的投影构建，绝不回写平台 DB。
// 写回 DB 的仅管线①(凭证绑定: litellm_credential_name/api_base 归凭证) 与 ④(定价镜像 model_info)；
// 管线②③(前缀解析/前缀化 model/补 /v1) 在 pushDeployment 经 ApplyPrefixProjection 临时投影，不落库——
// DB 的 litellm_params.model 始终存用户填的原始厂商模型名(如 glm-5.2)，编辑回显不再带 anthropic/ 前缀。

// BuildModelRouteName 两态路由名唯一入口：
// routable → modelKey 原名（协议差异只存在于 litellm_params.model 前缀，调用方一律用纯模型名，
// LiteLLM 同名混组 LB + 入站协议自动翻译）；!routable → 追加 "__disabled__" 摘出 LB 组
// （litellm_model_id 不变，保归因锚点）。
func BuildModelRouteName(modelKey string, routable bool) string {
	if !routable {
		return modelKey + gateway.ModelDisabledSuffix
	}
	return modelKey
}

// ApplyCredentialToParams 凭证绑定投影：credentialName 为空（内联部署）→ 原样副本（允许 inline api_key）；
// 否则剔除 inline api_key、写 litellm_credential_name 引用（改一次凭证全部部署生效）、
// api_base 归凭证（credential_values.api_base 非空覆盖，为空连历史 base 一起清掉——端点归属凭证）。
// 入参不变异，返回新 map。
func ApplyCredentialToParams(params map[string]any, credentialName string, credentialValues map[string]any) map[string]any {
	out := make(map[string]any, len(params)+1)
	for k, v := range params {
		out[k] = v
	}
	if credentialName == "" {
		return out
	}
	delete(out, "api_key")
	out["litellm_credential_name"] = credentialName
	apiBase, _ := credentialValues["api_base"].(string)
	if apiBase != "" {
		out["api_base"] = apiBase
	} else {
		delete(out, "api_base")
	}
	return out
}

// PrefixModelName 前缀化 model：prefix 为空 → 原样返回（不剥斜杠——内联部署的 model
// 可能本就是 azure/gpt-4 这类合法 LiteLLM 串）；否则剥掉旧前缀取末段重拼 {prefix}/{末段}。
func PrefixModelName(raw, prefix string) string {
	if prefix == "" {
		return raw
	}
	seg := raw
	if idx := strings.LastIndex(raw, "/"); idx >= 0 {
		seg = raw[idx+1:]
	}
	if seg == "" {
		seg = raw
	}
	return prefix + "/" + seg
}

// ApplyPrefixProjection 投影层前缀化(不写回 DB)：拷贝 params，prefix 非空时
// 前缀化 model(PrefixModelName 剥旧前缀取末段重拼) + api_base 补 /v1(EnsureV1Suffix)。
// 入参不变异；prefix 为空 → 原样拷贝(内联部署的 model 可能本就是 azure/gpt-4 合法串)。
// 仅 pushDeployment 推送前调用，确保 DB 的 litellm_params.model 保持原始厂商名。
func ApplyPrefixProjection(params map[string]any, prefix string, needsV1 bool) map[string]any {
	out := make(map[string]any, len(params))
	for k, v := range params {
		out[k] = v
	}
	if prefix != "" {
		if raw, ok := out["model"].(string); ok {
			out["model"] = PrefixModelName(raw, prefix)
		}
		if base, ok := out["api_base"].(string); ok {
			out["api_base"] = EnsureV1Suffix(base, needsV1)
		}
	}
	return out
}

// EnsureV1Suffix needs_v1 且 api_base 未含 /v1 时补齐（去尾斜杠；空串原样）。
func EnsureV1Suffix(apiBase string, needsV1 bool) string {
	if apiBase == "" || !needsV1 {
		return apiBase
	}
	apiBase = strings.TrimRight(apiBase, "/")
	if strings.Contains(apiBase, "/v1") {
		return apiBase
	}
	return apiBase + "/v1"
}

// litellm 定价四键（DB 侧 ¥/百万token，LiteLLM 侧 USD/token）
var costParamKeys = []string{
	"input_cost_per_token",
	"output_cost_per_token",
	"cache_read_input_token_cost",
	"cache_creation_input_token_cost",
}

// param 键 → model_info 镜像键
var costMirrorKeys = map[string]string{
	"input_cost_per_token":            "input_cost",
	"output_cost_per_token":           "output_cost",
	"cache_read_input_token_cost":     "cache_read_cost",
	"cache_creation_input_token_cost": "cache_creation_cost",
}

// ConvertCostsForLitellm 推送副本换算：¥/百万token → USD/token（÷rate÷1e6）。
// rate<=0 兜底 7.0；只处理存在的键；返回新 map（入参不变异，DB 永远人民币口径）。
func ConvertCostsForLitellm(params map[string]any, usdToCnyRate float64) map[string]any {
	if usdToCnyRate <= 0 {
		usdToCnyRate = 7.0
	}
	out := make(map[string]any, len(params))
	for k, v := range params {
		out[k] = v
	}
	for _, key := range costParamKeys {
		if raw, ok := out[key]; ok {
			if f, ok := toFloat(raw); ok && f != 0 {
				out[key] = f / usdToCnyRate / 1_000_000
			}
		}
	}
	return out
}

// MergeCostsToModelInfo 定价镜像：litellm_params 四键(¥/百万token 原值)改名拷贝进 model_info
// （input_cost/output_cost/cache_read_cost/cache_creation_cost，平台成本计算读这里）；
// param 侧不存在的键 → 反向清理 model_info 旧键（防残留旧价）。入参不变异。
func MergeCostsToModelInfo(modelInfo, litellmParams map[string]any) map[string]any {
	out := make(map[string]any, len(modelInfo)+len(costMirrorKeys))
	for k, v := range modelInfo {
		out[k] = v
	}
	for paramKey, mirrorKey := range costMirrorKeys {
		if v, ok := litellmParams[paramKey]; ok {
			out[mirrorKey] = v
		} else {
			delete(out, mirrorKey)
		}
	}
	return out
}

// UnmaskIncomingParams 部署参数更新的掩码还原：以 incoming 为基底替换语义，
// 敏感 key 传入值与库内旧值掩码一致 → 还原旧明文（复用凭证侧掩码协议）。
func UnmaskIncomingParams(oldParams, incoming map[string]any) map[string]any {
	if incoming == nil {
		return oldParams
	}
	out := make(map[string]any, len(incoming))
	for k, v := range incoming {
		if old, ok := oldParams[k].(string); ok && IsSensitiveKey(k) {
			if s, isStr := v.(string); isStr && s == MaskSecret(old) {
				out[k] = old
				continue
			}
		}
		out[k] = v
	}
	return out
}

var (
	sanitizeKeyRe = regexp.MustCompile(`sk-[A-Za-z0-9_-]+`)
	sanitizeKVRe  = regexp.MustCompile(`(?i)("(?:api_key|token|authorization|key)"\s*:\s*")[^"]*(")`)
)

// SanitizeTechnicalDetail 连通测试技术详情脱敏：sk-xxx 串与 JSON 键值对的敏感值掩码，截断 500 字符。
func SanitizeTechnicalDetail(detail string) string {
	s := sanitizeKeyRe.ReplaceAllString(detail, "sk-****")
	s = sanitizeKVRe.ReplaceAllString(s, "${1}****${2}")
	if len(s) > 500 {
		s = s[:500] + "..."
	}
	return s
}

// toFloat JSON 反序列化的数值容错转换（float64/json.Number/string）。
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		var f float64
		if err := json.Unmarshal([]byte(n), &f); err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}
