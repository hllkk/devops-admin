package system

import v1 "github.com/hllkk/devops-admin/server/api/v1"

type RouterGroup struct {
	InitRouter
	CaptchaRouter
}

var (
	dbApi      = v1.ApiGroupApp.SystemApiGroup.DBApi
	captchaApi = v1.ApiGroupApp.SystemApiGroup.CaptchaApi
)
