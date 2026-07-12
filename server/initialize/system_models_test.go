package initialize

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils/snowflake"
)

// TestSystemModelsMigrateAndSnowflake 验证：
// 1) 四张表能 AutoMigrate；2) SysUser 主键为 0 时雪花回调填非零 ID；
// 3) password 不外泄；4) 关联表复合主键的显式 ID 不被回调覆盖。
func TestSystemModelsMigrateAndSnowflake(t *testing.T) {
	// 雪花初始化（幂等；epoch 同 snowflake_test.go 的 testEpoch）
	snowflake.MustInit(1, time.Unix(1704067200, 0).UTC())

	// sqlite 内存库（cache=shared 让连接池共享同一库）
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	RegisterCallbacks(db)

	if err := db.AutoMigrate(
		&system.SysUser{},
		&system.SysRole{},
		&system.SysUserRole{},
		&system.SysRoleMenu{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// 主键为 0 → 雪花回调应填充
	u := system.SysUser{UserName: "alice", Password: "should-not-leak", Status: system.StatusEnable}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if u.UserId == 0 {
		t.Fatal("雪花回调应填充 UserId，得到 0")
	}

	// JSON 契约：userId 字符串、password 不外泄
	out, _ := json.Marshal(u)
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["userId"].(string); !ok {
		t.Errorf("userId 应为 JSON 字符串，实际 %T", m["userId"])
	}
	if _, has := m["password"]; has {
		t.Error("password 不应被序列化")
	}

	// 关联表复合主键：显式 ID 不被覆盖
	ur := system.SysUserRole{UserId: u.UserId, RoleId: 999}
	if err := db.Create(&ur).Error; err != nil {
		t.Fatalf("create user-role: %v", err)
	}
	if ur.UserId != u.UserId || ur.RoleId != 999 {
		t.Errorf("关联表显式主键被覆盖：got UserId=%d RoleId=%d", ur.UserId, ur.RoleId)
	}

	rm := system.SysRoleMenu{RoleId: 999, MenuId: 7}
	if err := db.Create(&rm).Error; err != nil {
		t.Fatalf("create role-menu: %v", err)
	}
	if rm.RoleId != 999 || rm.MenuId != 7 {
		t.Errorf("role-menu 显式主键被覆盖：got RoleId=%d MenuId=%d", rm.RoleId, rm.MenuId)
	}
}
