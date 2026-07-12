package system

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSysUserJSONContract(t *testing.T) {
	now := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)
	u := SysUser{
		UserId:   1234567890123456,
		UserName: "alice",
		Password: "secret",
		Status:   StatusEnable,
		AuditModel: AuditModel{
			CreateBy:   "admin",
			CreateTime: now,
			UpdateBy:   "admin",
			UpdateTime: now,
		},
	}
	out, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// userId 必须是字符串（雪花防精度丢失）
	if v, ok := m["userId"].(string); !ok || v != "1234567890123456" {
		t.Errorf("userId 应为字符串 \"1234567890123456\"，实际 %v", m["userId"])
	}
	// password 永不外泄
	if _, has := m["password"]; has {
		t.Error("password 不应被序列化")
	}
	// deletedAt 不外泄
	if _, ok := m["deletedAt"]; ok {
		t.Error("deletedAt 不应被序列化")
	}
	// 审计与业务字段 camelCase 齐全
	for _, k := range []string{"userName", "status", "createBy", "createTime", "updateBy", "updateTime"} {
		if _, ok := m[k]; !ok {
			t.Errorf("缺少字段 %s", k)
		}
	}
}
