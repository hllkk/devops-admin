package gateway

import "testing"

// TestMergeMissingKeys 主 Key 自愈差集/建 Key 默认授权的合并口径:
//   - 可见免审批模型 + 本人已批准申请的模型并集,剔重剔空
//   - current 已持有的不重复补(幂等,自愈反复触发不追加)
func TestMergeMissingKeys(t *testing.T) {
	cases := []struct {
		name    string
		current []string
		sources [][]string
		want    []string
	}{
		{
			name:    "两来源并集去重保序",
			current: []string{"m1"},
			sources: [][]string{{"m1", "m2", "m3"}, {"m3", "m4"}},
			want:    []string{"m2", "m3", "m4"},
		},
		{
			name:    "空串与空来源跳过",
			current: []string{"m1"},
			sources: [][]string{{"", "m2", ""}, nil},
			want:    []string{"m2"},
		},
		{
			name:    "无缺失返回空非nil",
			current: []string{"m1", "m2"},
			sources: [][]string{{"m1"}, {"m2"}},
			want:    []string{},
		},
		{
			name:    "current为空全量补齐",
			current: nil,
			sources: [][]string{{"b", "a"}, {"a", "c"}},
			want:    []string{"b", "a", "c"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mergeMissingKeys(c.current, c.sources...)
			if len(got) != len(c.want) {
				t.Fatalf("mergeMissingKeys()=%v, 期望 %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("mergeMissingKeys()=%v, 期望 %v(顺序不符)", got, c.want)
				}
			}
		})
	}
}
