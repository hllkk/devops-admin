package gateway

import (
	"math"
	"testing"
)

// budgetLimitToUsd 纯函数单测：¥→USD 换算与汇率兜底(与 ConvertCostsForLitellm 同口径)。

func TestBudgetLimitToUsd_NormalRate(t *testing.T) {
	if got := budgetLimitToUsd(700, 7.0); math.Abs(got-100) > 1e-9 {
		t.Errorf("¥700@rate7 应换算 $100, 实得 %v", got)
	}
	if got := budgetLimitToUsd(100, 7.2); math.Abs(got-100.0/7.2) > 1e-9 {
		t.Errorf("¥100@rate7.2 应换算 %v, 实得 %v", 100.0/7.2, got)
	}
}

func TestBudgetLimitToUsd_RateFallback(t *testing.T) {
	// rate<=0 兜底 7.0，防配置缺失时按 0 除得 +Inf 下发 LiteLLM
	if got := budgetLimitToUsd(700, 0); math.Abs(got-100) > 1e-9 {
		t.Errorf("rate=0 应兜底 7.0 → $100, 实得 %v", got)
	}
	if got := budgetLimitToUsd(700, -1); math.Abs(got-100) > 1e-9 {
		t.Errorf("rate=-1 应兜底 7.0 → $100, 实得 %v", got)
	}
}

func TestBudgetLimitToUsd_ZeroLimit(t *testing.T) {
	// 停用语义(0 额度)换算后仍为 0，不受兜底影响
	if got := budgetLimitToUsd(0, 7.0); got != 0 {
		t.Errorf("¥0 应换算 $0, 实得 %v", got)
	}
}
