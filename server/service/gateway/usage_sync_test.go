package gateway

import (
	"testing"
)

// applyTokenFallback 纯函数单测：顶层 token 列为 0 时从 metadata.usage_object 兜底。

func usageMeta(prompt, completion any) map[string]any {
	return map[string]any{"usage_object": map[string]any{
		"prompt_tokens": prompt, "completion_tokens": completion,
	}}
}

func TestApplyTokenFallback_AllFromUsageObject(t *testing.T) {
	// 顶层全 0(部分 LiteLLM 版本只写 usage_object) → 全兜底,total 回落之和
	p, c, tt := applyTokenFallback(0, 0, 0, usageMeta(120.0, 30.0))
	if p != 120 || c != 30 || tt != 150 {
		t.Errorf("全兜底失败: prompt=%d completion=%d total=%d, 期望 120/30/150", p, c, tt)
	}
}

func TestApplyTokenFallback_PartialFallback(t *testing.T) {
	// 仅 prompt 为 0 → 只补 prompt,不覆盖非零 completion;total 非零不动
	p, c, tt := applyTokenFallback(0, 30, 99, usageMeta(120.0, 45.0))
	if p != 120 || c != 30 || tt != 99 {
		t.Errorf("部分兜底失败: prompt=%d completion=%d total=%d, 期望 120/30/99", p, c, tt)
	}
}

func TestApplyTokenFallback_TopLevelWins(t *testing.T) {
	// 顶层非零 → usage_object 不参与(不覆盖顶层列值)
	p, c, tt := applyTokenFallback(100, 20, 120, usageMeta(999.0, 999.0))
	if p != 100 || c != 20 || tt != 120 {
		t.Errorf("顶层值应优先: prompt=%d completion=%d total=%d, 期望 100/20/120", p, c, tt)
	}
}

func TestApplyTokenFallback_NoUsageObject(t *testing.T) {
	// 无 usage_object 且顶层全 0 → 保持 0(归因失败/无 token 的调用),total 不凭空造数
	p, c, tt := applyTokenFallback(0, 0, 0, map[string]any{})
	if p != 0 || c != 0 || tt != 0 {
		t.Errorf("无兜底来源应保持 0: prompt=%d completion=%d total=%d", p, c, tt)
	}
}

func TestApplyTokenFallback_TotalOnlyMissing(t *testing.T) {
	// 顶层 prompt/completion 有值但 total 为 0 → total 回落之和
	p, c, tt := applyTokenFallback(100, 20, 0, nil)
	if p != 100 || c != 20 || tt != 120 {
		t.Errorf("total 回落失败: prompt=%d completion=%d total=%d, 期望 100/20/120", p, c, tt)
	}
}
