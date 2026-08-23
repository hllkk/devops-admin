package gateway

import "strings"

// 本文件是凭证投影与脱敏的纯函数层（零 DB/配置依赖，可单测）。
// 总原则（对齐 aiDoc「实现参照·AIHelms」第 3 条）：平台 DB 永远存原始值（人民币口径/明文键值，
// 键值本身经 AES 加密落列），派生值只存在于发往 LiteLLM 前的投影构建，绝不回写平台 DB。

// BuildLitellmCredentialValues 构建发往 LiteLLM 的凭证投影值。
// 拷贝入参（绝不变异、不回写）；仅当 providerType==vllm 且 info.format==anthropic 时
// 派生 extra_headers.authorization=Bearer <api_key>（已有 authorization 头则不覆盖，大小写不敏感），
// 其余供应商/格式组合原样返回副本。参照 AIHelms litellm_credential_payload.py。
func BuildLitellmCredentialValues(values, info map[string]any, providerType string) map[string]any {
	out := make(map[string]any, len(values)+1)
	for k, v := range values {
		out[k] = v
	}
	if providerType != "vllm" {
		return out
	}
	if s, ok := info["format"].(string); !ok || s != "anthropic" {
		return out
	}
	apiKey, _ := values["api_key"].(string)
	if apiKey == "" {
		return out
	}
	headers := map[string]any{}
	if existing, ok := values["extra_headers"].(map[string]any); ok {
		for name, v := range existing {
			headers[name] = v
		}
	}
	for name := range headers {
		if strings.EqualFold(name, "authorization") {
			return out // 已有 authorization 头，不覆盖
		}
	}
	headers["authorization"] = "Bearer " + apiKey
	out["extra_headers"] = headers
	return out
}

// IsSensitiveKey 判定凭证键名是否敏感（值需掩码出网）。命中 key/secret/token/password 子串，大小写不敏感。
func IsSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	return strings.Contains(lower, "key") || strings.Contains(lower, "secret") ||
		strings.Contains(lower, "token") || strings.Contains(lower, "password")
}

// MaskSecret 掩码单个敏感值：长度 <=8 全 '*'，否则前 4 + "****" + 后 4。
func MaskSecret(s string) string {
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + "****" + s[len(s)-4:]
}

// MaskCredentialValues 掩码凭证键值出网：仅对 string 且敏感 key 的值掩码，api_base 等非敏感值明文。
// 入参为解密后的明文 map，返回新 map 不变异入参。
func MaskCredentialValues(values map[string]any) map[string]any {
	out := make(map[string]any, len(values))
	for k, v := range values {
		if s, ok := v.(string); ok && IsSensitiveKey(k) {
			out[k] = MaskSecret(s)
			continue
		}
		out[k] = v
	}
	return out
}

// MergeCredentialValues 合并更新传入的凭证键值（浅 merge，对齐 AIHelms 更新语义）。
// 前端编辑时敏感值以掩码回传：传入值与旧值掩码一致 → 保留旧明文（用户未改）；
// 传入新值 → 覆盖；新增 key → 写入；非敏感 key → 直接覆盖。
func MergeCredentialValues(oldValues, incoming map[string]any) map[string]any {
	out := make(map[string]any, len(oldValues)+len(incoming))
	for k, v := range oldValues {
		out[k] = v
	}
	for k, v := range incoming {
		if old, ok := oldValues[k].(string); ok && IsSensitiveKey(k) {
			if s, isStr := v.(string); isStr && s == MaskSecret(old) {
				continue // 掩码回传=未修改，保留旧明文
			}
		}
		out[k] = v
	}
	return out
}
