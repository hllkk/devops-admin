package server

import "github.com/hllkk/devops-admin/server/service"

// ApiGroup 服务器模块 API 组(挂在 PrivateGroup，对齐前端 /server/* 资源)。
type ApiGroup struct {
	AssetApi
	CredentialApi
}

var (
	assetService      = service.ServiceGroupApp.ServerServiceGroup.AssetService
	credentialService = service.ServiceGroupApp.ServerServiceGroup.CredentialService
)
