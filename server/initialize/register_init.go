package initialize

import (
	_ "github.com/hllkk/devops-admin/server/source/gateway"
	_ "github.com/hllkk/devops-admin/server/source/system"
)

func init() {
	// do nothing,only import source package so that inits can be registered
}
