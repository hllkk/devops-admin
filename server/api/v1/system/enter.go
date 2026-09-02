package system

import "github.com/hllkk/devops-admin/server/service"

type ApiGroup struct {
	DBApi
	BaseApi
	DictApi
	PostApi
	DeptApi
	MenuApi
	RouteApi
	RoleApi
	UserApi
	LoginLogApi
	OperLogApi
	NoticeApi
	SettingApi
	SysErrorApi
	TimedTaskApi
	SocialApi
	WecomAuthApi
	WecomContactApi
	OnlineApi
}

var (
	initDBService         = service.ServiceGroupApp.SystemServiceGroup.InitDBService
	securityConfigService = service.ServiceGroupApp.SystemServiceGroup.SecurityConfigService
	ldapConfigService     = service.ServiceGroupApp.SystemServiceGroup.LdapConfigService
	notifyConfigService   = service.ServiceGroupApp.SystemServiceGroup.NotifyConfigService
	notifySendService     = service.ServiceGroupApp.SystemServiceGroup.NotifySendService
	userService           = service.ServiceGroupApp.SystemServiceGroup.UserService
	loginLogService       = service.ServiceGroupApp.SystemServiceGroup.LoginLogService
	captchaService        = service.ServiceGroupApp.SystemServiceGroup.CaptchaService
	dictTypeService       = service.ServiceGroupApp.SystemServiceGroup.DictTypeService
	dictDataService       = service.ServiceGroupApp.SystemServiceGroup.DictDataService
	postService           = service.ServiceGroupApp.SystemServiceGroup.PostService
	departmentService     = service.ServiceGroupApp.SystemServiceGroup.DepartmentService
	menuService           = service.ServiceGroupApp.SystemServiceGroup.MenuService
	routeService          = service.ServiceGroupApp.SystemServiceGroup.RouteService
	roleService           = service.ServiceGroupApp.SystemServiceGroup.RoleService
	operLogService        = service.ServiceGroupApp.SystemServiceGroup.SysOperLogService
	noticeService         = service.ServiceGroupApp.SystemServiceGroup.NoticeService
	settingService        = service.ServiceGroupApp.SystemServiceGroup.SettingService
	sysErrorService       = service.ServiceGroupApp.SystemServiceGroup.SysErrorService
	timedTaskService      = service.ServiceGroupApp.SystemServiceGroup.TimedTaskService
	socialService         = service.ServiceGroupApp.SystemServiceGroup.SocialService
	wecomContactService   = service.ServiceGroupApp.SystemServiceGroup.WecomContactService
	onlineService         = service.ServiceGroupApp.SystemServiceGroup.OnlineService
)
