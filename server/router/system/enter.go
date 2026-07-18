package system

import v1 "github.com/hllkk/devops-admin/server/api/v1"

type RouterGroup struct {
	InitRouter
	BaseRouter
	UserRouter
	AuthRouter
	RouteRouter
}

var (
	baseApi = v1.ApiGroupApp.SystemApiGroup.BaseApi
	dbApi   = v1.ApiGroupApp.SystemApiGroup.DBApi
)
