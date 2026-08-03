package service

import (
	"github.com/hllkk/devops-admin/server/service/disk"
	"github.com/hllkk/devops-admin/server/service/system"
)

var ServiceGroupApp = new(ServiceGroup)

type ServiceGroup struct {
	SystemServiceGroup system.ServiceGroup
	DiskServiceGroup   disk.ServiceGroup
}
