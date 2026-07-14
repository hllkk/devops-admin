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

const logoutTokenCode = "8888"

// Login
// @Tags     Base
// @Summary  用户登录
// @Produce   application/json
// @Param    data  body      systemReq.Login  true  "用户名, 密码"
// @Success  200   {object}  response.Response{data=object,msg=string}  "access/refresh 写入 httpOnly cookie，返回 expiresAt(毫秒)"
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
	// 验证码先于密码校验（防绕过暴力破解）；按触发策略决定是否必须
	ip := c.ClientIP()
	if captchaService.NeedCaptcha(l.Username, ip) {
		if err := captchaService.VerifyCaptcha(l.CaptchaId, l.Captcha); err != nil {
			captchaService.RecordLoginResult(l.Username, ip, false)
			response.FailWithMessage(err.Error(), c)
			return
		}
	}
	access, refresh, _, err := userService.Login(l.Username, l.Password)
	if err != nil {
		captchaService.RecordLoginResult(l.Username, ip, false)
		global.OPS_LOG.Warn("登录失败", zap.String("user", l.Username), zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	captchaService.RecordLoginResult(l.Username, ip, true)
	utils.SetLoginCookies(c, access, refresh)
	j := utils.NewJWT()
	claims, _ := j.ParseAccessToken(access)
	expiresAt := int64(0)
	if claims != nil && claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Unix() * 1000
	}
	response.OkWithDetailed(gin.H{"expiresAt": expiresAt}, "登录成功", c)
}

// RefreshToken
// @Tags     Base
// @Summary  刷新令牌
// @Produce   application/json
// @Success  200  {object}  response.Response{data=object,msg=string}  "用 refresh-token cookie 换发新 access/refresh，返回 expiresAt(毫秒)；失败 code=8888"
// @Router   /auth/refreshToken [post]
func (b *BaseApi) RefreshToken(c *gin.Context) {
	refresh, err := c.Cookie("refresh-token")
	if err != nil || refresh == "" {
		response.NoAuthWithCode(logoutTokenCode, "refresh token 不存在，请重新登录", c)
		return
	}
	j := utils.NewJWT()
	claims, err := j.ParseRefreshToken(refresh)
	if err != nil || utils.IsBlacklisted(refresh) {
		response.NoAuthWithCode(logoutTokenCode, "refresh token 已失效，请重新登录", c)
		return
	}
	bc := claims.BaseClaims
	access, err := j.CreateAccessToken(bc)
	if err != nil {
		response.NoAuthWithCode(logoutTokenCode, "令牌刷新失败，请重新登录", c)
		return
	}
	newRefresh, err := j.CreateRefreshToken(bc)
	if err != nil {
		response.NoAuthWithCode(logoutTokenCode, "令牌刷新失败，请重新登录", c)
		return
	}
	utils.JoinBlacklist(refresh)
	if oldAccess, e := utils.GetToken(c); e == nil && oldAccess != "" {
		utils.JoinBlacklist(oldAccess)
	}
	utils.SetLoginCookies(c, access, newRefresh)
	expiresAt := int64(0)
	if ac, perr := j.ParseAccessToken(access); perr == nil && ac != nil && ac.ExpiresAt != nil {
		expiresAt = ac.ExpiresAt.Unix() * 1000
	}
	response.OkWithDetailed(gin.H{"expiresAt": expiresAt}, "刷新成功", c)
}

// Logout
// @Tags     Base
// @Summary  退出登录
// @Produce   application/json
// @Success  200  {object}  response.Response  "清除登录 cookie，当前 token 入黑名单"
// @Router   /auth/logout [post]
func (b *BaseApi) Logout(c *gin.Context) {
	utils.ClearLoginCookies(c)
	if token, err := utils.GetToken(c); err == nil && token != "" {
		utils.JoinBlacklist(token)
	}
	response.OkWithMessage("退出成功", c)
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
