package system

import "github.com/hllkk/devops-admin/server/service"

type ApiGroup struct {
	DBApi
	BaseApi
	LoginLogApi
}

var (
	initDBService   = service.ServiceGroupApp.SystemServiceGroup.InitDBService
	userService     = service.ServiceGroupApp.SystemServiceGroup.UserService
	captchaService  = service.ServiceGroupApp.SystemServiceGroup.CaptchaService
	loginLogService = service.ServiceGroupApp.SystemServiceGroup.LoginLogService
)
