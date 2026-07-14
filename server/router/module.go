package router

import "github.com/gin-gonic/gin"

// ModuleRouter 模块路由注册接口
// 每个业务模块实现此接口，将路由注册按认证级别分为三个方法。
//
// 使用方式：
//
//	modules := []ModuleRouter{
//	    &RouterGroupApp.System,
//	    // ...其他模块
//	}
//	for _, m := range modules {
//	    m.RegisterPublic(publicGroup)
//	    m.RegisterPrivate(privateGroup)
//	    m.RegisterAdmin(adminGroup)
//	}
type ModuleRouter interface {
	// RegisterPublic 注册公开路由（无需认证）
	RegisterPublic(r *gin.RouterGroup)
	// RegisterPrivate 注册需认证路由（JWT）
	RegisterPrivate(r *gin.RouterGroup)
	// RegisterAdmin 注册管理员路由（JWT + RequireAdmin）
	RegisterAdmin(r *gin.RouterGroup)
}
