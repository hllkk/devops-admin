package utils

import (
	"testing"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
)

// newMemEnforcer 用 casbinModelText 构造一个内存 enforcer(默认 adapter,不依赖 DB),
// 供 matcher 行为测试。验证启用 CasbinHandler 后 Enforce 的判定与 model 文本一致。
func newMemEnforcer(t *testing.T) *casbin.SyncedCachedEnforcer {
	t.Helper()
	m, err := model.NewModelFromString(casbinModelText)
	if err != nil {
		t.Fatalf("加载 model 失败: %v", err)
	}
	e, err := casbin.NewSyncedCachedEnforcer(m)
	if err != nil {
		t.Fatalf("创建内存 enforcer 失败: %v", err)
	}
	return e
}

// TestCasbinMatcher 验证 casbinModelText 的 matcher 行为:
//   - act="*" 放行所有 HTTP 方法(菜单级策略)
//   - keyMatch2 路径通配:/system/user/* 匹配子路径;裸路径 /system/user 需单独 policy
//   - 未授权资源/角色拒绝
func TestCasbinMatcher(t *testing.T) {
	e := newMemEnforcer(t)
	// 角色 1 授权 user 资源:裸路径 + 子路径通配,act=* (对齐菜单 api_prefix 填法)
	if _, err := e.AddPolicies([][]string{
		{"1", "/system/user", "*"},
		{"1", "/system/user/*", "*"},
	}); err != nil {
		t.Fatalf("AddPolicies 失败: %v", err)
	}

	cases := []struct {
		name   string
		sub    string
		obj    string
		act    string
		expect bool
	}{
		{"裸路径 POST(新增)", "1", "/system/user", "POST", true},
		{"裸路径 PUT(修改)", "1", "/system/user", "PUT", true},
		{"子路径 GET(列表)", "1", "/system/user/list", "GET", true},
		{"子路径带ID(详情)", "1", "/system/user/123", "GET", true},
		{"批量删除多ID", "1", "/system/user/1,2,3", "DELETE", true},
		{"act=* 覆盖导出 POST", "1", "/system/user/export", "POST", true},
		{"通配不误伤其他资源", "1", "/system/role/list", "GET", false},
		{"其他角色无策略", "2", "/system/user/list", "GET", false},
	}
	for _, c := range cases {
		ok, err := e.Enforce(c.sub, c.obj, c.act)
		if err != nil {
			t.Errorf("%s: Enforce 出错: %v", c.name, err)
			continue
		}
		if ok != c.expect {
			t.Errorf("%s: Enforce(%s,%s,%s)=%v, 期望 %v", c.name, c.sub, c.obj, c.act, ok, c.expect)
		}
	}
}

// TestCasbinMatcherActExact 验证 act 精确匹配:按钮级策略只放行指定方法,
// 非指定方法被拒(用于操作级精细控制)。
func TestCasbinMatcherActExact(t *testing.T) {
	e := newMemEnforcer(t)
	if _, err := e.AddPolicy("1", "/system/user/export", "POST"); err != nil {
		t.Fatalf("AddPolicy 失败: %v", err)
	}
	if ok, _ := e.Enforce("1", "/system/user/export", "POST"); !ok {
		t.Error("已授权 export POST,应放行")
	}
	if ok, _ := e.Enforce("1", "/system/user/export", "GET"); ok {
		t.Error("未授权 export GET,应拒绝")
	}
}

// TestCasbinMatcherBarePathNotMatchedWildcard 验证 keyMatch2 的边界:
// 仅声明 /system/user/* 时,裸路径 /system/user 不被匹配 —— 这正是菜单 api_prefix
// 必须同时声明 /system/user 与 /system/user/* 的原因(POST/PUT 走裸路径)。
func TestCasbinMatcherBarePathNotMatchedWildcard(t *testing.T) {
	e := newMemEnforcer(t)
	if _, err := e.AddPolicy("1", "/system/user/*", "*"); err != nil {
		t.Fatalf("AddPolicy 失败: %v", err)
	}
	if ok, _ := e.Enforce("1", "/system/user", "POST"); ok {
		t.Error("仅声明 /system/user/* 时裸路径不应被匹配,本断言守护 api_prefix 填写约定")
	}
	if ok, _ := e.Enforce("1", "/system/user/list", "GET"); !ok {
		t.Error("子路径应被 /system/user/* 匹配")
	}
}
