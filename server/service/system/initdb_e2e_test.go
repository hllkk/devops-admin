package system_test

import (
	"context"
	"sort"
	"strconv"
	"testing"

	sysSvc "github.com/hllkk/devops-admin/server/service/system"
	_ "github.com/hllkk/devops-admin/server/source/system" // 触发 seed initializer 的 init() 自注册
	"github.com/hllkk/devops-admin/server/utils"
)

// TestE2EPermissionBaseClosedLoop 端到端验证权限基座闭环：
// 驱动全部 seed initializer（按 order）→ 验证 seed 计数 → Login 三用户 →
// GetUserInfo 聚合三分支（super *:*:* / admin 系统管理 perms / test1 空）→ casbin 兜底。
func TestE2EPermissionBaseClosedLoop(t *testing.T) {
	db := sysSvc.SetupTestDBExternal(t)
	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)
	ctx = context.WithValue(ctx, "adminPassword", "123456")

	inits := *sysSvc.ExportedInitializers
	sort.Sort(inits)
	// 先建表
	for _, init := range inits {
		if init.TableCreated(ctx) {
			continue
		}
		next, err := init.MigrateTable(ctx)
		if err != nil {
			t.Fatalf("migrate %s: %v", init.InitializerName(), err)
		}
		ctx = next
	}
	// 再灌数据（按 order，依赖由 order 保证）
	for _, init := range inits {
		if init.DataInserted(ctx) {
			continue
		}
		next, err := init.InitializeData(ctx)
		if err != nil {
			t.Fatalf("init %s: %v", init.InitializerName(), err)
		}
		ctx = next
	}

	// 1. seed 计数
	var userCount, roleCount, menuCount, roleMenuCount int64
	db.Table("sys_user").Count(&userCount)
	db.Table("sys_role").Count(&roleCount)
	db.Table("sys_menu").Count(&menuCount)
	db.Table("sys_role_menu").Count(&roleMenuCount)
	if userCount != 3 || roleCount != 3 || menuCount != 26 {
		t.Fatalf("seed counts: user=%d role=%d menu=%d (want 3/3/26)", userCount, roleCount, menuCount)
	}
	if roleMenuCount != 52 { // super(1)+admin(2) 各挂 26 菜单
		t.Fatalf("role_menu count=%d (want 52)", roleMenuCount)
	}

	// 2. Login 三用户
	svc := sysSvc.UserService{}
	for _, u := range []struct{ name, pw string }{
		{"super", "123456"}, {"admin", "123456"}, {"test1", "123456"},
	} {
		tok, rft, _, err := svc.Login(u.name, u.pw)
		if err != nil || tok == "" || rft == "" {
			t.Fatalf("login %s: %v tok=%q rft=%q", u.name, err, tok, rft)
		}
	}
	if _, _, _, err := svc.Login("admin", "wrong"); err == nil {
		t.Fatal("wrong password should fail")
	}

	// 3. GetUserInfo 聚合三分支
	_, _, superKeys, superPerms, err := svc.GetUserInfo(101)
	if err != nil || len(superPerms) != 1 || superPerms[0] != "*:*:*" || len(superKeys) != 1 || superKeys[0] != "superadmin" {
		t.Fatalf("super getUserInfo: keys=%v perms=%v err=%v", superKeys, superPerms, err)
	}
	_, _, _, adminPerms, err := svc.GetUserInfo(102)
	if err != nil || len(adminPerms) == 0 {
		t.Fatalf("admin perms should be non-empty: %v err=%v", adminPerms, err)
	}
	_, _, _, test1Perms, err := svc.GetUserInfo(103)
	if err != nil || len(test1Perms) != 0 {
		t.Fatalf("test1 perms should be empty: %v err=%v", test1Perms, err)
	}

	// 4. casbin 兜底：admin(role2) 命中 /system/user/list；test1(role3) 被拒
	e := utils.GetCasbin()
	if ok, _ := e.Enforce(strconv.Itoa(2), "/system/user/list", "GET"); !ok {
		t.Fatal("admin should pass casbin for /system/user/list")
	}
	if ok, _ := e.Enforce(strconv.Itoa(3), "/system/user/list", "GET"); ok {
		t.Fatal("test1 should be denied by casbin")
	}
}
