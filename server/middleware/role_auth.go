package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/model/common/response"
	SysRequest "github.com/hllkk/devops-admin/server/model/system/request"
)

// 角色码常量：统一各处散落的硬编码
const (
	RoleSuper = "SUPER" // 超级管理员
	RoleAdmin = "ADMIN" // 管理员
)

// isAdminRole 判断角色码是否属于管理员（超管或管理员）
func isAdminRole(roleCode string) bool {
	return roleCode == RoleSuper || roleCode == RoleAdmin
}

// RequireAdmin 要求当前登录用户为管理员（SUPER/ADMIN），否则返回权限不足。
// 用于保护纯管理类路由（用户/角色/菜单/API/系统设置等）。
// 背景：Casbin 细粒度授权已注释禁用，此处作为角色码白名单兜底，
// 防止已登录的普通用户调用管理接口造成越权。
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleCode := roleCodeFromContext(c)
		if roleCode == "" {
			response.FailWithDetailed(gin.H{}, "登录状态异常，请重新登录", c)
			c.Abort()
			return
		}
		if !isAdminRole(roleCode) {
			response.FailWithDetailed(gin.H{}, "权限不足，需要管理员权限", c)
			c.Abort()
			return
		}
		c.Next()
	}
}

// roleCodeFromContext 从 gin.Context 的 claims 中提取角色码。
// 兼容 *CustomClaims 与 map[string]interface{} 两种 claims 存储形式。
func roleCodeFromContext(c *gin.Context) string {
	claims, exists := c.Get("claims")
	if !exists {
		return ""
	}
	switch v := claims.(type) {
	case *SysRequest.CustomClaims:
		return v.BaseClaims.RoleCode
	case map[string]interface{}:
		if rc, ok := v["roleCode"].(string); ok {
			return rc
		}
	}
	return ""
}
