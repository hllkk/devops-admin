package system

import "github.com/hllkk/devops-admin/server/service"

type ApiGroup struct {
	DBApi
	BaseApi
}

var (
	initDBService         = service.ServiceGroupApp.SystemServiceGroup.InitDBService
	securityConfigService = service.ServiceGroupApp.SystemServiceGroup.SecurityConfigService
	userService           = service.ServiceGroupApp.SystemServiceGroup.UserService
	loginLogService       = service.ServiceGroupApp.SystemServiceGroup.LoginLogService
	captchaService        = service.ServiceGroupApp.SystemServiceGroup.CaptchaService
)
