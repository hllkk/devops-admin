package request

import (
	"github.com/hllkk/devops-admin/server/model/common"
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
	RoleId      int64   `json:"roleId,string" form:"roleId"` // 主角色 ID
	RoleIds     []int64 `json:"roleIds" form:"roleIds"`      // 多角色
	DeptId      int64   `json:"deptId,string" form:"deptId"` // 主部门 ID
	Email       string  `json:"email" form:"email"`
	Phonenumber string  `json:"phonenumber" form:"phonenumber"`
}

// UserSearch 用户分页查询(对齐前端 Api.System.UserSearchParams,GET query 传输)。
// deptId/userName/nickName/phonenumber 过滤;roleId>0 时 join sys_user_role。
type UserSearch struct {
	commonReq.PageInfo
	DeptId      int64  `json:"deptId,string" form:"deptId"`    // 主部门ID(精确)
	UserName    string `json:"userName" form:"userName"`       // 用户名(模糊)
	NickName    string `json:"nickName" form:"nickName"`       // 昵称(模糊)
	Phonenumber string `json:"phonenumber" form:"phonenumber"` // 手机号(模糊)
	Status      string `json:"status" form:"status"`           // 状态(精确 '0'正常/'1'停用)
	RoleId      int64  `json:"roleId,string" form:"roleId"`    // 角色ID(精确,join sys_user_role)
}

// UserOperateParams 用户新增/修改请求(对齐前端 Api.System.UserOperateParams)。
// create 时 userId 为空、password 必填;update 时 userId 必填、password 空=不改。
// roleIds/postIds 用 []common.Int64String 兼容前端 IdType[](string 混合)。
type UserOperateParams struct {
	UserId      common.Int64String   `json:"userId"`      // 用户ID(新增时为空)
	DeptId      common.Int64String   `json:"deptId"`      // 主部门ID
	UserName    string               `json:"userName"`    // 用户名(create 必填,唯一)
	NickName    string               `json:"nickName"`    // 昵称
	Email       string               `json:"email"`       // 邮箱
	Phonenumber string               `json:"phonenumber"` // 手机号
	Sex         string               `json:"sex"`         // 性别 0男1女2未知
	Password    string               `json:"password"`    // 密码(create 必填;update 空=不改)
	Status      string               `json:"status"`      // 状态
	Remark      string               `json:"remark"`      // 备注
	RoleIds     []common.Int64String `json:"roleIds"`     // 角色ID列表(全量替换)
	PostIds     []common.Int64String `json:"postIds"`     // 岗位ID列表(全量替换)
}

// ResetUserPwdParams 重置用户密码(对齐前端 PUT /system/user/resetPwd,{userId,password})。
// 项目无加密中间件,password 明文传输(传输层由网关/TLS 保护),后端 bcrypt 存储。
type ResetUserPwdParams struct {
	UserId   common.Int64String `json:"userId"`   // 用户ID
	Password string             `json:"password"` // 新密码(明文)
}

// ChangeMyPasswordParams 当前用户自助修改密码(对齐前端 PUT /system/user/profile/updatePwd)。
// oldPassword 二次确认身份(防会话劫持后改密);复杂度由 service 校验。明文传输,后端 bcrypt 存储。
type ChangeMyPasswordParams struct {
	OldPassword string `json:"oldPassword" binding:"required"` // 旧密码(明文,二次校验)
	NewPassword string `json:"newPassword" binding:"required"` // 新密码(明文,复杂度由 service 校验)
}

// UpdateMyProfileParams 当前用户自助修改基本资料(对齐前端 PUT /system/user/profile,UserProfileOperateParams)。
// 仅允许改 nickName/email/phonenumber/sex;userName/角色/部门/状态等敏感字段不在自助范围,走管理员侧接口。
type UpdateMyProfileParams struct {
	NickName    string `json:"nickName"`    // 昵称
	Email       string `json:"email"`       // 邮箱
	Phonenumber string `json:"phonenumber"` // 手机号
	Sex         string `json:"sex"`         // 性别 0男1女2未知
}
