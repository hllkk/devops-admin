package v1

import (
	"github.com/hllkk/devops-admin/server/api/v1/disk"
	"github.com/hllkk/devops-admin/server/api/v1/system"
)

var ApiGroupApp = new(ApiGroup)

type ApiGroup struct {
	SystemApiGroup system.ApiGroup
	DiskApiGroup   disk.ApiGroup
}
