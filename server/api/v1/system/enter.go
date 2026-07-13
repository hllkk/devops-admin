package system

import "github.com/hllkk/devops-admin/server/service"

type ApiGroup struct {
	DBApi
	BaseApi
}

var (
	initDBService = service.ServiceGroupApp.SystemServiceGroup.InitDBService
)
