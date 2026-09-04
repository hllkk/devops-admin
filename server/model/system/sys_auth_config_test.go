package system

import "testing"

// TestNormalizeWecomDomainFileName 校验文件名规范化：trim / 仅中段补全 / 前后缀缺失补全
func TestNormalizeWecomDomainFileName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"完整文件名原样保留", "WW_verify_abc123.txt", "WW_verify_abc123.txt"},
		{"仅中段自动补前后缀", "abc123", "WW_verify_abc123.txt"},
		{"中段+txt自动补前缀", "abc123.txt", "WW_verify_abc123.txt"},
		{"有前缀无后缀自动补", "WW_verify_abc123", "WW_verify_abc123.txt"},
		{"首尾空白剔除", "  WW_verify_abc123.txt \n", "WW_verify_abc123.txt"},
		{"纯空白归空串", "   ", ""},
		{"空串原样", "", ""},
	}
	for _, tc := range cases {
		if got := NormalizeWecomDomainFileName(tc.in); got != tc.want {
			t.Errorf("%s: NormalizeWecomDomainFileName(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}
