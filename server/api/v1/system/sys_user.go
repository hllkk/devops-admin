package system

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"github.com/hllkk/devops-admin/server/utils"
)

// Login
// @Tags     Base
// @Summary  用户登录
// @Produce   application/json
// @Param    data  body      systemReq.Login  true  "用户名, 密码, 验证码"
// @Success  200   {object}  response.Response{data=object,msg=string}  "返回 token, refreshToken"
// @Router   /auth/login [post]
func (b *BaseApi) Login(c *gin.Context) {
	var l systemReq.Login
	if err := c.ShouldBindJSON(&l); err != nil {
		response.FailWithMessage("参数校验不通过", c)
		return
	}
	if err := utils.Verify(l, utils.LoginVerify); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	// 本期验证码关闭（/auth/code 返回 captchaEnabled=false），故不校验 captcha
	token, refreshToken, _, err := userService.Login(l.Username, l.Password)
	if err != nil {
		global.OPS_LOG.Warn("登录失败", zap.String("user", l.Username), zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	// 对齐前端 Api.Auth.LoginToken{token, refreshToken}
	response.OkWithDetailed(gin.H{
		"token":        token,
		"refreshToken": refreshToken,
	}, "登录成功", c)
}

// GetUserInfo
// @Tags      Base
// @Summary   获取用户信息
// @Security  ApiKeyAuth
// @Produce   application/json
// @Success   200  {object}  response.Response{data=object,msg=string}  "返回 user, roles, permissions"
// @Router    /auth/getUserInfo [get]
func (b *BaseApi) GetUserInfo(c *gin.Context) {
	claims, err := utils.GetClaims(c)
	if err != nil || claims == nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}
	userId := int64(claims.BaseClaims.ID) // BaseClaims.ID 与 jwt.RegisteredClaims.ID 同名，显式消歧
	user, roles, roleKeys, perms, err := userService.GetUserInfo(userId)
	if err != nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}
	// 对齐前端 Api.Auth.UserInfo{user, roles, permissions}：
	// user 为完整 Api.System.User & {roles: Role[]}（SysUser.Password 因 json:"-" 不泄露）
	type userInfoVO struct {
		system.SysUser
		Roles []system.SysRole `json:"roles"`
	}
	response.OkWithDetailed(gin.H{
		"user":        userInfoVO{SysUser: user, Roles: roles},
		"roles":       roleKeys,
		"permissions": perms,
	}, "获取成功", c)
}
