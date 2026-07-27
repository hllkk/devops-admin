package system

import (
	"testing"

	"github.com/hllkk/devops-admin/server/model/system"
)

// TestExpandApiPrefixes 验证从菜单 ApiPrefix 展开 casbin obj pattern:
// 逗号分隔、去首尾空白、去重、跳过空值。结果顺序与首次出现顺序一致。
func TestExpandApiPrefixes(t *testing.T) {
	menus := []system.SysMenu{
		{ApiPrefix: "/system/user, /system/user/*"},
		{ApiPrefix: " /system/role ,/system/role/* "}, // 带首尾空白
		{ApiPrefix: "/system/user/*"},                 // 与首条重复,应去重
		{ApiPrefix: ""},                               // 空:跳过
		{ApiPrefix: "  ,  "},                          // 全空白:跳过
		{ApiPrefix: "/system/dict/type, /system/dict/type/*, /system/dict/data, /system/dict/data/*"},
	}
	got := expandApiPrefixes(menus)
	want := []string{
		"/system/user", "/system/user/*",
		"/system/role", "/system/role/*",
		"/system/dict/type", "/system/dict/type/*", "/system/dict/data", "/system/dict/data/*",
	}
	if len(got) != len(want) {
		t.Fatalf("展开结果数=%d, 期望 %d, got=%v", len(got), len(want), got)
	}
	wantSet := make(map[string]bool, len(want))
	for _, w := range want {
		wantSet[w] = true
	}
	for _, g := range got {
		if !wantSet[g] {
			t.Errorf("展开出意外 pattern: %s", g)
		}
	}
	// 验证去重:结果中无重复
	seen := make(map[string]bool, len(got))
	for _, g := range got {
		if seen[g] {
			t.Errorf("结果中存在重复 pattern: %s", g)
		}
		seen[g] = true
	}
}

// TestExpandApiPrefixesEmpty 验证空输入与全空 ApiPrefix 返回空切片。
func TestExpandApiPrefixesEmpty(t *testing.T) {
	if got := expandApiPrefixes(nil); len(got) != 0 {
		t.Errorf("nil 输入应返回空, got %v", got)
	}
	if got := expandApiPrefixes([]system.SysMenu{{ApiPrefix: ""}, {ApiPrefix: "  "}}); len(got) != 0 {
		t.Errorf("全空 ApiPrefix 应返回空, got %v", got)
	}
}
