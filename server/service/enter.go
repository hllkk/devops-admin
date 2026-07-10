package service

import (
	// "github.com/hllkk/devops-admin/server/service/example"
	"github.com/hllkk/devops-admin/server/service/system"
)

var ServiceGroupApp = new(ServiceGroup)

type ServiceGroup struct {
	SystemServiceGroup system.ServiceGroup
	// ExampleServiceGroup example.ServiceGroup
}
