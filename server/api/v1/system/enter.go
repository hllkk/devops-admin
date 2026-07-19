package system

import "github.com/hllkk/devops-admin/server/service"

type ApiGroup struct {
	DBApi
	BaseApi
	DictApi
	PostApi
	DeptApi
	MenuApi
	RoleApi
	UserApi
}

var (
	initDBService         = service.ServiceGroupApp.SystemServiceGroup.InitDBService
	securityConfigService = service.ServiceGroupApp.SystemServiceGroup.SecurityConfigService
	userService           = service.ServiceGroupApp.SystemServiceGroup.UserService
	loginLogService       = service.ServiceGroupApp.SystemServiceGroup.LoginLogService
	captchaService        = service.ServiceGroupApp.SystemServiceGroup.CaptchaService
	dictTypeService       = service.ServiceGroupApp.SystemServiceGroup.DictTypeService
	dictDataService       = service.ServiceGroupApp.SystemServiceGroup.DictDataService
	postService           = service.ServiceGroupApp.SystemServiceGroup.PostService
	departmentService     = service.ServiceGroupApp.SystemServiceGroup.DepartmentService
	menuService           = service.ServiceGroupApp.SystemServiceGroup.MenuService
	roleService           = service.ServiceGroupApp.SystemServiceGroup.RoleService
)
