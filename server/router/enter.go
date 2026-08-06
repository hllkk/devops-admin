package router

import (
	"github.com/hllkk/devops-admin/server/router/gateway"
	"github.com/hllkk/devops-admin/server/router/media"
	"github.com/hllkk/devops-admin/server/router/system"
)

var RouterGroupApp = new(RouterGroup)

type RouterGroup struct {
	System  system.RouterGroup
	Media   media.RouterGroup
	Gateway gateway.RouterGroup
}
