package service

import (
	"github.com/hllkk/devops-admin/server/service/gateway"
	"github.com/hllkk/devops-admin/server/service/system"
)

var ServiceGroupApp = new(ServiceGroup)

type ServiceGroup struct {
	SystemServiceGroup  system.ServiceGroup
	GatewayServiceGroup gateway.ServiceGroup
}
