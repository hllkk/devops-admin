package system

import v1 "github.com/hllkk/devops-admin/server/api/v1"

type RouterGroup struct {
	InitRouter
	BaseRouter
	UserRouter
	AuthRouter
	RouteRouter
	DictRouter
	PostRouter
	DeptRouter
	MenuRouter
	RoleRouter
	LoginLogRouter
	OperLogRouter
	NoticeRouter
	SettingRouter
	SysErrorRouter
}

var (
	baseApi     = v1.ApiGroupApp.SystemApiGroup.BaseApi
	dbApi       = v1.ApiGroupApp.SystemApiGroup.DBApi
	dictApi     = v1.ApiGroupApp.SystemApiGroup.DictApi
	postApi     = v1.ApiGroupApp.SystemApiGroup.PostApi
	deptApi     = v1.ApiGroupApp.SystemApiGroup.DeptApi
	menuApi     = v1.ApiGroupApp.SystemApiGroup.MenuApi
	roleApi     = v1.ApiGroupApp.SystemApiGroup.RoleApi
	userApi     = v1.ApiGroupApp.SystemApiGroup.UserApi
	loginLogApi = v1.ApiGroupApp.SystemApiGroup.LoginLogApi
	operLogApi  = v1.ApiGroupApp.SystemApiGroup.OperLogApi
	noticeApi   = v1.ApiGroupApp.SystemApiGroup.NoticeApi
	settingApi  = v1.ApiGroupApp.SystemApiGroup.SettingApi
	sysErrorApi = v1.ApiGroupApp.SystemApiGroup.SysErrorApi
)
