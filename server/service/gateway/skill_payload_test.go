package gateway

import (
	"testing"
	"time"

	"github.com/hllkk/devops-admin/server/model/gateway"
)

func TestNormalizeSkillVersion(t *testing.T) {
	cases := []struct {
		in   string
		want string
		err  bool
	}{
		{"", "1.0.0", false},
		{"1", "1", false},
		{"1.0", "1.0", false},
		{"2.10.3", "2.10.3", false},
		{"v1.0.0", "", true},  // 前缀字母非法
		{"1.0.0.0", "", true}, // 超3段
		{"1..0", "", true},
		{"abc", "", true},
	}
	for _, c := range cases {
		got, err := NormalizeSkillVersion(c.in)
		if c.err {
			if err == nil {
				t.Errorf("NormalizeSkillVersion(%q) 期望报错", c.in)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("NormalizeSkillVersion(%q) = %q,%v want %q", c.in, got, err, c.want)
		}
	}
}

func TestCleanSkillTags(t *testing.T) {
	got := CleanSkillTags([]string{" pdf ", "", "doc", "pdf", "  ", "检索"})
	want := []string{"pdf", "doc", "检索"}
	if len(got) != len(want) {
		t.Fatalf("CleanSkillTags = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("CleanSkillTags = %v, want %v", got, want)
		}
	}
}

func TestSkillTagsMarshalRoundtrip(t *testing.T) {
	raw := MarshalSkillTags(nil)
	if string(raw) != "[]" {
		t.Fatalf("MarshalSkillTags(nil) = %s, want []", raw)
	}
	tags := UnmarshalSkillTags(MarshalSkillTags([]string{"a", "b"}))
	if len(tags) != 2 || tags[0] != "a" || tags[1] != "b" {
		t.Fatalf("roundtrip = %v", tags)
	}
}

func TestSkillZipFilename(t *testing.T) {
	now := time.Date(2026, 8, 28, 9, 5, 3, 0, time.UTC)
	got := SkillZipFilename(123456789, now)
	want := "123456789_20260828090503.zip"
	if got != want {
		t.Fatalf("SkillZipFilename = %q, want %q", got, want)
	}
}

func TestValidSkillUploadFilename(t *testing.T) {
	cases := []struct {
		name string
		ok   bool
	}{
		{"skill.zip", true},
		{"my skill v2.ZIP", true}, // 后缀大小写兼容
		{"", false},
		{"skill.tar.gz", false},
		{"../evil.zip", false}, // 路径穿越
		{`a\b.zip`, false},
	}
	for _, c := range cases {
		if got := ValidSkillUploadFilename(c.name); got != c.ok {
			t.Errorf("ValidSkillUploadFilename(%q) = %v, want %v", c.name, got, c.ok)
		}
	}
}

func TestSkillIdentityOf(t *testing.T) {
	s := gateway.Skill{SkillId: 42}
	if got := SkillIdentityOf(s); got != "42" {
		t.Fatalf("SkillIdentityOf = %q, want 42", got)
	}
}
