package disk

import "github.com/hllkk/devops-admin/server/service"

// ApiGroup 网盘 API 聚合入口(在 api/v1/enter.go 注册为 DiskApiGroup)。
type ApiGroup struct {
	DiskFileApi
}

var diskFileService = service.ServiceGroupApp.DiskServiceGroup.DiskFileService
var diskUploadService = service.ServiceGroupApp.DiskServiceGroup.DiskUploadService
