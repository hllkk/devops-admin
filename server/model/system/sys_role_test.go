package system

import (
	"encoding/json"
	"testing"
)

func TestSysRoleJSONContract(t *testing.T) {
	r := SysRole{
		RoleId:   1,
		RoleName: "超管",
		RoleKey:  "SUPER",
		RoleSort: 1,
		Status:   StatusEnable,
	}
	out, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// roleId 必须是字符串
	if v, ok := m["roleId"].(string); !ok || v != "1" {
		t.Errorf("roleId 应为字符串 \"1\"，实际 %v", m["roleId"])
	}
	// 业务字段齐全
	for _, k := range []string{"roleName", "roleKey", "roleSort", "menuCheckStrictly", "status", "superAdmin", "flag"} {
		if _, ok := m[k]; !ok {
			t.Errorf("缺少字段 %s", k)
		}
	}
	// deletedAt 不外泄
	if _, ok := m["deletedAt"]; ok {
		t.Error("deletedAt 不应被序列化")
	}
}
