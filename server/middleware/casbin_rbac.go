package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/utils"
)

// rbacWhitelistPrivate 已登录用户必需的"自己操作自己"/基础接口前缀(不走 casbin 策略校验)。
// 这些接口与角色无关、只涉及自身数据或登录链路必需,放行避免登录后基础功能不可用
// (如拿不到路由 / 改不了自己密码 / 看不到未读通知)。新增"所有登录用户都可访问"的接口时,在此登记。
// 注意:精确到具体 path,不要用管理资源前缀(如 /system/notice),以免误放行 list/增删改等管理接口。
var rbacWhitelistPrivate = []string{
	"/auth",                 // getUserInfo/logout/refreshToken: 登录链路必需
	"/route",                // getUserRoutes/isRouteExist: 路由下发必需
	"/user/getUserInfo",     // 自身信息查询
	"/system/user/profile",  // 自助改密/改资料/头像(自己操作自己)
	"/monitor/online",       // 个人在线设备视图(仅当前用户自己)
	"/system/notice/unread",  // 当前用户未读通知列表(顶栏小红点/通知中心,所有用户必需)
	"/system/notice/read",    // 标记通知已读(个人操作)
	"/system/dict/data/type", // 按字典类型查字典数据(DictTag/DictRadio 公共组件渲染,任意页面可用,只读基础数据)
	// AI 身份自身数据(home「我的AI身份」页,所有登录用户可用;数据范围由 JWT 锁定:
	// identity/my 取 utils.GetUserID 仅查本人主Key明文,dashboard 非超管强制 scope=self)。
	// 管理操作(建删改 Key/scenario/POST aggregate)不在此列,仍走 casbin 菜单授权。
	"/gateway/ai-key/identity/my",             // 我的 AI 身份(未开通返回 opened=false)
	"/gateway/ai-key/identity/available-models", // 可授权模型列表(只读公共数据)
		"/gateway/model/active", // 用户侧可见模型列表(模型广场/home,按发布可见性过滤)
	"/gateway/dashboard/overview",             // 我的用量总览
	"/gateway/dashboard/trend",                // 我的成本趋势
	"/gateway/dashboard/top",                  // 我的成本Top
	"/gateway/dashboard/budget",               // 我的预算执行率
}

// isRbacWhitelisted 判断去前缀后的接口路径是否命中已登录用户白名单。
func isRbacWhitelisted(obj string) bool {
	for _, p := range rbacWhitelistPrivate {
		if strings.HasPrefix(obj, p) {
			return true
		}
	}
	return false
}

// CasbinHandler 拦截器
func CasbinHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		waitUse, _ := utils.GetClaims(c)
		// 超管经 SuperAdmin 标志直接放行,绕过策略校验
		// (见 source/system/sys_role_menu.go: super 虽有 SuperAdmin 标志可绕过权限检查)
		if waitUse != nil && waitUse.SuperAdmin {
			c.Next()
			return
		}
		//获取请求的PATH
		path := c.Request.URL.Path
		obj := strings.TrimPrefix(path, global.OPS_CONFIG.System.RouterPrefix)
		// 已登录用户必需的自身接口(登录链路/个人中心)直接放行,不进 casbin 策略
		if isRbacWhitelisted(obj) {
			c.Next()
			return
		}
		// 获取请求方法
		act := c.Request.Method
		// 获取用户的角色
		sub := strconv.Itoa(int(waitUse.RoleId))
		e := utils.GetCasbin() // 判断策略中是否存在
		success, _ := e.Enforce(sub, obj, act)
		if !success {
			response.FailWithDetailed(gin.H{}, "权限不足", c)
			c.Abort()
			return
		}
		c.Next()
	}
}
