package disk

// ServiceGroup 网盘服务聚合入口(在 service/enter.go 注册为 DiskServiceGroup)。
type ServiceGroup struct {
	DiskFileService
	DiskUploadService
}
