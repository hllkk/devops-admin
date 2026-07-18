package request

import (
	commonReq "github.com/hllkk/devops-admin/server/model/common/request"
)

// Login 登录请求(对齐前端 PwdLoginForm: username/password/captchaId/captcha)
type Login struct {
	Username  string `json:"username" form:"username"`   // 用户名
	Password  string `json:"password" form:"password"`   // 密码(明文,传输层由网关/加密中间件保护)
	CaptchaId string `json:"captchaId" form:"captchaId"` // 验证码会话 ID
	Captcha   string `json:"captcha" form:"captcha"`     // 验证码答案(go-captcha 为 JSON 字符串)
}

// Register 注册请求(管理员建号;字段对齐 SysUser)
type Register struct {
	Username    string  `json:"username" form:"username"`
	NickName    string  `json:"nickName" form:"nickName"`
	Password    string  `json:"password" form:"password"`
	RoleId      int64   `json:"roleId,string" form:"roleId"`       // 主角色 ID
	RoleIds     []int64 `json:"roleIds" form:"roleIds"`             // 多角色
	DeptId      int64   `json:"deptId,string" form:"deptId"`        // 主部门 ID
	Email       string  `json:"email" form:"email"`
	Phonenumber string  `json:"phonenumber" form:"phonenumber"`
}

// GetUserList 用户分页查询(内嵌通用分页 + 过滤)
type GetUserList struct {
	commonReq.PageInfo
	UserName string `json:"userName" form:"userName"`
	DeptId   int64  `json:"deptId,string" form:"deptId"`
}
