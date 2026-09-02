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

// effectiveRetentionDays 纯函数单测：保留期与对账窗口联动。

func TestEffectiveRetentionDays_Disabled(t *testing.T) {
	// 0/负值=清理禁用，无论窗口多大
	for _, cfg := range []int{0, -1, -30} {
		if got := effectiveRetentionDays(cfg, 30); got != 0 {
			t.Errorf("cfg=%d 应禁用返回0, 实际 %d", cfg, got)
		}
	}
}

func TestEffectiveRetentionDays_Normal(t *testing.T) {
	// 配置值 >= 对账窗口+7 → 原样生效
	if got := effectiveRetentionDays(90, 30); got != 90 {
		t.Errorf("90天应原样生效, 实际 %d", got)
	}
	// 恰好等于下限(30+7)不抬升
	if got := effectiveRetentionDays(37, 30); got != 37 {
		t.Errorf("37=下限边界应原样生效, 实际 %d", got)
	}
}

func TestEffectiveRetentionDays_LiftToWindow(t *testing.T) {
	// 配置值 < 对账窗口+7 → 抬升到下限(防删了又被对账重灌的抖动循环)
	if got := effectiveRetentionDays(10, 30); got != 37 {
		t.Errorf("10天应联动抬升至37, 实际 %d", got)
	}
	// 自定义更大的对账窗口同样联动
	if got := effectiveRetentionDays(50, 60); got != 67 {
		t.Errorf("窗口60时50天应抬升至67, 实际 %d", got)
	}
}

func TestEffectiveRetentionDays_WindowFallback(t *testing.T) {
	// 对账窗口未配(<=0)兜底默认30 → 下限37
	if got := effectiveRetentionDays(10, 0); got != 37 {
		t.Errorf("窗口未配应兜底30+7=37, 实际 %d", got)
	}
	// 窗口未配但配置值已超下限 → 原样
	if got := effectiveRetentionDays(90, 0); got != 90 {
		t.Errorf("窗口未配90天应原样生效, 实际 %d", got)
	}
}
