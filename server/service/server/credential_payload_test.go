package server

import (
	"reflect"
	"testing"

	servermod "github.com/hllkk/devops-admin/server/model/server"
)

func TestMaskCredentialValues(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]string
		want map[string]string
	}{
		{
			name: "敏感键打码非敏感键明文",
			in:   map[string]string{"username": "root", "password": "p@ss", "port": "22"},
			want: map[string]string{"username": "root", "password": MaskedValue, "port": "22"},
		},
		{
			name: "SNMP community 与 v3 密钥打码",
			in:   map[string]string{"community": "public", "authPassword": "a", "privPassword": "p", "version": "v3"},
			want: map[string]string{"community": MaskedValue, "authPassword": MaskedValue, "privPassword": MaskedValue, "version": "v3"},
		},
		{
			name: "nil 入参返回空 map",
			in:   nil,
			want: map[string]string{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := MaskCredentialValues(c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestMergeCredentialValues(t *testing.T) {
	cases := []struct {
		name     string
		old      map[string]string
		incoming map[string]string
		want     map[string]string
	}{
		{
			name:     "掩码回传保留旧明文,新值覆盖",
			old:      map[string]string{"username": "root", "password": "old"},
			incoming: map[string]string{"username": "admin", "password": MaskedValue},
			want:     map[string]string{"username": "admin", "password": "old"},
		},
		{
			name:     "旧键不在新值中即删除",
			old:      map[string]string{"username": "root", "password": "old", "extra": "x"},
			incoming: map[string]string{"username": "root", "password": "new"},
			want:     map[string]string{"username": "root", "password": "new"},
		},
		{
			name:     "新增键直接落入",
			old:      map[string]string{"username": "root"},
			incoming: map[string]string{"username": "root", "password": "brand-new"},
			want:     map[string]string{"username": "root", "password": "brand-new"},
		},
		{
			name:     "掩码回传但旧值无该键(异常容错,键丢弃)",
			old:      map[string]string{"username": "root"},
			incoming: map[string]string{"username": "root", "password": MaskedValue},
			want:     map[string]string{"username": "root"},
		},
		{
			name:     "入参不被修改(返回新 map)",
			old:      map[string]string{"password": "old"},
			incoming: map[string]string{"password": MaskedValue},
			want:     map[string]string{"password": "old"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := MergeCredentialValues(c.old, c.incoming)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			// 入参不可变性
			if c.incoming["password"] == MaskedValue && got["password"] != MaskedValue {
				// 已由 DeepEqual 覆盖；此断言防实现直接改写 incoming
				if v, ok := c.incoming["password"]; ok && v != MaskedValue {
					t.Fatalf("incoming 被修改: %v", c.incoming)
				}
			}
		})
	}
}

func TestValidateCredentialType(t *testing.T) {
	for _, ok := range []string{servermod.CredentialTypeSSH, servermod.CredentialTypeBMC, servermod.CredentialTypeSNMP, servermod.CredentialTypeDB} {
		if !ValidateCredentialType(ok) {
			t.Fatalf("合法类型 %q 被拒绝", ok)
		}
	}
	for _, bad := range []string{"", "foo", "SSH", "docker"} {
		if ValidateCredentialType(bad) {
			t.Fatalf("非法类型 %q 被接受", bad)
		}
	}
}

func TestValidateAssetType(t *testing.T) {
	for _, ok := range []string{servermod.AssetTypePhysical, servermod.AssetTypeVm, servermod.AssetTypeDockerHost, servermod.AssetTypeDbInstance, servermod.AssetTypeNetDevice} {
		if !ValidateAssetType(ok) {
			t.Fatalf("合法类型 %q 被拒绝", ok)
		}
	}
	for _, bad := range []string{"", "machine", "Physical", "network"} {
		if ValidateAssetType(bad) {
			t.Fatalf("非法类型 %q 被接受", bad)
		}
	}
}
