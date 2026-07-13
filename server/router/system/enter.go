package system

import api "github.com/hllkk/devops-admin/server/api/v1"

type RouterGroup struct {
	InitRouter
	BaseRouter
}

var (
	dbApi   = api.ApiGroupApp.SystemApiGroup.DBApi
	baseApi = api.ApiGroupApp.SystemApiGroup.BaseApi
)
