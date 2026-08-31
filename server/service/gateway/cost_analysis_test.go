package gateway

import (
	"testing"
	"time"
)

// normalizeCostRange：缺省回退本月、非法串忽略、start>end 交换、天数闭区间
func TestNormalizeCostRange(t *testing.T) {
	now := time.Date(2026, 8, 30, 10, 0, 0, 0, time.Local)

	// 缺省：本月首日~今天
	s, e, d := normalizeCostRange("", "", now)
	if s != "2026-08-01" || e != "2026-08-30" || d != 30 {
		t.Fatalf("缺省本月失败: %s~%s days=%d", s, e, d)
	}
	// 显式区间
	s, e, d = normalizeCostRange("2026-08-10", "2026-08-19", now)
	if s != "2026-08-10" || e != "2026-08-19" || d != 10 {
		t.Fatalf("显式区间失败: %s~%s days=%d", s, e, d)
	}
	// start>end 交换
	s, e, _ = normalizeCostRange("2026-08-19", "2026-08-10", now)
	if s != "2026-08-10" || e != "2026-08-19" {
		t.Fatalf("交换失败: %s~%s", s, e)
	}
	// 非法串回退缺省
	s, e, _ = normalizeCostRange("not-a-date", "2026-08-15", now)
	if s != "2026-08-01" || e != "2026-08-15" {
		t.Fatalf("非法start回退失败: %s~%s", s, e)
	}
	// 同一天=1天
	_, _, d = normalizeCostRange("2026-08-15", "2026-08-15", now)
	if d != 1 {
		t.Fatalf("同一天天数=%d want 1", d)
	}
}

// prevCostRange：等长上一期，紧贴本期之前
func TestPrevCostRange(t *testing.T) {
	// 本期 10 天 → 上期 10 天，上期末=本期首前一天
	ps, pe := prevCostRange("2026-08-10", "2026-08-19")
	if ps != "2026-07-31" || pe != "2026-08-09" {
		t.Fatalf("等长上一期失败: %s~%s", ps, pe)
	}
	// 跨月
	ps, pe = prevCostRange("2026-08-01", "2026-08-01")
	if ps != "2026-07-31" || pe != "2026-07-31" {
		t.Fatalf("单日上一期失败: %s~%s", ps, pe)
	}
}

// costChange：上期 0 保护
func TestCostChange(t *testing.T) {
	if got := costChange(0, 0); got != 0 {
		t.Fatalf("双零应给0, got %v", got)
	}
	if got := costChange(100, 0); got != 0 {
		t.Fatalf("上期0应给0, got %v", got)
	}
	if got := costChange(150, 100); got != 50 {
		t.Fatalf("环比+50%%失败, got %v", got)
	}
	if got := costChange(50, 100); got != -50 {
		t.Fatalf("环比-50%%失败, got %v", got)
	}
}

// 维度与排序白名单
func TestCostDimensionAndSortWhitelist(t *testing.T) {
	if costDimensionOf("user") != dimensionUser || costDimensionOf("model") != dimensionModel ||
		costDimensionOf("aiKey") != dimensionAiKey || costDimensionOf("provider") != costDimensionProvider ||
		costDimensionOf("date") != costDimensionDate {
		t.Fatal("合法维度被拒")
	}
	for _, v := range []string{"", "department", "xxx", "cost"} {
		if costDimensionOf(v) != costDimensionDepartment {
			t.Fatalf("非法维度 %q 应回退 department", v)
		}
	}
	if costSortColumn("internal") != "internal_cost" || costSortColumn("external") != "external_cost" ||
		costSortColumn("requests") != "requests" || costSortColumn("tokens") != "total_tokens" {
		t.Fatal("合法排序键被拒")
	}
	for _, v := range []string{"", "xxx; DROP TABLE", "1=1"} {
		if costSortColumn(v) != "internal_cost" {
			t.Fatalf("非法排序键 %q 应回退 internal_cost", v)
		}
	}
}

// costGroupExpr：各维度分组/取值表达式成对返回，department 锚点含 CASE
func TestCostGroupExpr(t *testing.T) {
	if g, _ := costGroupExpr(dimensionUser); g != "s.user_id" {
		t.Fatalf("user 分组表达式=%s", g)
	}
	if g, _ := costGroupExpr(dimensionModel); g != "s.model" {
		t.Fatalf("model 分组表达式=%s", g)
	}
	g, v := costGroupExpr(costDimensionDepartment)
	if g != costDeptAnchor || v != "CAST("+costDeptAnchor+" AS TEXT)" {
		t.Fatalf("department 分组/取值表达式异常: %s / %s", g, v)
	}
}
