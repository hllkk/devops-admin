package system

import "github.com/hllkk/devops-admin/server/service"

type ApiGroup struct {
	DBApi
	CaptchaApi
}

var (
	initDBService  = service.ServiceGroupApp.SystemServiceGroup.InitDBService
	captchaService = service.ServiceGroupApp.SystemServiceGroup.CaptchaService
)
