package middleware

import "testing"

// TestIPAllowed 覆盖 IP 黑白名单校验的关键分支:blacklist/whitelist、单 IP/CIDR、混合分隔、空模式、非法 IP。
func TestIPAllowed(t *testing.T) {
	cases := []struct {
		name      string
		mode      string
		blacklist string
		whitelist string
		ip        string
		wantOK    bool
	}{
		// blacklist 模式(默认)
		{"blacklist 命中单 IP 拒绝", "blacklist", "1.2.3.4", "", "1.2.3.4", false},
		{"blacklist 未命中放行", "blacklist", "1.2.3.4", "", "5.6.7.8", true},
		{"blacklist CIDR 命中拒绝", "blacklist", "10.0.0.0/8", "", "10.1.2.3", false},
		{"blacklist CIDR 未命中放行", "blacklist", "10.0.0.0/8", "", "11.1.1.1", true},
		{"blacklist 逗号换行混合命中拒绝", "blacklist", "1.2.3.4\n5.6.7.8, 9.9.9.9", "", "5.6.7.8", false},
		{"空模式默认走 blacklist 放行", "", "", "", "8.8.8.8", true},

		// whitelist 模式
		{"whitelist 命中放行", "whitelist", "", "1.2.3.4", "1.2.3.4", true},
		{"whitelist 未命中拒绝", "whitelist", "", "1.2.3.4", "5.6.7.8", false},
		{"whitelist CIDR 命中放行", "whitelist", "", "192.168.0.0/16", "192.168.1.1", true},
		{"whitelist 空名单全拒绝", "whitelist", "", "", "1.2.3.4", false},

		// 非法 IP
		{"非法 IP 拒绝", "blacklist", "", "", "not-an-ip", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, msg := ipAllowed(tc.mode, tc.blacklist, tc.whitelist, tc.ip)
			if got != tc.wantOK {
				t.Fatalf("ipAllowed(%q,%q,%q,%q) = (%v, %q), want %v",
					tc.mode, tc.blacklist, tc.whitelist, tc.ip, got, msg, tc.wantOK)
			}
		})
	}
}
