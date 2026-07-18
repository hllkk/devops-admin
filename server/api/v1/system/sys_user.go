package system

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/common/response"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	systemRes "github.com/hllkk/devops-admin/server/model/system/response"
	systemSvc "github.com/hllkk/devops-admin/server/service/system"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
	"github.com/redis/go-redis/v9"
)

// Login
// @Tags     Base
// @Summary  用户登录
// @Produce   application/json
// @Param    data  body      systemReq.Login                                             true  "用户名, 密码, 验证码"
// @Success  200   {object}  response.Response{data=systemRes.LoginResponse,msg=string}  "返回包括用户信息,token,过期时间"
// @Router   /base/login [post]
func (b *BaseApi) Login(c *gin.Context) {
	var l systemReq.Login
	err := c.ShouldBindJSON(&l)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(l, utils.LoginVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	cfg := securityConfigService.Current(c.Request.Context())

	// 1. 账号锁定检查
	if cfg.LockEnable && systemSvc.IsAccountLocked(c.Request.Context(), l.Username) {
		response.FailWithMessage("账号已锁定，请 "+strconv.Itoa(cfg.LockDuration)+" 分钟后再试", c)
		loginLogService.CreateLoginLog(c.Request.Context(), system.SysLoginLog{
			Username: l.Username, Ip: c.ClientIP(), Agent: c.Request.UserAgent(),
			Status: false, ErrorMessage: "账号已锁定",
		})
		return
	}

	// 2. 验证码检查（按 IP 计数）
	key := c.ClientIP()
	openCaptcha := cfg.CaptchaOpen
	openCaptchaTimeOut := cfg.CaptchaTimeout
	v, ok := global.OPS_CACHE.Get(key)
	if !ok {
		global.OPS_CACHE.Set(key, int64(1), time.Second*time.Duration(openCaptchaTimeOut))
	}
	var oc = openCaptcha == 0 || openCaptcha < interfaceToInt(v)
	if oc && (l.Captcha == "" || l.CaptchaId == "" || !store.Verify(l.CaptchaId, l.Captcha, true)) {
		global.OPS_CACHE.Increment(key, 1)
		response.FailWithMessage("验证码错误", c)
		loginLogService.CreateLoginLog(c.Request.Context(), system.SysLoginLog{
			Username: l.Username, Ip: c.ClientIP(), Agent: c.Request.UserAgent(),
			Status: false, ErrorMessage: "验证码错误",
		})
		return
	}

	// 3. 凭证校验
	u := &system.SysUser{Username: l.Username, Password: l.Password}
	user, err := userService.Login(c.Request.Context(), u)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("登陆失败! 用户名不存在或者密码错误!")
		global.OPS_CACHE.Increment(key, 1)
		systemSvc.RecordLoginFail(c.Request.Context(), l.Username, cfg)
		response.FailWithMessage("用户名不存在或者密码错误", c)
		loginLogService.CreateLoginLog(c.Request.Context(), system.SysLoginLog{
			Username: l.Username, Ip: c.ClientIP(), Agent: c.Request.UserAgent(),
			Status: false, ErrorMessage: "用户名不存在或者密码错误",
		})
		return
	}
	if user.Enable != 1 {
		logger.WithCtx(c.Request.Context()).Mod("biz").Error("登陆失败! 用户被禁止登录!")
		global.OPS_CACHE.Increment(key, 1)
		response.FailWithMessage("用户被禁止登录", c)
		loginLogService.CreateLoginLog(c.Request.Context(), system.SysLoginLog{
			Username: l.Username, Ip: c.ClientIP(), Agent: c.Request.UserAgent(),
			Status: false, ErrorMessage: "用户被禁止登录", UserID: user.ID,
		})
		return
	}

	// 4. 登录成功 清除失败计数与锁
	systemSvc.ClearLoginFail(c.Request.Context(), l.Username)

	// 5. 密码过期检查
	needChange := systemSvc.IsPasswordExpired(c.Request.Context(), user.PasswordUpdatedAt, cfg, time.Now())
	b.TokenNext(c, *user, needChange)
}

// TokenNext 登录以后签发 jwt
func (b *BaseApi) TokenNext(c *gin.Context, user system.SysUser, mustChangePwd bool) {
	token, claims, err := utils.LoginTokenWithExpire(&user, mustChangePwd)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取token失败!")
		response.FailWithMessage("获取token失败", c)
		return
	}
	// 记录登录成功日志
	loginLogService.CreateLoginLog(c.Request.Context(), system.SysLoginLog{
		Username: user.Username, Ip: c.ClientIP(), Agent: c.Request.UserAgent(),
		Status: true, UserID: user.ID, ErrorMessage: "登录成功",
	})
	if !global.OPS_CONFIG.System.UseMultipoint {
		utils.SetToken(c, token, int(claims.RegisteredClaims.ExpiresAt.Unix()-time.Now().Unix()))
		response.OkWithDetailed(systemRes.LoginResponse{
			User:               user,
			Token:              token,
			ExpiresAt:          claims.RegisteredClaims.ExpiresAt.Unix() * 1000,
			NeedChangePassword: mustChangePwd,
		}, "登录成功", c)
		return
	}

	if jwtStr, err := jwtService.GetRedisJWT(c.Request.Context(), user.Username); err == redis.Nil {
		if err := utils.SetRedisJWT(token, user.Username); err != nil {
			logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("设置登录状态失败!")
			response.FailWithMessage("设置登录状态失败", c)
			return
		}
		utils.SetToken(c, token, int(claims.RegisteredClaims.ExpiresAt.Unix()-time.Now().Unix()))
		response.OkWithDetailed(systemRes.LoginResponse{
			User: user, Token: token,
			ExpiresAt:          claims.RegisteredClaims.ExpiresAt.Unix() * 1000,
			NeedChangePassword: mustChangePwd,
		}, "登录成功", c)
	} else if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("设置登录状态失败!")
		response.FailWithMessage("设置登录状态失败", c)
	} else {
		var blackJWT system.JwtBlacklist
		blackJWT.Jwt = jwtStr
		if err := jwtService.JsonInBlacklist(c.Request.Context(), blackJWT); err != nil {
			response.FailWithMessage("jwt作废失败", c)
			return
		}
		if err := utils.SetRedisJWT(token, user.GetUsername()); err != nil {
			response.FailWithMessage("设置登录状态失败", c)
			return
		}
		utils.SetToken(c, token, int(claims.RegisteredClaims.ExpiresAt.Unix()-time.Now().Unix()))
		response.OkWithDetailed(systemRes.LoginResponse{
			User: user, Token: token,
			ExpiresAt:          claims.RegisteredClaims.ExpiresAt.Unix() * 1000,
			NeedChangePassword: mustChangePwd,
		}, "登录成功", c)
	}
}

// Register
// @Tags     SysUser
// @Summary  用户注册账号
// @Produce   application/json
// @Param    data  body      systemReq.Register                                            true  "用户名, 昵称, 密码, 角色ID"
// @Success  200   {object}  response.Response{data=systemRes.SysUserResponse,msg=string}  "用户注册账号,返回包括用户信息"
// @Router   /user/admin_register [post]
func (b *BaseApi) Register(c *gin.Context) {
	var r systemReq.Register
	err := c.ShouldBindJSON(&r)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(r, utils.RegisterVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err = utils.ValidatePasswordComplexity(r.Password, securityConfigService.Current(c.Request.Context())); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	var authorities []system.SysAuthority
	for _, v := range r.AuthorityIds {
		authorities = append(authorities, system.SysAuthority{
			AuthorityId: v,
		})
	}
	user := &system.SysUser{Username: r.Username, NickName: r.NickName, Password: r.Password, HeaderImg: r.HeaderImg, AuthorityId: r.AuthorityId, Authorities: authorities, Enable: r.Enable, Phone: r.Phone, Email: r.Email}
	userReturn, err := userService.Register(c.Request.Context(), *user)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("注册失败!")
		response.FailWithDetailed(systemRes.SysUserResponse{User: userReturn}, "注册失败", c)
		return
	}
	response.OkWithDetailed(systemRes.SysUserResponse{User: userReturn}, "注册成功", c)
}

// GetUserList
// @Tags      SysUser
// @Summary   分页获取用户列表
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.GetUserList                                        true  "页码, 每页大小"
// @Success   200   {object}  response.Response{data=response.PageResult,msg=string}  "分页获取用户列表,返回包括列表,总数,页码,每页数量"
// @Router    /user/getUserList [post]
func (b *BaseApi) GetUserList(c *gin.Context) {
	var pageInfo systemReq.GetUserList
	err := c.ShouldBindJSON(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(pageInfo, utils.PageInfoVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := userService.GetUserInfoList(c.Request.Context(), pageInfo)
	if err != nil {
		logger.WithCtx(c.Request.Context()).Mod("biz").Err(err).Error("获取失败!")
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     pageInfo.Page,
		PageSize: pageInfo.PageSize,
	}, "获取成功", c)
}
