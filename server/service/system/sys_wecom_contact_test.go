package system

import (
	"testing"
)

// TestWecomGenderToSex 验证企微性别(string)→项目 Sex 字典(string,0男1女2未知)转换:
// 仅 "1"/"2" 是真实值,其余("0"未定义/空/脏值)归"未知"。
func TestWecomGenderToSex(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"1", "0"}, // 企微男 → 项目男
		{"2", "1"}, // 企微女 → 项目女
		{"0", "2"}, // 未定义 → 未知
		{"", "2"},  // 缺失 → 未知
		{"x", "2"}, // 脏值 → 未知
	}
	for _, c := range cases {
		if got := WecomGenderToSex(c.in); got != c.want {
			t.Errorf("WecomGenderToSex(%q)=%q, 期望 %q", c.in, got, c.want)
		}
	}
}

// TestPostCodeHex 验证岗位名→8 位 hex 的稳定性(同名同值、异名几乎必异)。
func TestPostCodeHex(t *testing.T) {
	if got := postCodeHex("前端工程师"); got != postCodeHex("前端工程师") {
		t.Errorf("同名岗位生成的 post_code 不稳定: %s", got)
	}
	if len(postCodeHex("前端工程师")) != 8 {
		t.Errorf("post_code hex 长度=%d, 期望 8", len(postCodeHex("前端工程师")))
	}
	if postCodeHex("前端工程师") == postCodeHex("后端工程师") {
		t.Errorf("不同岗位名生成了相同 post_code(sha1 前4字节碰撞)")
	}
}

// TestDeptSetEqual 验证部门 id 集合比较:长度不同/元素不同为 false,相同为 true(nil 与空集等价)。
func TestDeptSetEqual(t *testing.T) {
	if !deptSetEqual(map[int64]bool{1: true, 2: true}, map[int64]bool{2: true, 1: true}) {
		t.Errorf("相同集合(顺序无关)应相等")
	}
	if deptSetEqual(map[int64]bool{1: true}, map[int64]bool{1: true, 2: true}) {
		t.Errorf("长度不同的集合不应相等")
	}
	if deptSetEqual(map[int64]bool{1: true}, map[int64]bool{2: true}) {
		t.Errorf("元素不同的集合不应相等")
	}
	if !deptSetEqual(map[int64]bool{}, nil) {
		t.Errorf("空集与 nil 应相等")
	}
}

// TestResolveLocalDeptIds 验证企微部门 id 列表→本地部门 id 列表映射:
// 未同步的(不在 map/值为 0)被过滤,顺序保持。
func TestResolveLocalDeptIds(t *testing.T) {
	byWecom := map[int64]int64{10: 101, 20: 202, 30: 0}
	got := resolveLocalDeptIds([]int64{10, 99, 20, 30}, byWecom)
	want := []int64{101, 202}
	if len(got) != len(want) {
		t.Fatalf("映射结果数=%d, 期望 %d, got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("映射结果[%d]=%d, 期望 %d", i, got[i], want[i])
		}
	}
}
