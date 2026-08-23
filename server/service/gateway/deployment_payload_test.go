package gateway

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestBuildModelRouteName(t *testing.T) {
	cases := []struct {
		modelKey, format string
		routable         bool
		want             string
	}{
		{"gpt-4o-mini", "openai", true, "gpt-4o-mini"},
		{"gpt-4o-mini", "anthropic", true, "gpt-4o-mini(Anthropic)"},     // anthropic 独立分组
		{"gpt-4o-mini", "openai", false, "gpt-4o-mini__disabled__"},      // 停用出池
		{"gpt-4o-mini", "anthropic", false, "gpt-4o-mini(Anthropic)__disabled__"}, // 叠加
	}
	for _, c := range cases {
		if got := BuildModelRouteName(c.modelKey, c.format, c.routable); got != c.want {
			t.Fatalf("BuildModelRouteName(%q,%q,%v) = %q, 期望 %q", c.modelKey, c.format, c.routable, got, c.want)
		}
	}
}

func TestApplyCredentialToParams(t *testing.T) {
	params := map[string]any{"model": "gpt-4o-mini", "api_key": "sk-inline-12345678", "api_base": "http://old:8000"}

	// 绑定凭证：剔 inline key、写引用、api_base 取凭证值
	out := ApplyCredentialToParams(params, "my-cred", map[string]any{"api_base": "http://vllm:9000"})
	if _, exists := out["api_key"]; exists {
		t.Fatalf("绑定凭证应剔除 inline api_key: %v", out)
	}
	if out["litellm_credential_name"] != "my-cred" {
		t.Fatalf("应写 litellm_credential_name: %v", out)
	}
	if out["api_base"] != "http://vllm:9000" {
		t.Fatalf("api_base 应取凭证值: %v", out["api_base"])
	}

	// 凭证无 api_base → 连历史 base 一起清掉
	out = ApplyCredentialToParams(params, "my-cred", map[string]any{})
	if _, exists := out["api_base"]; exists {
		t.Fatalf("凭证无 api_base 应清除历史 base: %v", out)
	}

	// 内联部署（无凭证名）：原样副本，保留 inline key
	out = ApplyCredentialToParams(params, "", nil)
	if out["api_key"] != "sk-inline-12345678" {
		t.Fatalf("内联部署应保留 api_key: %v", out)
	}

	// 入参不变异
	if params["api_base"] != "http://old:8000" || params["api_key"] != "sk-inline-12345678" {
		t.Fatalf("入参被变异: %v", params)
	}
}

func TestPrefixModelName(t *testing.T) {
	cases := []struct{ raw, prefix, want string }{
		{"gpt-4o-mini", "openai", "openai/gpt-4o-mini"},       // 无前缀直接拼
		{"hosted_vllm/qwen", "openai", "openai/qwen"},         // 剥旧前缀取末段
		{"a/b/c-model", "anthropic", "anthropic/c-model"},     // 多级路径取最后段
		{"azure/gpt-4", "", "azure/gpt-4"},                    // 空 prefix 原样(内联合法斜杠串)
		{"", "openai", "openai/"},                             // 空串兜底
	}
	for _, c := range cases {
		if got := PrefixModelName(c.raw, c.prefix); got != c.want {
			t.Fatalf("PrefixModelName(%q,%q) = %q, 期望 %q", c.raw, c.prefix, got, c.want)
		}
	}
}

func TestEnsureV1Suffix(t *testing.T) {
	cases := []struct {
		apiBase string
		needsV1 bool
		want    string
	}{
		{"http://x:8000", true, "http://x:8000/v1"},   // 补 v1
		{"http://x:8000/", true, "http://x:8000/v1"},  // 去尾斜杠再补
		{"http://x:8000/v1", true, "http://x:8000/v1"}, // 已含跳过
		{"http://x:8000", false, "http://x:8000"},     // 不需要
		{"", true, ""},                                // 空串原样
	}
	for _, c := range cases {
		if got := EnsureV1Suffix(c.apiBase, c.needsV1); got != c.want {
			t.Fatalf("EnsureV1Suffix(%q,%v) = %q, 期望 %q", c.apiBase, c.needsV1, got, c.want)
		}
	}
}

func TestConvertCostsForLitellm(t *testing.T) {
	params := map[string]any{
		"model":                   "openai/gpt-4o-mini",
		"input_cost_per_token":    10.5, // ¥/百万token
		"output_cost_per_token":   42.0,
	}
	out := ConvertCostsForLitellm(params, 7.0)

	wantIn, wantOut := 10.5/7.0/1_000_000, 42.0/7.0/1_000_000
	if got := out["input_cost_per_token"].(float64); math.Abs(got-wantIn) > 1e-15 {
		t.Fatalf("input 换算错误: %v", got)
	}
	if got := out["output_cost_per_token"].(float64); math.Abs(got-wantOut) > 1e-15 {
		t.Fatalf("output 换算错误: %v", got)
	}
	if out["model"] != "openai/gpt-4o-mini" {
		t.Fatalf("非定价键不应被改: %v", out)
	}
	// 入参保持人民币口径
	if params["input_cost_per_token"] != 10.5 {
		t.Fatalf("入参被变异: %v", params)
	}
	// rate 兜底 7.0
	out = ConvertCostsForLitellm(map[string]any{"input_cost_per_token": 7.0}, 0)
	if got := out["input_cost_per_token"].(float64); math.Abs(got-1e-6) > 1e-15 {
		t.Fatalf("rate=0 应兜底 7.0: %v", got)
	}
}

func TestMergeCostsToModelInfo(t *testing.T) {
	params := map[string]any{"input_cost_per_token": 10.5, "output_cost_per_token": 42.0}
	info := map[string]any{"mode": "chat", "input_cost": 99.9, "cache_read_cost": 1.0}

	out := MergeCostsToModelInfo(info, params)
	if out["input_cost"] != 10.5 {
		t.Fatalf("镜像应覆盖旧值: %v", out["input_cost"])
	}
	if out["output_cost"] != 42.0 {
		t.Fatalf("镜像应新增: %v", out)
	}
	if _, exists := out["cache_read_cost"]; exists {
		t.Fatalf("param 侧不存在应清理 model_info 旧键: %v", out)
	}
	if out["mode"] != "chat" {
		t.Fatalf("非定价键应保留: %v", out)
	}
}

func TestUnmaskIncomingParams(t *testing.T) {
	old := map[string]any{"model": "openai/gpt-4o-mini", "api_key": "sk-deploy-1234567890", "api_base": "http://old:8000"}

	// 掩码回传=未修改；api_base 覆盖
	incoming := map[string]any{"model": "openai/gpt-4o-mini", "api_key": MaskSecret("sk-deploy-1234567890"), "api_base": "http://new:9000"}
	merged := UnmaskIncomingParams(old, incoming)
	if merged["api_key"] != "sk-deploy-1234567890" {
		t.Fatalf("掩码回传应还原旧明文: %v", merged["api_key"])
	}
	if merged["api_base"] != "http://new:9000" {
		t.Fatalf("非敏感应覆盖: %v", merged)
	}
	// nil 入参 → 原样
	if !reflect.DeepEqual(UnmaskIncomingParams(old, nil), old) {
		t.Fatalf("nil 入参应原样返回")
	}
}

func TestSanitizeTechnicalDetail(t *testing.T) {
	in := `{"error":{"message":"invalid api_key sk-abc123XYZ789 provided","type":"request_error"}}`
	out := SanitizeTechnicalDetail(in)
	if strings.Contains(out, "sk-abc123XYZ789") {
		t.Fatalf("sk- 串应被掩码: %q", out)
	}
	kv := SanitizeTechnicalDetail(`{"api_key":"secret-value-1"}`)
	if strings.Contains(kv, "secret-value-1") {
		t.Fatalf("JSON 敏感值应被掩码: %q", kv)
	}
	// 超长截断
	long := make([]byte, 800)
	for i := range long {
		long[i] = 'a'
	}
	if got := SanitizeTechnicalDetail(string(long)); len(got) != 503 {
		t.Fatalf("应截断到 500+3, 实际 %d", len(got))
	}
}
