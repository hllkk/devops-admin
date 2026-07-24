package response

import (
	"github.com/hllkk/devops-admin/server/model/system"
)

// LoginResponse 登录响应(对齐前端 LoginToken: 前端主要消费 expiresAt)
//
// token 不进响应体,只通过 httpOnly Cookie 下发(前端 getAuthorization 返回 null,
// 浏览器自动携带);NeedChangePassword 用于密码过期强制改密。
type LoginResponse struct {
	ExpiresAt          int64 `json:"expiresAt"`                    // access token 过期毫秒时间戳
	NeedChangePassword bool  `json:"needChangePassword,omitempty"` // 密码过期需强制修改
}

// SysUserResponse 用户信息响应
type SysUserResponse struct {
	User system.SysUser `json:"user"`
}

// UserInfoResponse /auth/getUserInfo 响应(对齐前端 Api.Auth.UserInfo)
//   - User 含 roles 对象数组(SysUser.Roles)
//   - Roles 为 roleKey 字符串数组(前端超管判定/路由过滤用)
//   - Permissions 为 perms 字符串数组(按钮权限,超管 *:*:*)
type UserInfoResponse struct {
	User          system.SysUser `json:"user"`
	Roles         []string       `json:"roles"`
	Permissions   []string       `json:"permissions"`
	DefaultRouter string         `json:"defaultRouter"` // 默认路由(主角色 DefaultRouter;登录入口)
}
