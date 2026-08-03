package disk

import v1 "github.com/hllkk/devops-admin/server/api/v1"

// RouterGroup 网盘路由聚合入口(在 router/enter.go 注册为 Disk)。
type RouterGroup struct {
	DiskFileRouter
}

var diskFileApi = v1.ApiGroupApp.DiskApiGroup.DiskFileApi
