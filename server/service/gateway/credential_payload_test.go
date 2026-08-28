package gateway

import (
	"reflect"
	"testing"
)

// 参照 AIHelms test_litellm_credential_payload.py 的三组核心断言（派生/不覆盖/不变异）+ 掩码合并协议。

func TestBuildLitellmCredentialValues_VllmAnthropicDerivesAuthorization(t *testing.T) {
	values := map[string]any{"api_key": "sk-test-1234567890", "api_base": "http://vllm:8000"}
	out := BuildLitellmCredentialValues(values, map[string]any{"format": "anthropic"}, "vllm")

	headers, ok := out["extra_headers"].(map[string]any)
	if !ok {
		t.Fatalf("期望派生 extra_headers, 实际无: %v", out)
	}
	if headers["authorization"] != "Bearer sk-test-1234567890" {
		t.Fatalf("authorization 派生错误: %v", headers["authorization"])
	}
	if out["api_key"] != "sk-test-1234567890" {
		t.Fatalf("原值应保留: %v", out["api_key"])
	}
}

func TestBuildLitellmCredentialValues_ExistingAuthorizationNotOverwritten(t *testing.T) {
	values := map[string]any{
		"api_key":       "sk-test-1234567890",
		"extra_headers": map[string]any{"Authorization": "Bearer custom"},
	}
	out := BuildLitellmCredentialValues(values, map[string]any{"format": "anthropic"}, "vllm")

	headers := out["extra_headers"].(map[string]any)
	if len(headers) != 1 || headers["Authorization"] != "Bearer custom" {
		t.Fatalf("已有 Authorization(大写) 不应被覆盖或重复: %v", headers)
	}
}

func TestBuildLitellmCredentialValues_NonVllmUnchanged(t *testing.T) {
	values := map[string]any{"api_key": "sk-test-1234567890"}
	for _, tt := range []struct {
		providerType string
		info         map[string]any
	}{
		{"openai", map[string]any{"format": "openai"}},
		{"openai", map[string]any{"format": "anthropic"}}, // 非 vllm 即使 anthropic 也不派生
		{"vllm", map[string]any{"format": "openai"}},      // vllm 但 openai 格式不派生
		{"vllm", map[string]any{}},                        // vllm 无 format 不派生
	} {
		out := BuildLitellmCredentialValues(values, tt.info, tt.providerType)
		if _, exists := out["extra_headers"]; exists {
			t.Fatalf("%s/%v 不应派生 extra_headers: %v", tt.providerType, tt.info, out)
		}
		if !reflect.DeepEqual(out, values) {
			t.Fatalf("%s/%v 应原样返回副本: %v", tt.providerType, tt.info, out)
		}
	}
}

func TestBuildLitellmCredentialValues_DoesNotMutateInput(t *testing.T) {
	values := map[string]any{"api_key": "sk-test-1234567890"}
	snapshot := map[string]any{"api_key": "sk-test-1234567890"}
	_ = BuildLitellmCredentialValues(values, map[string]any{"format": "anthropic"}, "vllm")

	if !reflect.DeepEqual(values, snapshot) {
		t.Fatalf("入参 values 被变异: %v", values)
	}
	if len(values) != 1 {
		t.Fatalf("入参 values 不应新增 key: %v", values)
	}
}

func TestMaskSecret(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"short", "*****"},                     // <=8 全 *
		{"12345678", "********"},               // 恰 8 位全 *
		{"sk-test-1234567890", "sk-t****7890"}, // 前4+****+后4
		{"", ""},                               // 空串
	}
	for _, c := range cases {
		if got := MaskSecret(c.in); got != c.want {
			t.Fatalf("MaskSecret(%q) = %q, 期望 %q", c.in, got, c.want)
		}
	}
}

func TestIsSensitiveKey(t *testing.T) {
	sensitive := []string{"api_key", "API_KEY", "apiKey", "secret", "ClientSecret", "access_token", "refreshToken", "password", "Password"}
	for _, k := range sensitive {
		if !IsSensitiveKey(k) {
			t.Fatalf("%q 应判定为敏感", k)
		}
	}
	plain := []string{"api_base", "region", "format", "baseUrl"}
	for _, k := range plain {
		if IsSensitiveKey(k) {
			t.Fatalf("%q 不应判定为敏感", k)
		}
	}
}

func TestMergeCredentialValues(t *testing.T) {
	old := map[string]any{"api_key": "sk-test-1234567890", "api_base": "http://old:8000"}

	// 掩码回传=未修改 → 保留旧明文；api_base 覆盖；新增 region
	incoming := map[string]any{"api_key": MaskSecret("sk-test-1234567890"), "api_base": "http://new:9000", "region": "cn"}
	merged := MergeCredentialValues(old, incoming)
	if merged["api_key"] != "sk-test-1234567890" {
		t.Fatalf("掩码回传应保留旧明文: %v", merged["api_key"])
	}
	if merged["api_base"] != "http://new:9000" || merged["region"] != "cn" {
		t.Fatalf("非敏感覆盖/新增失败: %v", merged)
	}

	// 新 api_key 明文 → 覆盖
	merged = MergeCredentialValues(old, map[string]any{"api_key": "sk-brand-new-9999"})
	if merged["api_key"] != "sk-brand-new-9999" {
		t.Fatalf("新明文应覆盖: %v", merged["api_key"])
	}

	// 入参为空 → 返回旧值副本
	merged = MergeCredentialValues(old, nil)
	if !reflect.DeepEqual(merged, old) {
		t.Fatalf("空入参应返回旧值副本: %v", merged)
	}
}
