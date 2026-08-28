package middleware

import "testing"

// TestIsRbacWhitelisted 验证白名单前缀匹配:
//   - 登录链路/个人接口/只读基础数据接口放行
//   - 不误伤同前缀的管理接口(如 notice/list、dict/data/list 不被放行)
// 守护"个人/基础接口精确白名单、管理接口仍走 casbin"的边界。
func TestIsRbacWhitelisted(t *testing.T) {
	cases := []struct {
		obj    string
		expect bool
	}{
		// 登录链路 / 个人接口:放行
		{"/auth/getUserInfo", true},
		{"/auth/logout", true},
		{"/route/getUserRoutes", true},
		{"/user/getUserInfo", true},
		{"/system/user/profile", true},
		{"/system/user/profile/updatePwd", true},
		{"/monitor/online", true},
		{"/system/notice/unread", true},
		{"/system/notice/read", true},
		{"/system/dict/data/type/sys_user_sex", true},

		// AI 身份自身数据(home「我的AI身份」页,所有登录用户):放行
		{"/gateway/ai-key/identity/my", true},
		{"/gateway/ai-key/identity/available-models", true},
		{"/gateway/model/active", true},
		{"/gateway/dashboard/overview", true},
		{"/gateway/dashboard/trend", true},
		{"/gateway/dashboard/top", true},
		{"/gateway/dashboard/budget", true},

		// 同前缀的管理接口:不得被误放行
		{"/system/notice/list", false},
		{"/system/dict/data/list", false},
		{"/system/dict/data", false},      // POST 新增 / PUT 修改字典数据
		{"/system/dict/type/list", false}, // 字典类型管理
		{"/system/user/list", false},      // 用户管理
		{"/system/role/list", false},      // 角色管理
		{"/system/menu/list", false},
		{"/gateway/ai-key/list", false},          // 密钥管理列表
		{"/gateway/ai-key/1", false},             // 密钥详情/改/删
		{"/gateway/ai-key/scenario/list", false}, // 场景管理
		{"/gateway/model/list", false},           // 模型管理列表
		{"/gateway/model/1", false},              // 模型详情/改/删
		{"/gateway/model/publish", false},        // 发布设置(管理操作)
		{"/gateway/dashboard/aggregate", false}, // 手动触发聚合(管理操作)
	}
	for _, c := range cases {
		got := isRbacWhitelisted(c.obj)
		if got != c.expect {
			t.Errorf("isRbacWhitelisted(%q)=%v, 期望 %v", c.obj, got, c.expect)
		}
	}
}
